package docdiff

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/arran4/golang-gtroff/source"
)

// Compare inventories complete supplied snapshots and compares exact source bytes
// or existing man-projected content. It never loads resources or mutates input.
// Missing/incomplete inputs return a diagnostic report without invented changes.
// Duplicate identities are reported as ambiguity rather than paired by position.
func Compare(ctx context.Context, before, after Input, o Options) (report *Report, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	report = &Report{SchemaVersion: 1, Command: "diff", Status: "incomplete", Changes: []Change{}, Ambiguities: []Ambiguity{}}
	defer func() {
		sortChanges(report)
		report.Different = len(report.Changes) > 0
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			report.Error = err.Error()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				report.Complete = false
				report.Status = "incomplete"
			}
		}
		sort.SliceStable(report.Diagnostics, func(i, j int) bool {
			a, b := report.Diagnostics[i], report.Diagnostics[j]
			if a.Side != b.Side {
				return a.Side == "before"
			}
			if a.Source != b.Source {
				return a.Source < b.Source
			}
			if a.Position.Offset != b.Position.Offset {
				return a.Position.Offset < b.Position.Offset
			}
			return a.Code < b.Code
		})
	}()
	o, err = defaults(o)
	if err != nil {
		return report, err
	}
	report.View = o.View
	report.Assessment = "man-content-projection"
	if o.View == Source {
		report.Assessment = "exact-source-bytes"
	}
	m := &meter{ctx, o.MaxWork}
	left, leftErr := inspect(before, "before", o, m)
	report.Before = left.info
	right, rightErr := inspect(after, "after", o, m)
	report.After = right.info
	if leftErr != nil || rightErr != nil {
		err = joinDistinct(leftErr, rightErr, before.Error, after.Error)
		if o.Strict && (errors.Is(before.Error, source.ErrStrict) || errors.Is(after.Error, source.ErrStrict)) {
			err = errors.Join(ErrStrict, err)
		}
		return report, err
	}
	if left.disabled && right.children || right.disabled && left.children {
		return report, fmt.Errorf("%w: include loading policies are not comparable", ErrInput)
	}
	data := 0
	old, err := buildIndex(left, "before", o, m, &data)
	report.Diagnostics = append(report.Diagnostics, old.diagnostics...)
	report.Before.Objects = len(old.entities)
	if err != nil {
		return report, err
	}
	next, err := buildIndex(right, "after", o, m, &data)
	report.Diagnostics = append(report.Diagnostics, next.diagnostics...)
	report.After.Objects = len(next.entities)
	if err != nil {
		return report, err
	}
	matching := &matcher{report: report, options: o, meter: m}
	if err = matching.compare(old.entities, next.entities); err != nil {
		return report, err
	}
	report.Complete = true
	report.Status = "equal"
	report.Different = len(report.Changes) > 0
	if report.Different {
		report.Status = "different"
	} else if len(report.Ambiguities) > 0 {
		report.Status = "ambiguous"
	}
	if o.Strict && (len(report.Before.Diagnostics) > 0 || len(report.After.Diagnostics) > 0 || len(report.Diagnostics) > 0 || len(report.Ambiguities) > 0) || errors.Is(before.Error, source.ErrStrict) || errors.Is(after.Error, source.ErrStrict) {
		report.Status = "strict-invalid"
		return report, errors.Join(ErrStrict, before.Error, after.Error)
	}
	return report, nil
}

func joinDistinct(causes ...error) error {
	var joined error
	for _, cause := range causes {
		if cause != nil && !errors.Is(joined, cause) {
			joined = errors.Join(joined, cause)
		}
	}
	return joined
}
