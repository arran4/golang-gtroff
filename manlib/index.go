// Package manlib builds a bounded immutable man-page library from explicit
// filesystems and serves it without reading request-selected paths or resources.
package manlib

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

var ErrLimit = errors.New("man library resource limit exceeded")

// Root is an explicit trusted fs.FS boundary. OpenDirectory supplies an OS-backed
// root that prevents symlink escape. Arbitrary FS implementations must provide
// the same confinement contract; directories must implement fs.ReadDirFile.
type Root struct {
	Label string
	FS    fs.FS
}
type Limits struct {
	MaxEntries, MaxScanEntries, MaxDepth, MaxFileBytes, MaxTotalBytes, MaxNodes int
}

// ValidateLimits checks index budgets without opening a filesystem.
func ValidateLimits(l Limits) error { _, err := limits(l); return err }

func limits(l Limits) (Limits, error) {
	pairs := []struct {
		p        *int
		def, max int
	}{{&l.MaxEntries, 10000, 50000}, {&l.MaxScanEntries, 100000, 500000}, {&l.MaxDepth, 12, 64}, {&l.MaxFileBytes, 2 << 20, 8 << 20}, {&l.MaxTotalBytes, 64 << 20, 256 << 20}, {&l.MaxNodes, 20000, 100000}}
	for _, p := range pairs {
		if *p.p == 0 {
			*p.p = p.def
		}
		if *p.p < 1 || *p.p > p.max {
			return l, fmt.Errorf("library limits must be positive and within documented ceilings")
		}
	}
	return l, nil
}

type Notice struct{ Root, Path, Message string }
type Entry struct {
	Name, Section, Description, Root, Path, URL string
	Diagnostics                                 int
}
type indexedPage struct {
	entry  Entry
	source string
	folded string
}

// Index is a source/search snapshot. Editing or removing source files after
// Build does not change request results; rebuild explicitly for fresh content.
type Index struct {
	pages                []indexedPage
	lookup               map[string][]int
	notices              []Notice
	bytes                int
	nodeLimit, fileLimit int
}

func (i *Index) Entries() []Entry {
	if i == nil {
		return nil
	}
	out := make([]Entry, len(i.pages))
	for n, p := range i.pages {
		out[n] = p.entry
	}
	return out
}
func (i *Index) Notices() []Notice {
	if i == nil {
		return nil
	}
	return append([]Notice(nil), i.notices...)
}
func pageKey(name, section string) string { return name + "\x00" + section }
func pageURL(name, section, root string) string {
	return "/page/" + url.PathEscape(section) + "/" + url.PathEscape(name) + "?root=" + url.QueryEscape(root)
}

var sectionPattern = regexp.MustCompile(`^(?:[0-9][0-9A-Za-z]{0,15}|[nl])$`)

func validName(s string) bool {
	if s == "" || s == "." || s == ".." || len(s) > 512 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || strings.ContainsRune("._+-@", r)) {
			return false
		}
	}
	return true
}
func splitPage(file string) (string, string, bool) {
	file = strings.TrimSuffix(file, ".gz")
	at := strings.LastIndexByte(file, '.')
	if at <= 0 {
		return "", "", false
	}
	name, section := file[:at], file[at+1:]
	return name, section, validName(name) && sectionPattern.MatchString(section)
}

// Build scans deterministically and closes every owned file before returning.
// Budget/cancellation failures return the partial snapshot plus an error; callers
// should not serve a partial index unless they explicitly choose to do so.
func Build(ctx context.Context, roots []Root, l Limits) (*Index, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil build context")
	}
	l, err := limits(l)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 || len(roots) > 32 {
		return nil, fmt.Errorf("configure 1..32 explicit roots")
	}
	labels := map[string]bool{}
	for _, r := range roots {
		if r.FS == nil || r.Label == "" || len(r.Label) > 64 || strings.ContainsAny(r.Label, "/\\\x00\r\n") || !utf8.ValidString(r.Label) || labels[r.Label] {
			return nil, fmt.Errorf("roots require unique valid labels and explicit filesystems")
		}
		labels[r.Label] = true
	}
	idx := &Index{lookup: map[string][]int{}, nodeLimit: l.MaxNodes, fileLimit: l.MaxFileBytes}
	defer idx.finalize()
	scan := 0
	for _, r := range roots {
		var walk func(string, int) error
		walk = func(dir string, depth int) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if depth > l.MaxDepth {
				return ErrLimit
			}
			entries, err := readDirectory(ctx, r.FS, dir, &scan, l.MaxScanEntries)
			if err != nil {
				return err
			}
			for _, de := range entries {
				if err := ctx.Err(); err != nil {
					return err
				}
				name := de.Name()
				if name == "." || name == ".." || strings.ContainsAny(name, "/\\") || len(name) > 512 {
					return fmt.Errorf("filesystem returned invalid directory entry")
				}
				file := path.Join(dir, name)
				if de.Type()&fs.ModeSymlink != 0 {
					idx.notice(r.Label, file, "symlink skipped")
					continue
				}
				if de.IsDir() {
					if strings.HasPrefix(name, ".") {
						continue
					}
					if err := walk(file, depth+1); err != nil {
						return err
					}
					continue
				}
				pageName, section, ok := splitPage(name)
				if !ok {
					continue
				}
				info, err := de.Info()
				if err != nil {
					idx.notice(r.Label, file, "file metadata unavailable; skipped")
					continue
				}
				if !info.Mode().IsRegular() {
					idx.notice(r.Label, file, "nonregular file skipped")
					continue
				}
				if len(idx.pages) >= l.MaxEntries {
					return ErrLimit
				}
				data, cost, err := readPage(ctx, r.FS, file, l.MaxFileBytes)
				if err != nil {
					if errors.Is(err, ErrLimit) || ctx.Err() != nil {
						return err
					}
					idx.notice(r.Label, file, "unreadable or invalid compressed page; skipped")
					continue
				}
				if cost > l.MaxTotalBytes-idx.bytes {
					return ErrLimit
				}
				idx.bytes += cost
				sourceName := r.Label + ":" + file
				tree, err := groff.ParseAST(bytes.NewReader(data), groff.WithPackage("man"), groff.WithSourceName(sourceName), groff.WithASTBytes(l.MaxFileBytes), groff.WithASTNodes(l.MaxNodes))
				if err != nil {
					return fmt.Errorf("%s: bounded syntax parse: %w", sourceName, ErrLimit)
				}
				doc, err := man.Build(tree, man.Options{MaxBytes: l.MaxFileBytes, MaxNodes: l.MaxNodes})
				if err != nil {
					return fmt.Errorf("%s: bounded semantic parse: %w", sourceName, ErrLimit)
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				text := doc.Text(1, man.TextOptions{Normalize: true})
				e := Entry{Name: pageName, Section: section, Description: description(doc), Root: r.Label, Path: file, Diagnostics: len(doc.Diagnostics)}
				e.URL = pageURL(e.Name, e.Section, e.Root)
				idx.pages = append(idx.pages, indexedPage{entry: e, source: doc.Source(), folded: strings.ToLower(e.Name + "\n" + e.Section + "\n" + e.Description + "\n" + text)})
			}
			return nil
		}
		if err := walk(".", 0); err != nil {
			return idx, err
		}
	}
	return idx, nil
}

func (idx *Index) finalize() {
	order := map[string]int{}
	for _, p := range idx.pages {
		if _, ok := order[p.entry.Root]; !ok {
			order[p.entry.Root] = len(order)
		}
	}
	sort.SliceStable(idx.pages, func(a, b int) bool {
		x, y := idx.pages[a].entry, idx.pages[b].entry
		if x.Name != y.Name {
			return x.Name < y.Name
		}
		if x.Section != y.Section {
			return x.Section < y.Section
		}
		if x.Root != y.Root {
			return order[x.Root] < order[y.Root]
		}
		return x.Path < y.Path
	})
	for n, p := range idx.pages {
		idx.lookup[pageKey(p.entry.Name, p.entry.Section)] = append(idx.lookup[pageKey(p.entry.Name, p.entry.Section)], n)
	}
}

func (i *Index) notice(root, file, message string) {
	if len(i.notices) < 256 {
		i.notices = append(i.notices, Notice{root, file, message})
	} else if len(i.notices) == 256 {
		i.notices = append(i.notices, Notice{Message: "additional collection notices suppressed"})
	}
}
func readDirectory(ctx context.Context, sys fs.FS, name string, count *int, maxEntries int) ([]fs.DirEntry, error) {
	f, err := sys.Open(name)
	if err != nil {
		return nil, err
	}
	rd, ok := f.(fs.ReadDirFile)
	if !ok {
		_ = f.Close()
		return nil, fmt.Errorf("filesystem directory lacks streaming ReadDir")
	}
	var all []fs.DirEntry
	for {
		if err = ctx.Err(); err != nil {
			break
		}
		var batch []fs.DirEntry
		batch, err = rd.ReadDir(64)
		*count += len(batch)
		if *count > maxEntries {
			err = ErrLimit
			break
		}
		all = append(all, batch...)
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			break
		}
		if len(batch) == 0 {
			err = fmt.Errorf("filesystem ReadDir made no progress")
			break
		}
	}
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name() < all[j].Name() })
	return all, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(b)
}
func readPage(ctx context.Context, sys fs.FS, name string, maxBytes int) ([]byte, int, error) {
	f, err := sys.Open(name)
	if err != nil {
		return nil, 0, err
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, 0, fmt.Errorf("page is not a readable regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(contextReader{ctx, f}, int64(maxBytes)+1))
	closeErr := f.Close()
	if err != nil {
		return nil, 0, err
	}
	if closeErr != nil {
		return nil, 0, closeErr
	}
	if len(raw) > maxBytes {
		return nil, 0, ErrLimit
	}
	if !strings.HasSuffix(name, ".gz") {
		return raw, len(raw), nil
	}
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(io.LimitReader(contextReader{ctx, z}, int64(maxBytes)+1))
	closeErr = z.Close()
	if err != nil {
		return nil, 0, err
	}
	if closeErr != nil {
		return nil, 0, closeErr
	}
	if len(data) > maxBytes {
		return nil, 0, ErrLimit
	}
	return data, len(raw) + len(data), nil
}
func description(d *man.Document) string {
	for _, n := range d.Nodes {
		if n.Kind != man.Section || !strings.EqualFold(strings.TrimSpace(n.Heading), "NAME") {
			continue
		}
		s := strings.TrimSpace(strings.TrimPrefix(d.Text(n.ID, man.TextOptions{Normalize: true}), n.Heading))
		for _, sep := range []string{" - ", " — ", " – "} {
			if _, tail, ok := strings.Cut(s, sep); ok {
				s = tail
				break
			}
		}
		r := []rune(s)
		if len(r) > 256 {
			s = string(r[:256]) + "…"
		}
		return s
	}
	return ""
}

// Document reparses the indexed source snapshot into caller-owned semantic data.
// It never reopens a filesystem path. Root empty selects the first configured
// collection for duplicate name/section identities.
func (i *Index) Document(ctx context.Context, name, section, root string) (*man.Document, error) {
	if i == nil || ctx == nil {
		return nil, fmt.Errorf("nil index/context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := i.find(name, section, root)
	if p == nil {
		return nil, fs.ErrNotExist
	}
	tree, err := groff.ParseAST(strings.NewReader(p.source), groff.WithPackage("man"), groff.WithSourceName(p.entry.Root+":"+p.entry.Path), groff.WithASTBytes(i.fileLimit), groff.WithASTNodes(i.nodeLimit))
	if err != nil {
		return nil, err
	}
	doc, err := man.Build(tree, man.Options{MaxBytes: i.fileLimit, MaxNodes: i.nodeLimit})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}
