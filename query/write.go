package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Format controls serialization only; it does not change query selection.
type Format string

const (
	JSON      Format = "json"
	JSONL     Format = "jsonl"
	PlainText Format = "text"
	RawSource Format = "raw"
	Grep      Format = "grep"
	CSV       Format = "csv"
	TSV       Format = "tsv"
)

// Write serializes a result to an explicit writer. JSON exports the full report;
// JSONL writes self-contained matches. Text/raw concatenate each match in ordinal
// order (overlapping selections deliberately repeat bytes). Grep is one normalized
// line per match. CSV/TSV accept explicit fields; nil uses location/content fields.
// RawSource is byte-exact per selected span; JSON follows UTF-8 replacement rules.
// Unsupported fields/formats are rejected before any bytes are written.
func (r *Result) Write(w io.Writer, format Format, fields []string) error {
	if w == nil {
		return fmt.Errorf("nil query output writer")
	}
	if r == nil {
		return fmt.Errorf("nil query result")
	}
	if err := ValidateFormat(format, fields); err != nil {
		return err
	}
	out := checkedWriter{w}
	switch format {
	case JSON:
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(r)
	case JSONL:
		encoder := json.NewEncoder(out)
		for _, match := range r.Matches {
			if err := encoder.Encode(match); err != nil {
				return err
			}
		}
		return nil
	case PlainText, RawSource, Grep:
		for _, match := range r.Matches {
			value := match.Text
			if format == RawSource {
				value = match.Raw
			} else if format == Grep {
				value = fmt.Sprintf("%s:%d:%d:%s\n", match.Source.Name, match.Span.Start.Line, match.Span.Start.Column, match.NormalizedText)
			} else if !strings.HasSuffix(value, "\n") {
				value += "\n"
			}
			if _, err := out.Write([]byte(value)); err != nil {
				return err
			}
		}
		return nil
	case CSV, TSV:
		if len(fields) == 0 {
			fields = []string{"index", "file", "line", "column", "kind", "term", "text"}
		}
		writer := csv.NewWriter(out)
		if format == TSV {
			writer.Comma = '\t'
		}
		if err := writer.Write(fields); err != nil {
			return err
		}
		for _, match := range r.Matches {
			values := make([]string, len(fields))
			for i, field := range fields {
				values[i] = recordField(match, field)
			}
			if err := writer.Write(values); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	}
	return fmt.Errorf("unsupported query output format")
}

// ValidateFormat checks output choices without writing or inspecting a source.
// fields apply only to CSV/TSV; this avoids silently ignored output options.
func ValidateFormat(format Format, fields []string) error {
	switch format {
	case JSON, JSONL, PlainText, RawSource, Grep, CSV, TSV:
	default:
		return fmt.Errorf("format must be json, jsonl, text, raw, grep, csv or tsv")
	}
	if len(fields) > 0 && format != CSV && format != TSV {
		return fmt.Errorf("fields require csv or tsv output")
	}
	if len(fields) > 32 {
		return fmt.Errorf("at most 32 output fields are supported")
	}
	for _, field := range fields {
		switch field {
		case "index", "id", "source-id", "file", "line", "column", "offset", "end-offset", "kind", "name", "term", "text", "normalized-text", "raw":
		default:
			return fmt.Errorf("unknown query output field %q", field)
		}
	}
	return nil
}
func recordField(match Match, field string) string {
	switch field {
	case "index":
		return strconv.Itoa(match.Index)
	case "id":
		return match.ID
	case "source-id":
		return strconv.FormatUint(uint64(match.Source.ID), 10)
	case "file":
		return match.Source.Name
	case "line":
		return strconv.Itoa(match.Span.Start.Line)
	case "column":
		return strconv.Itoa(match.Span.Start.Column)
	case "offset":
		return strconv.Itoa(match.Span.Start.Offset)
	case "end-offset":
		return strconv.Itoa(match.Span.End.Offset)
	case "kind":
		return string(match.Kind)
	case "name":
		return match.Name
	case "term":
		return match.Term
	case "text":
		return match.Text
	case "normalized-text":
		return match.NormalizedText
	case "raw":
		return match.Raw
	}
	return ""
}

type checkedWriter struct{ io.Writer }

func (w checkedWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}
