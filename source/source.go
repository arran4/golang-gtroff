// Package source loads bounded, non-executing roff source sets from an explicit
// filesystem. Each file retains its own immutable syntax snapshot. Includes are
// references, not macro execution; the logical line sequence retains directives.
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/internal/sourceio"
)

// ID identifies one source occurrence in a Set. Repeated includes have distinct
// IDs, even when they refer to the same file. NodeIDs remain local to each AST.
type ID uint64

// EdgeID identifies an include occurrence in a Set.
type EdgeID uint64

var (
	ErrLimit  = errors.New("source set resource limit exceeded")
	ErrStrict = errors.New("source set rejected diagnostics")
)

// Options configures only this load. FS is an explicit allowed source root for
// includes in Read; Load uses its fsys argument. An injected FS must enforce its
// own symlink containment (os.DirFS alone does not). OpenDirectory supplies an
// os.Root-backed FS. Paths are slash-separated and relative to the referring
// file's directory; .mso uses the same policy, without a system macro search.
// Nonpositive limits select defaults. Limits are cumulative across occurrences.
type Options struct {
	// Registry is an optional explicit token/block registry. Nil uses built-ins
	// only, isolating loads from process-global registrations. Configure it before
	// loading; each AST snapshots registrations, while handler objects remain shared.
	Registry                               *groff.Registry `json:"-"`
	FS                                     fs.FS           `json:"-"`
	Package                                string
	NoIncludes, Strict                     bool
	MaxBytes, MaxNodes, MaxFiles, MaxDepth int
}

// File is one immutable syntax snapshot and its include provenance.
type File struct {
	id     ID
	name   string
	tree   *groff.AST
	parent EdgeID
	depth  int
}

func (f *File) ID() ID             { return f.id }
func (f *File) Name() string       { return f.name }
func (f *File) AST() *groff.AST    { return f.tree }
func (f *File) ParentEdge() EdgeID { return f.parent }
func (f *File) Depth() int         { return f.depth }

// Edge retains an include directive even when its target cannot be loaded.
// State is resolved, disabled, unresolved, invalid, missing, denied, cycle,
// depth-limit, resource-limit or incomplete. To is nonzero for captured children,
// including partial children. Span and Node always belong to From's original AST.
type Edge struct {
	ID      EdgeID       `json:"id"`
	From    ID           `json:"from"`
	Node    groff.NodeID `json:"node"`
	Span    groff.Span   `json:"span"`
	Request string       `json:"request"`
	Target  string       `json:"target"`
	To      ID           `json:"to,omitempty"`
	State   string       `json:"state"`
}

// Line identifies a physical syntax line in logical source order. An include
// directive appears before its child lines. It is never replaced or executed.
type Line struct {
	Source ID           `json:"source"`
	Node   groff.NodeID `json:"node"`
	Depth  int          `json:"depth"`
}

// Diagnostic adds occurrence identity and a stable code to a parser diagnostic.
// Codes for parser findings are syntax; include codes match Edge.State.
type Diagnostic struct {
	Source ID     `json:"source"`
	Code   string `json:"code"`
	groff.Diagnostic
}

// IncludeError locates a failed include. It is returned in strict mode; its
// underlying fs/path error remains inspectable with errors.Is/As.
type IncludeError struct {
	Edge Edge
	Err  error
}

func (e *IncludeError) Error() string {
	return fmt.Sprintf("%s:%d:%d: include %q: %v", e.Edge.Span.Start.Filename, e.Edge.Span.Start.Line, e.Edge.Span.Start.Column, e.Edge.Target, e.Err)
}
func (e *IncludeError) Unwrap() error { return e.Err }

// Set retains files, references and logical order separately. Accessors return
// copies of slices; original ASTs are immutable. IDs are deterministic for equal
// filesystem contents/options, not persistent across edits. No ambient FS is used.
type Set struct {
	files       []*File
	edges       []Edge
	lines       []Line
	diagnostics []Diagnostic
}

func (s *Set) Files() []*File            { return append([]*File(nil), s.files...) }
func (s *Set) Edges() []Edge             { return append([]Edge(nil), s.edges...) }
func (s *Set) Lines() []Line             { return append([]Line(nil), s.lines...) }
func (s *Set) Diagnostics() []Diagnostic { return append([]Diagnostic(nil), s.diagnostics...) }

// File looks up a source occurrence in O(1); zero and out-of-range IDs are absent.
func (s *Set) File(id ID) *File {
	if id == 0 || id > ID(len(s.files)) {
		return nil
	}
	return s.files[id-1]
}

// Node looks up a file-local AST identity; absent source/node IDs return nil.
func (s *Set) Node(line Line) *groff.ASTNode {
	f := s.File(line.Source)
	if f == nil {
		return nil
	}
	return f.tree.Node(line.Node)
}

// MarshalJSON exports schema version 1, including each captured source and AST.
// JSON replaces invalid UTF-8 according to encoding/json; AST.Source retains the
// exact bytes. This report is not a deserializable source snapshot or roff writer.
func (s *Set) MarshalJSON() ([]byte, error) {
	type file struct {
		ID       ID         `json:"id"`
		Name     string     `json:"name"`
		Parent   EdgeID     `json:"parentEdge,omitempty"`
		Depth    int        `json:"depth"`
		Complete bool       `json:"complete"`
		Source   string     `json:"source"`
		Syntax   *groff.AST `json:"syntax"`
	}
	files := make([]file, 0, len(s.files))
	for _, f := range s.files {
		files = append(files, file{f.id, f.name, f.parent, f.depth, f.tree.SourceComplete(), f.tree.Source(), f.tree})
	}
	return json.Marshal(struct {
		SchemaVersion int          `json:"schemaVersion"`
		Files         []file       `json:"files"`
		Edges         []Edge       `json:"edges"`
		Lines         []Line       `json:"lines"`
		Diagnostics   []Diagnostic `json:"diagnostics,omitempty"`
	}{1, files, s.Edges(), s.Lines(), s.Diagnostics()})
}

// OpenDirectory explicitly opens an allowed OS source root. The caller owns and
// must close the returned root after loading. Root.FS confines symlink traversal;
// mount points and hard links inside an otherwise trusted root remain trusted.
func OpenDirectory(directory string) (*os.Root, error) { return os.OpenRoot(directory) }

// Load reads a named root file from an injected FS, closing every opened file.
// A missing root or input/resource/cancellation failure always returns an error.
// Broken includes preserve their directive and produce diagnostics; Strict also
// returns an error for any diagnostics. NoIncludes disables loading, not recognition.
// A non-nil Set is returned even on failure, containing all captured evidence.
func Load(ctx context.Context, fsys fs.FS, name string, o Options) (*Set, error) {
	o.FS = fsys
	l := newLoader(ctx, o)
	if !validName(name) {
		return l.set, fmt.Errorf("root source %q: %w", name, fs.ErrInvalid)
	}
	l.open(name, 0, 0)
	return l.finish()
}

// Read parses a borrowed root reader without closing it. With no explicit FS in
// Options, includes stay unresolved and never consult the host filesystem. name
// labels positions; include resolution requires it to be a valid relative FS path.
// Cancellation is checked around reads and between lines/files; a caller's blocked
// Read or FS.Open must implement its own I/O deadlines. Parsing CPU work is bounded
// by the byte/node limits and checked for cancellation before and after parsing.
func Read(ctx context.Context, r io.Reader, name string, o Options) (*Set, error) {
	l := newLoader(ctx, o)
	if sourceio.IsNil(r) {
		return l.set, fmt.Errorf("read source %q: nil reader", name)
	}
	l.visit(r, nil, name, 0, 0, nil)
	return l.finish()
}

type activeFile struct {
	name string
	info fs.FileInfo
}
type loader struct {
	ctx           context.Context
	o             Options
	set           *Set
	active        []activeFile
	bytes, nodes  int
	terminal      error
	includeErrors []error
}

func newLoader(ctx context.Context, o Options) *loader {
	if ctx == nil {
		ctx = context.Background()
	}
	if o.Registry == nil {
		o.Registry = groff.NewRegistry()
	}
	if o.Package == "" {
		o.Package = "man"
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = 8 << 20
	}
	if o.MaxNodes <= 0 {
		o.MaxNodes = 100000
	}
	if o.MaxFiles <= 0 {
		o.MaxFiles = 256
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = 16
	}
	if o.MaxDepth > 128 {
		o.MaxDepth = 128
	}
	return &loader{ctx: ctx, o: o, set: &Set{}}
}
func (l *loader) finish() (*Set, error) {
	// Occurrence IDs follow first traversal; diagnostics sort by source occurrence,
	// original byte position, then stable insertion order, independently of recursion.
	sort.SliceStable(l.set.diagnostics, func(i, j int) bool {
		a, b := l.set.diagnostics[i], l.set.diagnostics[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Position.Offset < b.Position.Offset
	})
	if l.terminal != nil {
		return l.set, l.terminal
	}
	if l.o.Strict && len(l.set.diagnostics) > 0 {
		return l.set, errors.Join(append([]error{fmt.Errorf("%w: %s", ErrStrict, l.set.diagnostics[0].Message)}, l.includeErrors...)...)
	}
	return l.set, nil
}
func (l *loader) budget() error {
	if err := l.ctx.Err(); err != nil {
		return err
	}
	if len(l.set.files) >= l.o.MaxFiles || l.bytes >= l.o.MaxBytes || l.nodes+5 >= l.o.MaxNodes {
		return ErrLimit
	}
	return nil
}
func (l *loader) open(name string, parent EdgeID, depth int) ID {
	if err := l.budget(); err != nil {
		l.terminal = errors.Join(l.terminal, err)
		if parent != 0 {
			l.edgeFailure(parent, "resource-limit", err)
		}
		return 0
	}
	if l.o.FS == nil {
		l.terminal = fmt.Errorf("open root %q: no filesystem configured", name)
		return 0
	}
	f, err := l.o.FS.Open(name)
	if err != nil {
		l.openFailure(parent, name, err)
		return 0
	}
	if sourceio.IsNil(f) {
		l.openFailure(parent, name, fmt.Errorf("filesystem returned a nil file"))
		return 0
	}
	info, err := f.Stat()
	if err == nil && (info == nil || !info.Mode().IsRegular()) {
		err = fmt.Errorf("%w: source is not a regular file", fs.ErrInvalid)
	}
	if err != nil {
		l.openFailure(parent, name, err)
		if closeErr := f.Close(); closeErr != nil {
			l.terminal = errors.Join(l.terminal, closeErr)
		}
		return 0
	}
	for _, a := range l.active {
		if name == a.name || a.info != nil && os.SameFile(a.info, info) {
			closeErr := f.Close()
			if parent != 0 {
				l.edgeFailure(parent, "cycle", fmt.Errorf("active source alias cycle"))
			} else {
				l.terminal = fmt.Errorf("root source cycle")
			}
			if closeErr != nil {
				l.terminal = errors.Join(l.terminal, closeErr)
			}
			return 0
		}
	}
	return l.visit(f, f, name, parent, depth, info)
}
func (l *loader) openFailure(parent EdgeID, name string, err error) {
	if parent == 0 {
		l.terminal = fmt.Errorf("open root %q: %w", name, err)
		return
	}
	state := "denied"
	if errors.Is(err, fs.ErrNotExist) {
		state = "missing"
	}
	l.edgeFailure(parent, state, err)
}
func (l *loader) visit(r io.Reader, closer io.Closer, name string, parent EdgeID, depth int, info fs.FileInfo) ID {
	tree, err := groff.ParseAST(contextReader{l.ctx, r}, groff.WithPackage(l.o.Package), groff.WithRegistry(l.o.Registry), groff.WithSourceName(name), groff.WithASTBytes(l.o.MaxBytes-l.bytes), groff.WithASTNodes(l.o.MaxNodes-l.nodes))
	if closer != nil {
		err = errors.Join(err, closer.Close())
	}
	if ctxErr := l.ctx.Err(); ctxErr != nil {
		err = errors.Join(err, ctxErr)
	}
	if tree == nil {
		l.terminal = errors.Join(l.terminal, err)
		return 0
	}
	var sourceLimit *groff.ASTSourceLimitError
	if errors.As(err, &sourceLimit) {
		err = errors.Join(ErrLimit, err)
	}
	l.terminal = errors.Join(l.terminal, err)
	id := ID(len(l.set.files) + 1)
	l.set.files = append(l.set.files, &File{id, name, tree, parent, depth})
	l.bytes += len(tree.Source())
	l.nodes += len(tree.Root().Descendants()) + 1
	for _, d := range tree.Diagnostics() {
		l.set.diagnostics = append(l.set.diagnostics, Diagnostic{id, "syntax", d})
	}
	l.active = append(l.active, activeFile{name, info})
	defer func() { l.active = l.active[:len(l.active)-1] }()
	for _, n := range tree.Root().FindKind(groff.ASTLine) {
		if l.terminal == nil {
			l.terminal = l.ctx.Err()
		}
		l.set.lines = append(l.set.lines, Line{id, n.ID(), depth})
		a := n.Attributes()
		if a.Opaque || a.Name != ".so" && a.Name != ".mso" {
			continue
		}
		edgeID := EdgeID(len(l.set.edges) + 1)
		target := ""
		if len(a.Arguments) > 0 {
			target = a.Arguments[0].Value
		}
		l.set.edges = append(l.set.edges, Edge{ID: edgeID, From: id, Node: n.ID(), Span: n.Span(), Request: a.Name, Target: target})
		switch {
		case l.o.NoIncludes:
			l.edgeFailure(edgeID, "disabled", fmt.Errorf("include loading explicitly disabled"))
		case l.terminal != nil:
			l.edgeFailure(edgeID, "incomplete", fmt.Errorf("include not attempted after input/resource failure"))
		case len(a.Arguments) != 1 || target == "" || !utf8.ValidString(target) || strings.ContainsAny(target, "\\:\x00\r\n"):
			l.edgeFailure(edgeID, "invalid", fmt.Errorf("include requires one literal local path"))
		case l.o.FS == nil:
			l.edgeFailure(edgeID, "unresolved", fmt.Errorf("no explicit include filesystem context"))
		case !validName(name):
			l.edgeFailure(edgeID, "unresolved", fmt.Errorf("root source label is not a relative filesystem path"))
		default:
			joined := path.Join(path.Dir(name), target)
			if path.IsAbs(target) || !validName(joined) {
				l.edgeFailure(edgeID, "denied", fs.ErrPermission)
				continue
			}
			// Target retains the literal argument. Resolved child Name records its FS path.
			cycle := false
			for _, a := range l.active {
				if a.name == joined {
					cycle = true
					break
				}
			}
			if cycle {
				l.edgeFailure(edgeID, "cycle", fmt.Errorf("active source cycle"))
				continue
			}
			if depth >= l.o.MaxDepth {
				l.edgeFailure(edgeID, "depth-limit", ErrLimit)
				l.terminal = errors.Join(l.terminal, ErrLimit)
				continue
			}
			if err := l.budget(); err != nil {
				l.edgeFailure(edgeID, "resource-limit", err)
				l.terminal = errors.Join(l.terminal, err)
				continue
			}
			child := l.open(joined, edgeID, depth+1)
			edge := &l.set.edges[edgeID-1]
			edge.To = child
			if child != 0 {
				edge.State = "resolved"
				if l.terminal != nil {
					l.edgeFailure(edgeID, "incomplete", l.terminal)
				}
			}
		}
	}
	return id
}
func (l *loader) edgeFailure(id EdgeID, state string, err error) {
	e := &l.set.edges[id-1]
	e.State = state
	l.includeErrors = append(l.includeErrors, &IncludeError{*e, err})
	l.set.diagnostics = append(l.set.diagnostics, Diagnostic{e.From, state, groff.Diagnostic{Message: "include " + state + ": " + err.Error(), Position: e.Span.Start, Severity: "warning"}})
}
func validName(s string) bool {
	return s != "." && fs.ValidPath(s) && !strings.ContainsAny(s, "\\:\x00")
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
