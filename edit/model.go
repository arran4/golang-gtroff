// Package edit applies source-backed transformations to an in-memory working
// document. Selectors stay stateless; plans belong to a particular working
// version. Filesystem access and writes are explicit host capabilities.
package edit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

var (
	ErrOptions      = errors.New("invalid edit options or command")
	ErrSelection    = errors.New("edit selection is missing or ambiguous")
	ErrTarget       = errors.New("invalid edit target or source owner")
	ErrStale        = errors.New("stale or foreign edit plan")
	ErrOverlap      = errors.New("overlapping source patches")
	ErrValidation   = errors.New("edit introduces invalid source")
	ErrNeedsRewrite = errors.New("description requires explicit canonical rewrite")
	ErrLimit        = errors.New("edit resource limit exceeded")
)

// Options bounds a working document. Source configures reparsing using ONLY
// captured files: its FS and Strict fields are ignored (use Options.Strict). Registry must remain unchanged while this
// working document is in use. Nonpositive budgets select defaults. Strict
// requires exactly one selected match after restrictions and rejects diagnostics.
type Options struct {
	Source                                     source.Options
	Strict, IncludeEdits                       bool
	MaxBytes, MaxPatches, MaxCommands, MaxWork int
}

// Restriction uses one-based, snapshot-scoped query indices. Zero means unset.
// At most one of Nth, inclusive From/To, FirstN, or All may be supplied. Strict
// mode still requires one result, including with explicit All.
type Restriction struct {
	Nth    int  `json:"nth,omitempty"`
	From   int  `json:"from,omitempty"`
	To     int  `json:"to,omitempty"`
	FirstN int  `json:"firstN,omitempty"`
	All    bool `json:"all,omitempty"`
}

type Operation string

const (
	ReplaceSource      Operation = "replace-source"
	FindReplaceSource  Operation = "find-replace-source"
	Delete             Operation = "delete"
	InsertBefore       Operation = "insert-before"
	InsertAfter        Operation = "insert-after"
	MoveBefore         Operation = "move-before"
	MoveAfter          Operation = "move-after"
	AppendParagraph    Operation = "append-paragraph"
	AppendSection      Operation = "append-section"
	AppendDefinition   Operation = "append-definition"
	ReplaceDescription Operation = "replace-description"
)

// Command is a typed transformation, not a query extension. Text/Find are exact
// bytes for source operations, literal UTF-8 data for named authoring operations.
// Destination is a unique selector for moves, evaluated in the same snapshot.
// Rewrite explicitly permits replacement of a complex description's source.
type Command struct {
	Select                    query.Selector
	View                      query.View
	Restrict                  Restriction
	Operation                 Operation
	Text, Find, Heading, Term string
	Destination               query.Selector
	Rewrite                   bool
}

// Root selects the working document's original root occurrence only.
func Root() query.Selector { return query.Select(query.Source).Where(query.SourceID, query.Equal, "1") }

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
	Match   int    `json:"match,omitempty"`
}

// Patch replaces [Start,End) in one physical captured file. Digest is SHA-256
// of the entire pre-patch source. Data is base64 in JSON, preserving invalid UTF-8.
// PlanPatches requires a digest even for insertions. Match is provenance only.
type Patch struct {
	Source string `json:"source"`
	Digest string `json:"digest"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Data   []byte `json:"data"`
	Match  int    `json:"match,omitempty"`
	Step   int    `json:"step"`
	// owner distinguishes separate physical insertion origins; not serialized.
	owner string
}

// Change compares a captured baseline with current or planned exact bytes.
// Before/After encode as base64 in JSON; human diffs quote bytes without loss.
type Change struct {
	Source       string `json:"source"`
	BeforeDigest string `json:"beforeDigest"`
	AfterDigest  string `json:"afterDigest"`
	Before       []byte `json:"before"`
	After        []byte `json:"after"`
}

// Plan is immutable. Its accessors return copies. A plan cannot be deserialized
// and executed: Apply checks owner and version against the live working object.
type Plan struct {
	owner       *byte
	version     uint64
	before      map[string]string
	after       map[string]string
	set         *source.Set
	patches     []Patch
	diagnostics []Diagnostic
}

func (p *Plan) Version() uint64           { return p.version }
func (p *Plan) Patches() []Patch          { return clonePatches(p.patches) }
func (p *Plan) Diagnostics() []Diagnostic { return append([]Diagnostic(nil), p.diagnostics...) }
func (p *Plan) Changes() []Change         { return changes(p.before, p.after) }

// Working owns mutable state; it is not safe for concurrent mutation. Original
// syntax snapshots remain immutable and query results never mutate this object.
// One source name denotes one physical file; repeated include occurrences share
// its bytes. Distinct names that alias a host file must be rejected by the host.
type Working struct {
	owner           *byte
	version         uint64
	root            string
	o               Options
	set             *source.Set
	files, original map[string]string
}

// New captures complete source snapshots without I/O. A host must reject any
// input loading error before calling New; EOF alone cannot prove a successful
// load. Strict additionally rejects existing parser/include diagnostics.
func New(set *source.Set, o Options) (*Working, error) {
	if o.MaxBytes <= 0 {
		o.MaxBytes = 16 << 20
	}
	if o.MaxPatches <= 0 {
		o.MaxPatches = 10000
	}
	if o.MaxCommands <= 0 {
		o.MaxCommands = 256
	}
	if o.MaxWork <= 0 {
		o.MaxWork = 256 << 20
	}
	if o.MaxBytes > 64<<20 || o.MaxPatches > 100000 || o.MaxCommands > 1024 || o.MaxWork > 1<<30 {
		return nil, ErrOptions
	}
	if set == nil || set.File(1) == nil {
		return nil, fmt.Errorf("%w: no root source", ErrTarget)
	}
	o.Source.FS = nil
	o.Source.Strict = false
	for _, e := range set.Edges() {
		if e.State == "disabled" {
			o.Source.NoIncludes = true
			break
		}
	}
	w := &Working{owner: new(byte), root: set.File(1).Name(), o: o, set: set, files: map[string]string{}}
	total := 0
	for _, f := range set.Files() {
		if !f.AST().SourceComplete() {
			return nil, fmt.Errorf("%w: incomplete %q", ErrTarget, f.Name())
		}
		data := f.AST().Source()
		if old, ok := w.files[f.Name()]; ok {
			if old != data {
				return nil, fmt.Errorf("%w: inconsistent repeated file %q", ErrTarget, f.Name())
			}
			continue
		}
		if len(data) > o.MaxBytes-total {
			return nil, ErrLimit
		}
		total += len(data)
		w.files[f.Name()] = data
	}
	if o.Strict && len(set.Diagnostics()) != 0 {
		return nil, fmt.Errorf("%w: existing source diagnostics", ErrValidation)
	}
	for _, d := range set.Diagnostics() {
		if strings.Contains(d.Message, "budget exceeded") {
			return nil, ErrLimit
		}
	}
	w.original = cloneFiles(w.files)
	return w, nil
}

func (w *Working) Version() uint64      { return w.version }
func (w *Working) RootName() string     { return w.root }
func (w *Working) Sources() *source.Set { return w.set }
func (w *Working) Names() []string {
	names := make([]string, 0, len(w.files))
	for n := range w.files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
func (w *Working) Data(name string) ([]byte, error) {
	s, ok := w.files[name]
	if !ok {
		return nil, ErrTarget
	}
	return []byte(s), nil
}
func (w *Working) Changes() []Change { return changes(w.original, w.files) }

// Query executes against the current immutable snapshot. Stored indices become
// stale after Apply; selectors in subsequent commands execute again.
func (w *Working) Query(ctx context.Context, selector query.Selector, o query.Options) (*query.Result, error) {
	return query.Execute(ctx, w.set, selector, o)
}

// Apply publishes a validated plan in memory, atomically. Even a no-op advances
// Version so plans can never be replayed accidentally. No filesystem is touched.
func (w *Working) Apply(p *Plan) error {
	if p == nil || p.owner != w.owner || p.version != w.version || !sameFiles(w.files, p.before) {
		return ErrStale
	}
	w.files = cloneFiles(p.after)
	w.set = p.set
	w.version++
	return nil
}
func digest(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }
func cloneFiles(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for n, s := range in {
		out[n] = s
	}
	return out
}
func sameFiles(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for n, s := range a {
		if t, ok := b[n]; !ok || t != s {
			return false
		}
	}
	return true
}
func clonePatches(in []Patch) []Patch {
	out := append([]Patch(nil), in...)
	for i := range out {
		out[i].Data = append([]byte(nil), out[i].Data...)
	}
	return out
}
func changes(before, after map[string]string) []Change {
	var result []Change
	for n, s := range after {
		if before[n] != s {
			result = append(result, Change{n, digest(before[n]), digest(s), []byte(before[n]), []byte(s)})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Source < result[j].Source })
	return result
}
