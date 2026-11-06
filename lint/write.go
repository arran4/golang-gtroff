package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Format string

const (
	Text  Format = "text"
	JSON  Format = "json"
	JSONL Format = "jsonl"
)

func ValidateFormat(format Format) error {
	if format != Text && format != JSON && format != JSONL {
		return fmt.Errorf("lint format must be text, json or jsonl")
	}
	return nil
}

// Write serializes policy findings without running rules or performing I/O beyond
// writer. JSON contains the full report. JSONL starts with a summary, followed by
// finding and suppressed records. Text includes input/config diagnostics and
// retained suppressions. JSON follows encoding/json UTF-8 replacement rules;
// exact source bytes remain in the original snapshots, not the lint report.
func (r *Report) Write(writer io.Writer, format Format) error {
	if writer == nil || r == nil {
		return fmt.Errorf("nil lint report/writer")
	}
	if err := ValidateFormat(format); err != nil {
		return err
	}
	out := checkedWriter{writer}
	switch format {
	case JSON:
		e := json.NewEncoder(out)
		e.SetIndent("", "  ")
		return e.Encode(r)
	case JSONL:
		e := json.NewEncoder(out)
		summary := *r
		summary.Findings = nil
		summary.Suppressed = nil
		if err := e.Encode(struct {
			SchemaVersion int    `json:"schemaVersion"`
			Type          string `json:"type"`
			Report        Report `json:"report"`
		}{1, "summary", summary}); err != nil {
			return err
		}
		for _, list := range [][]Finding{r.Findings, r.Suppressed} {
			for _, f := range list {
				kind := "finding"
				if f.Suppressed {
					kind = "suppressed"
				}
				if err := e.Encode(struct {
					SchemaVersion int     `json:"schemaVersion"`
					Type          string  `json:"type"`
					Finding       Finding `json:"finding"`
				}{1, kind, f}); err != nil {
					return err
				}
			}
		}
		return nil
	case Text:
		var text strings.Builder
		fmt.Fprintf(&text, "lint: documents=%d findings=%d suppressed=%d complete=%t fail-on=%s failed=%t\n", len(r.Documents), len(r.Findings), len(r.Suppressed), r.Complete, r.Threshold, r.Failed())
		if r.Policy.ClosedWorld {
			fmt.Fprintf(&text, "collection reference check: %s\n", r.ReferenceCheck)
		}
		for _, doc := range r.Documents {
			fmt.Fprintf(&text, "document %d: %s (profile=%s input-invalid=%t)\n", doc.Index, doc.Name, doc.Profile.Name, doc.InputInvalid)
			for _, d := range doc.Diagnostics {
				fmt.Fprintf(&text, "  source %d %s:%d:%d: input %s %s: %s\n", d.Source, d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Code, d.Message)
			}
			if doc.InputError != "" {
				fmt.Fprintf(&text, "  input error: %s\n", doc.InputError)
			}
		}
		for _, f := range r.Findings {
			fmt.Fprintf(&text, "document %d source %d %s:%d:%d: %s %s [%s]: %s\n", f.Document, f.Source, f.Span.Start.Filename, f.Span.Start.Line, f.Span.Start.Column, f.Severity, f.Rule, f.Category, f.Message)
			if f.Suggestion != "" {
				fmt.Fprintf(&text, "  suggestion: %s\n", f.Suggestion)
			}
			for _, related := range f.Related {
				fmt.Fprintf(&text, "  related document %d source %d %s:%d:%d: %s\n", related.Document, related.Source, related.Span.Start.Filename, related.Span.Start.Line, related.Span.Start.Column, related.Message)
			}
		}
		for _, f := range r.Suppressed {
			fmt.Fprintf(&text, "suppressed document %d source %d %s:%d: %s (%s)\n", f.Document, f.Source, f.Span.Start.Filename, f.Span.Start.Line, f.Rule, f.Suppression)
		}
		for _, d := range r.Diagnostics {
			fmt.Fprintf(&text, "document %d source %d %s:%d: suppression configuration: %s\n", d.Document, d.Source, d.Span.Start.Filename, d.Span.Start.Line, d.Message)
		}
		if r.Error != "" {
			fmt.Fprintf(&text, "lint error: %s\n", r.Error)
		}
		_, err := out.Write([]byte(text.String()))
		return err
	}
	return nil
}

type checkedWriter struct{ io.Writer }

func (w checkedWriter) Write(data []byte) (int, error) {
	n, err := w.Writer.Write(data)
	if n < 0 || n > len(data) {
		return 0, fmt.Errorf("invalid writer count %d", n)
	}
	if n != len(data) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}
