package extract

import (
	"context"
	"errors"
	"fmt"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func defaults(o Options) (Options, error) {
	if o.MaxNodes <= 0 {
		o.MaxNodes = 200000
	}
	if o.MaxRecords <= 0 {
		o.MaxRecords = 10000
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = 16 << 20
	}
	if o.MaxWork <= 0 {
		o.MaxWork = 64 << 20
	}
	if o.MaxNodes > 1000000 || o.MaxRecords > 100000 || o.MaxBytes > 64<<20 || o.MaxWork > 1<<30 {
		return o, fmt.Errorf("%w: limits exceed supported ceilings", ErrOptions)
	}
	return o, nil
}

// Select uses the same query IR and executor as query.Execute, then constructs
// extraction records. Both phases share MaxWork. No selector state or source
// mutation is retained. Execution/strict errors return useful partial reports.
func Select(ctx context.Context, set *source.Set, selector query.Selector, o Options) (*Report, error) {
	o, err := defaults(o)
	if err != nil {
		report := newReport(nil, o)
		report.Error = err.Error()
		return report, err
	}
	q := o.Query
	if q.MaxWork <= 0 || q.MaxWork > o.MaxWork {
		q.MaxWork = o.MaxWork
	}
	if q.MaxMatches <= 0 || q.MaxMatches > o.MaxRecords {
		q.MaxMatches = o.MaxRecords
	}
	if q.MaxNodes <= 0 || q.MaxNodes > o.MaxNodes {
		q.MaxNodes = o.MaxNodes
	}
	if q.MaxResultBytes <= 0 || q.MaxResultBytes > o.MaxBytes {
		q.MaxResultBytes = o.MaxBytes
	}
	selected, queryErr := query.Execute(ctx, set, selector, q)
	report, extractErr := fromQuery(ctx, set, selected, o, selected.WorkUnits)
	if queryErr != nil {
		report.QueryError = queryErr.Error()
	}
	return report, errors.Join(queryErr, extractErr)
}

// FromQuery consumes an existing result (or an ordered subset of its matches).
// It verifies source digests/occurrence metadata and source-backed node/spans,
// then derives content again from the immutable source, rather than trusting
// editable Match.Text/Raw/Term fields. It does not authenticate who selected a
// match. Query indices/IDs are provenance labels; filtered indices need not be
// contiguous. Construction has its own budget, independent of prior query cost.
func FromQuery(ctx context.Context, set *source.Set, selected *query.Result, o Options) (*Report, error) {
	return fromQuery(ctx, set, selected, o, 0)
}
func newReport(selected *query.Result, o Options) *Report {
	r := &Report{SchemaVersion: 1, Command: "extract", Records: []Record{}, HTMLPrepared: o.HTML}
	if selected != nil {
		r.View = selected.View
		r.QueryComplete = selected.Complete
		r.QueryError = selected.Error
		r.InputError = selected.InputError
		r.Diagnostics = append(r.Diagnostics, selected.Diagnostics...)
	}
	return r
}
