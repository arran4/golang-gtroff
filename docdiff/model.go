// Package docdiff compares explicit roff source snapshots using the existing
// query/extraction man projection or a separate exact-source view. It performs
// no input I/O, macro execution or mutation. Findings describe observed source
// or projected content, not universal groff rendering equivalence.
package docdiff

import (
	"context"
	"errors"
	"fmt"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

type View string

const (
	Semantic View = "semantic"
	Source   View = "source"
)

type Kind string

const (
	Metadata   Kind = "metadata"
	Section    Kind = "section"
	Subsection Kind = "subsection"
	Definition Kind = "definition"
	Reference  Kind = "reference"
	Preamble   Kind = "preamble"
	Opaque     Kind = "opaque"
	SourceFile Kind = "source"
)

type Input struct {
	Name    string
	Sources *source.Set
	Error   error
}

// Location always belongs to the named original source occurrence/revision.
// SourceKey relates include occurrences without using volatile byte offsets;
// IDs are still local to that side's captured snapshot.
type Location struct {
	Source     query.SourceInfo `json:"source"`
	SourceKey  string           `json:"sourceKey"`
	Span       groff.Span       `json:"span"`
	ASTIDs     []groff.NodeID   `json:"astIds,omitempty"`
	SemanticID man.ID           `json:"semanticId,omitempty"`
}

// Object is a compared content unit. Value is a deterministic content payload,
// not original roff; Digest hashes that payload. Opaque/source objects retain
// exact selected bytes. Formatting style/flow spacing are ignored in semantic
// view; preformatted text, targets, metadata and uninterpreted bytes remain data.
type Object struct {
	Index         int      `json:"index"`
	Kind          Kind     `json:"kind"`
	Identity      string   `json:"identity"`
	Scope         string   `json:"scope"`
	Location      Location `json:"location"`
	Value         string   `json:"value"`
	Digest        string   `json:"digest"`
	Uninterpreted bool     `json:"uninterpreted"`
}
type ChangeKind string

const (
	Added   ChangeKind = "added"
	Removed ChangeKind = "removed"
	Changed ChangeKind = "changed"
	Moved   ChangeKind = "moved"
)

type Change struct {
	SchemaVersion  int        `json:"schemaVersion"`
	Index          int        `json:"index"`
	Kind           ChangeKind `json:"kind"`
	Before         *Object    `json:"before,omitempty"`
	After          *Object    `json:"after,omitempty"`
	Moved          bool       `json:"moved,omitempty"`
	ContentChanged bool       `json:"contentChanged,omitempty"`
	Confidence     string     `json:"confidence"`
	Reason         string     `json:"reason"`
}
type Ambiguity struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          Kind     `json:"kind"`
	Identity      string   `json:"identity"`
	Before        []Object `json:"before"`
	After         []Object `json:"after"`
	Reason        string   `json:"reason"`
}
type Diagnostic struct {
	Side     string         `json:"side,omitempty"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Source   source.ID      `json:"source,omitempty"`
	Position groff.Position `json:"position"`
}
type InputInfo struct {
	Name        string              `json:"name"`
	Error       string              `json:"error,omitempty"`
	Files       []query.SourceInfo  `json:"files"`
	Includes    []source.Edge       `json:"includes,omitempty"`
	Diagnostics []source.Diagnostic `json:"diagnostics,omitempty"`
	Objects     int                 `json:"objects"`
	Complete    bool                `json:"complete"`
}

// Complete describes comparison enumeration. Status is equal, different,
// ambiguous, strict-invalid or incomplete. Equality concerns only Assessment;
// unsupported syntax and ambiguous identities are retained explicitly.
type Report struct {
	SchemaVersion int          `json:"schemaVersion"`
	Command       string       `json:"command"`
	View          View         `json:"view"`
	Assessment    string       `json:"assessment"`
	Status        string       `json:"status"`
	Complete      bool         `json:"complete"`
	Different     bool         `json:"different"`
	Error         string       `json:"error,omitempty"`
	Before        InputInfo    `json:"before"`
	After         InputInfo    `json:"after"`
	Changes       []Change     `json:"changes"`
	Ambiguities   []Ambiguity  `json:"ambiguities"`
	Diagnostics   []Diagnostic `json:"diagnostics,omitempty"`
}
type Options struct {
	View                                                                    View
	Strict                                                                  bool
	MaxNodes, MaxObjects, MaxChanges, MaxDataBytes, MaxResultBytes, MaxWork int
}

var (
	ErrOptions = errors.New("invalid documentation diff options")
	ErrInput   = errors.New("documentation diff input is incomplete or invalid")
	ErrStrict  = errors.New("strict documentation diff rejected diagnostics")
	ErrLimit   = errors.New("documentation diff resource limit exceeded")
)

type meter struct {
	ctx       context.Context
	remaining int
}

func (m *meter) spend(n int) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > m.remaining {
		return ErrLimit
	}
	m.remaining -= n
	return nil
}
func defaults(o Options) (Options, error) {
	if o.View == "" {
		o.View = Semantic
	}
	if o.View != Semantic && o.View != Source {
		return o, fmt.Errorf("%w: view must be semantic or source", ErrOptions)
	}
	if o.MaxNodes <= 0 {
		o.MaxNodes = 200000
	}
	if o.MaxObjects <= 0 {
		o.MaxObjects = 10000
	}
	if o.MaxChanges <= 0 {
		o.MaxChanges = 10000
	}
	if o.MaxDataBytes <= 0 {
		o.MaxDataBytes = 32 << 20
	}
	if o.MaxResultBytes <= 0 {
		o.MaxResultBytes = 16 << 20
	}
	if o.MaxWork <= 0 {
		o.MaxWork = 64 << 20
	}
	if o.MaxNodes > 1000000 || o.MaxObjects > 100000 || o.MaxChanges > 100000 || o.MaxDataBytes > 64<<20 || o.MaxResultBytes > 64<<20 || o.MaxWork > 1<<30 {
		return o, fmt.Errorf("%w: limits exceed supported ceilings", ErrOptions)
	}
	return o, nil
}
