package edit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/arran4/golang-gtroff/source"
)

// capturedFS is deliberately unable to open a file not captured in the initial
// snapshot. New/retargeted includes never acquire a filesystem capability.
type capturedFS map[string]string

func (f capturedFS) Open(name string) (fs.File, error) {
	s, ok := f[name]
	if !ok || !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &memoryFile{Reader: strings.NewReader(s), name: name, size: int64(len(s))}, nil
}

type memoryFile struct {
	*strings.Reader
	name string
	size int64
}

func (f *memoryFile) Close() error               { return nil }
func (f *memoryFile) Stat() (fs.FileInfo, error) { return memoryInfo{f.name, f.size}, nil }

type memoryInfo struct {
	name string
	size int64
}

func (i memoryInfo) Name() string       { return path.Base(i.name) }
func (i memoryInfo) Size() int64        { return i.size }
func (i memoryInfo) Mode() fs.FileMode  { return 0400 }
func (i memoryInfo) ModTime() time.Time { return time.Time{} }
func (i memoryInfo) IsDir() bool        { return false }
func (i memoryInfo) Sys() any           { return nil }

func (w *Working) reparse(ctx context.Context, next map[string]string, patches []Patch, m *budget) (*source.Set, error) {
	for _, s := range next {
		if err := m.spend(len(s)); err != nil {
			return nil, err
		}
	}
	o := w.o.Source
	o.FS = capturedFS(next)
	o.Strict = false
	set, err := source.Read(ctx, strings.NewReader(next[w.root]), w.root, o)
	// Non-strict Source.Read retains syntax diagnostics. Input/budget failures
	// remain errors; existing diagnostics may survive at mapped untouched offsets.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if errors.Is(err, source.ErrLimit) {
		return nil, fmt.Errorf("%w: %w", ErrLimit, err)
	}
	if set == nil {
		return nil, fmt.Errorf("%w: %w", ErrValidation, err)
	}
	for _, f := range set.Files() {
		if !f.AST().SourceComplete() {
			return nil, fmt.Errorf("%w: incomplete reparsed source", ErrValidation)
		}
	}
	for _, d := range set.Diagnostics() {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		if strings.Contains(d.Message, "budget exceeded") {
			return nil, fmt.Errorf("%w: %s", ErrLimit, d.Message)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidation, err)
	}

	locations := newOffsetMaps(patches)
	allowed := map[string]int{}
	for _, d := range w.set.Diagnostics() {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		f := w.set.File(d.Source)
		if f == nil {
			continue
		}
		offset, ok := locations[f.Name()].mapped(d.Position.Offset)
		if !ok {
			continue
		}
		key := diagnosticKey(f.Name(), offset, d.Code, d.Severity, d.Message)
		allowed[key]++
	}
	for _, d := range set.Diagnostics() {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		if strings.Contains(d.Message, "budget exceeded") {
			return nil, fmt.Errorf("%w: %s", ErrLimit, d.Message)
		}
		if w.o.Strict {
			return nil, fmt.Errorf("%w: %s", ErrValidation, d.Message)
		}
		f := set.File(d.Source)
		if f == nil {
			return nil, ErrValidation
		}
		key := diagnosticKey(f.Name(), d.Position.Offset, d.Code, d.Severity, d.Message)
		// Include failure messages vary by resolver. Compare unresolved edges
		// separately using exact original directive/target and mapped position.
		if d.Code != "syntax" || d.Severity != "error" {
			continue
		}
		if allowed[key] == 0 {
			return nil, fmt.Errorf("%w: %s:%d: %s", ErrValidation, f.Name(), d.Position.Line, d.Message)
		}
		allowed[key]--
	}
	oldEdges := map[string]int{}
	for _, e := range w.set.Edges() {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		if e.State == "resolved" || e.State == "disabled" {
			continue
		}
		f := w.set.File(e.From)
		if f == nil {
			continue
		}
		offset, ok := locations[f.Name()].mapped(e.Span.Start.Offset)
		if !ok {
			continue
		}
		oldEdges[fmt.Sprintf("%s:%d:%s:%s", f.Name(), offset, e.Request, e.Target)]++
	}
	for _, e := range set.Edges() {
		if err := m.spend(1); err != nil {
			return nil, err
		}
		if e.State == "resolved" || e.State == "disabled" {
			continue
		}
		f := set.File(e.From)
		key := fmt.Sprintf("%s:%d:%s:%s", f.Name(), e.Span.Start.Offset, e.Request, e.Target)
		if oldEdges[key] == 0 {
			return nil, fmt.Errorf("%w: new unresolved include in %q", ErrValidation, f.Name())
		}
		oldEdges[key]--
	}
	return set, nil
}
func diagnosticKey(name string, offset int, code, severity, message string) string {
	return fmt.Sprintf("%s:%d:%s:%s:%s", name, offset, code, severity, message)
}

type offsetMap struct {
	patches []Patch
	delta   []int
}

func newOffsetMaps(patches []Patch) map[string]offsetMap {
	result := map[string]offsetMap{}
	for _, p := range patches {
		entry := result[p.Source]
		entry.patches = append(entry.patches, p)
		previous := 0
		if len(entry.delta) > 0 {
			previous = entry.delta[len(entry.delta)-1]
		}
		entry.delta = append(entry.delta, previous+len(p.Data)-(p.End-p.Start))
		result[p.Source] = entry
	}
	return result
}
func (m offsetMap) mapped(offset int) (int, bool) {
	i := sort.Search(len(m.patches), func(i int) bool { return m.patches[i].End > offset })
	if i < len(m.patches) && m.patches[i].Start <= offset {
		return 0, false
	}
	delta := 0
	if i > 0 {
		delta = m.delta[i-1]
	}
	return offset + delta, true
}
