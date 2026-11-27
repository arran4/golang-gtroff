package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Format string

const (
	Text  Format = "text"
	JSON  Format = "json"
	JSONL Format = "jsonl"
)

// Write serializes classification without changing it. Text is a concise summary
// unless details is true. JSON contains the full report. JSONL emits a versioned
// summary record followed by versioned finding records; summary counts describe
// only the enumerated prefix if Complete is false. JSON replaces invalid UTF-8;
// source-exact bytes remain in the original source.Set, not this inventory.
func (r *Report) Write(writer io.Writer, format Format, details bool) error {
	if writer == nil || r == nil {
		return fmt.Errorf("nil analysis result/writer")
	}
	if err := ValidateFormat(format, details); err != nil {
		return err
	}
	out := checkedWriter{writer}
	switch format {
	case JSON:
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(r)
	case JSONL:
		encoder := json.NewEncoder(out)
		summary := *r
		summary.Findings = nil
		if err := encoder.Encode(struct {
			SchemaVersion int    `json:"schemaVersion"`
			Type          string `json:"type"`
			Report        Report `json:"report"`
		}{1, "summary", summary}); err != nil {
			return err
		}
		for _, finding := range r.Findings {
			if err := encoder.Encode(struct {
				SchemaVersion int     `json:"schemaVersion"`
				Type          string  `json:"type"`
				Finding       Finding `json:"finding"`
			}{1, "finding", finding}); err != nil {
				return err
			}
		}
		return nil
	case Text:
		var text strings.Builder
		fmt.Fprintf(&text, "profile: %s (confidence=%s)\nreason: %s\n", r.Profile.Name, r.Profile.Confidence, r.Profile.Reason)
		bytes, nodes, lines := 0, 0, 0
		for _, file := range r.Sources {
			bytes += file.Bytes
			nodes += file.Nodes
			lines += file.Lines
		}
		fmt.Fprintf(&text, "sources: %d; bytes: %d; nodes: %d; physical lines: %d; enumeration complete: %t\n", len(r.Sources), bytes, nodes, lines, r.Complete)
		fmt.Fprintln(&text, "capabilities: syntax and optional man interpretation assessed separately; rendering not assessed")
		fmt.Fprintf(&text, "flags: conditional=%t include=%t evaluation=%t preprocessor=%t side-effect-syntax=%t unknown=%t opaque=%t partial-source=%t\n", r.Flags.HasConditional, r.Flags.HasInclude, r.Flags.RequiresEvaluation, r.Flags.UsesPreprocessor, r.Flags.HasSideEffectSyntax, r.Flags.HasUnknown, r.Flags.HasOpaque, r.Flags.HasPartialSource)
		text.WriteString("requirements:")
		if len(r.Requirements) > 0 {
			fmt.Fprintf(&text, " %s", strings.Join(r.Requirements, ", "))
		}
		text.WriteByte('\n')
		countsLine(&text, "features", r.Counts.Features)
		countsLine(&text, "controls", r.Counts.Controls)
		countsLine(&text, "escapes", r.Counts.Escapes)
		for _, file := range r.Sources {
			fmt.Fprintf(&text, "source %d: %s (profile=%s depth=%d bytes=%d eof=%t)\n", file.ID, file.Name, file.Profile.Name, file.Depth, file.Bytes, file.Complete)
		}
		for _, edge := range r.Includes {
			fmt.Fprintf(&text, "include %s:%d:%d: %s [%s] -> %d\n", edge.Span.Start.Filename, edge.Span.Start.Line, edge.Span.Start.Column, edge.Target, edge.State, edge.To)
		}
		fmt.Fprintf(&text, "diagnostics: %d\n", len(r.Diagnostics))
		if details {
			for _, finding := range r.Findings {
				fmt.Fprintf(&text, "%s:%d:%d: %s %s [syntax=%s semantics=%s resolution=%s active=%t] %s\n", finding.Span.Start.Filename, finding.Span.Start.Line, finding.Span.Start.Column, finding.Feature, finding.Name, finding.Capability.Syntax, finding.Capability.Semantics, finding.Capability.Resolution, finding.Active, finding.Reason)
			}
			for _, diagnostic := range r.Diagnostics {
				fmt.Fprintf(&text, "%s:%d:%d: diagnostic %s: %s\n", diagnostic.Position.Filename, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Code, diagnostic.Message)
			}
		}
		if r.InputError != "" {
			fmt.Fprintf(&text, "input error: %s\n", r.InputError)
		}
		if r.Error != "" {
			fmt.Fprintf(&text, "analysis error: %s\n", r.Error)
		}
		_, err := out.Write([]byte(text.String()))
		return err
	}
	return nil
}
func ValidateFormat(format Format, details bool) error {
	if format != Text && format != JSON && format != JSONL {
		return fmt.Errorf("format must be text, json or jsonl")
	}
	if details && format != Text {
		return fmt.Errorf("details applies only to text output")
	}
	return nil
}
func countsLine(out *strings.Builder, name string, counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Fprintf(out, "%s:", name)
	for _, key := range keys {
		fmt.Fprintf(out, " %s=%d", key, counts[key])
	}
	out.WriteByte('\n')
}

type checkedWriter struct{ io.Writer }

func (w checkedWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}
