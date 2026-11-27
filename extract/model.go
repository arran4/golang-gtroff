// Package extract turns stateless query selections into source-linked records and
// task-specific exports. Source-exact roff, syntax text, optional man semantics
// and rendered HTML remain distinct representations. No input I/O or execution
// capabilities are acquired by the library.
package extract

import (
	"context"
	"errors"
	"io"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

var (
	ErrLimit      = errors.New("extraction resource limit exceeded")
	ErrStrict     = errors.New("strict extraction rejected diagnostics")
	ErrSnapshot   = errors.New("query match does not identify the supplied source snapshot")
	ErrIncomplete = errors.New("query selection is incomplete")
	ErrOptions    = errors.New("invalid extraction options")
)

type Heading struct {
	Source source.ID  `json:"source"`
	ID     man.ID     `json:"semanticId"`
	Name   string     `json:"name"`
	Span   groff.Span `json:"span"`
}

// Record is caller-owned extracted data. Blocks retain original local semantic
// IDs, runs, spans and external parent references; they are not a standalone man
// Document. Raw always belongs to one original source occurrence, not an include
// expansion. Description omits a definition's term; Text includes selected content.
// NormalizedText folds whitespace without changing case. HTML is populated only
// when explicitly requested and is produced by the existing man HTML renderer.
type Record struct {
	MetadataOrigin *man.Origin         `json:"metadataOrigin,omitempty"`
	Semantic       bool                `json:"semantic"`
	SchemaVersion  int                 `json:"schemaVersion"`
	Index          int                 `json:"index"`
	QueryIndex     int                 `json:"queryIndex"`
	ID             string              `json:"id"`
	View           query.View          `json:"view"`
	Kind           query.Kind          `json:"kind"`
	Source         query.SourceInfo    `json:"source"`
	Span           groff.Span          `json:"span"`
	ASTIDs         []groff.NodeID      `json:"astIds,omitempty"`
	SemanticID     man.ID              `json:"semanticId,omitempty"`
	Name           string              `json:"name,omitempty"`
	Headings       []Heading           `json:"headings,omitempty"`
	Metadata       *man.Metadata       `json:"metadata,omitempty"`
	Context        *man.State          `json:"context,omitempty"`
	Term           string              `json:"term,omitempty"`
	TermRuns       []man.Run           `json:"termRuns,omitempty"`
	Description    string              `json:"description,omitempty"`
	Text           string              `json:"text"`
	NormalizedText string              `json:"normalizedText"`
	Raw            string              `json:"raw"`
	Blocks         []man.Node          `json:"blocks,omitempty"`
	Unsupported    bool                `json:"unsupported"`
	Includes       []source.Edge       `json:"includes,omitempty"`
	Diagnostics    []source.Diagnostic `json:"diagnostics,omitempty"`
	HTML           string              `json:"html,omitempty"`
}

// Complete describes record construction. QueryComplete and Source.Complete
// describe separate query-enumeration and original-reader EOF facts. Partial
// results retain diagnostics; they are never proof of runtime/rendering support.
type Report struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Command       string              `json:"command"`
	View          query.View          `json:"view"`
	Complete      bool                `json:"complete"`
	QueryComplete bool                `json:"queryComplete"`
	Error         string              `json:"error,omitempty"`
	QueryError    string              `json:"queryError,omitempty"`
	InputError    string              `json:"inputError,omitempty"`
	Records       []Record            `json:"records"`
	Diagnostics   []source.Diagnostic `json:"diagnostics,omitempty"`
	HTMLPrepared  bool                `json:"htmlPrepared"`

	// WorkUnits records charged selection/construction work for composed callers.
	// It is execution accounting and is omitted from serialized reports.
	WorkUnits int `json:"-"`
}

// Options controls selection/construction, not serialization. Query options
// apply only to Select; FromQuery consumes the result's existing view/matches. MaxWork is shared
// with query execution by Select; FromQuery budgets only its construction phase.
// Defaults: 200000 source/model nodes, 10000 records, 16 MiB materialized data,
// 64 Mi charged work units. HTML is opt-in and produces independent fragments.
type Options struct {
	Query                                   query.Options
	Strict, HTML                            bool
	MaxNodes, MaxRecords, MaxBytes, MaxWork int
}

// Exporter is trusted host code for a contributed output representation. Export
// wraps its writer with cancellation and byte-limit checks; the callback must
// honor ctx for work that does not write. It receives caller-owned report data.
type Exporter interface {
	Export(context.Context, io.Writer, *Report) error
}
type ExporterFunc func(context.Context, io.Writer, *Report) error

func (f ExporterFunc) Export(ctx context.Context, w io.Writer, r *Report) error { return f(ctx, w, r) }

type Format string

const (
	JSON  Format = "json"
	JSONL Format = "jsonl"
	Text  Format = "text"
	Raw   Format = "raw"
	Grep  Format = "grep"
	CSV   Format = "csv"
	TSV   Format = "tsv"
	HTML  Format = "html"
)

type WriteOptions struct {
	Fields    []string
	Normalize bool
	MaxBytes  int
}
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
