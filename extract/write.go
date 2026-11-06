package extract

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Export invokes an explicit trusted exporter with a cancellation-aware bounded
// writer. Exporter or writer failures may leave a partial stream; named-file
// staging/atomic replacement belongs to an explicit host output capability.
func (r *Report) Export(ctx context.Context, w io.Writer, exporter Exporter, maxBytes int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || nilValue(w) || nilValue(exporter) {
		return fmt.Errorf("%w: nil report/writer/exporter", ErrOptions)
	}
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	if maxBytes > 64<<20 {
		return fmt.Errorf("%w: output limit exceeds 64 MiB", ErrOptions)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	out := &output{ctx: ctx, writer: w, remaining: maxBytes}
	if err := exporter.Export(ctx, out, r); err != nil {
		return errors.Join(err, out.terminal)
	}
	return errors.Join(out.terminal, ctx.Err())
}

// Write selects a built-in exporter. JSON contains the full report; JSONL emits
// a summary followed by records so partial/input/unsupported facts stay visible.
// Other outputs serialize each record in order; overlaps deliberately repeat.
// HTML requires Options.HTML during construction. Format/field errors are checked
// before writing. Raw is exact per selected span; text normalization preserves case.
func (r *Report) Write(ctx context.Context, w io.Writer, format Format, o WriteOptions) error {
	if err := ValidateFormat(format, o); err != nil {
		return err
	}
	if format == HTML && r != nil && !r.HTMLPrepared {
		return fmt.Errorf("%w: HTML was not prepared during extraction", ErrOptions)
	}
	return r.Export(ctx, w, ExporterFunc(func(ctx context.Context, w io.Writer, r *Report) error { return writeBuiltin(ctx, w, r, format, o) }), o.MaxBytes)
}
func ValidateFormat(format Format, o WriteOptions) error {
	switch format {
	case JSON, JSONL, Text, Raw, Grep, CSV, TSV, HTML:
	default:
		return fmt.Errorf("%w: format must be json, jsonl, text, raw, grep, csv, tsv or html", ErrOptions)
	}
	if len(o.Fields) > 0 && format != CSV && format != TSV {
		return fmt.Errorf("%w: fields require csv or tsv", ErrOptions)
	}
	if o.Normalize && format != Text {
		return fmt.Errorf("%w: normalize applies only to text output", ErrOptions)
	}
	if len(o.Fields) > 32 {
		return fmt.Errorf("%w: at most 32 fields", ErrOptions)
	}
	if o.MaxBytes < 0 || o.MaxBytes > 64<<20 {
		return fmt.Errorf("%w: output byte limit outside supported range", ErrOptions)
	}
	for _, field := range o.Fields {
		switch field {
		case "index", "query-index", "id", "source-id", "file", "digest", "line", "column", "offset", "end-offset", "kind", "name", "headings", "term", "description", "text", "normalized-text", "raw", "unsupported":
		default:
			return fmt.Errorf("%w: unknown output field %q", ErrOptions, field)
		}
	}
	return nil
}
func writeBuiltin(ctx context.Context, w io.Writer, r *Report, format Format, o WriteOptions) error {
	switch format {
	case JSON:
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(r)
	case JSONL:
		e := json.NewEncoder(w)
		summary := *r
		summary.Records = nil
		if err := e.Encode(struct {
			SchemaVersion int    `json:"schemaVersion"`
			Type          string `json:"type"`
			Report        Report `json:"report"`
		}{1, "summary", summary}); err != nil {
			return err
		}
		for _, record := range r.Records {
			if err := e.Encode(struct {
				SchemaVersion int    `json:"schemaVersion"`
				Type          string `json:"type"`
				Record        Record `json:"record"`
			}{1, "record", record}); err != nil {
				return err
			}
		}
		return nil
	case CSV, TSV:
		fields := o.Fields
		if len(fields) == 0 {
			fields = []string{"index", "file", "line", "kind", "term", "description", "text", "unsupported"}
		}
		writer := csv.NewWriter(w)
		if format == TSV {
			writer.Comma = '\t'
		}
		if err := writer.Write(fields); err != nil {
			return err
		}
		for _, record := range r.Records {
			if err := ctx.Err(); err != nil {
				return err
			}
			row := make([]string, len(fields))
			for i, field := range fields {
				row[i] = recordField(record, field)
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	default:
		for _, record := range r.Records {
			if err := ctx.Err(); err != nil {
				return err
			}
			text := record.Text
			switch format {
			case Raw:
				text = record.Raw
			case HTML:
				text = record.HTML
			case Grep:
				text = fmt.Sprintf("%s:%d:%d:%s\n", record.Source.Name, record.Span.Start.Line, record.Span.Start.Column, record.NormalizedText)
			case Text:
				if o.Normalize {
					text = record.NormalizedText
				}
				if !strings.HasSuffix(text, "\n") {
					text += "\n"
				}
			}
			if _, err := io.WriteString(w, text); err != nil {
				return err
			}
		}
		return nil
	}
}
func recordField(r Record, field string) string {
	switch field {
	case "index":
		return strconv.Itoa(r.Index)
	case "query-index":
		return strconv.Itoa(r.QueryIndex)
	case "id":
		return r.ID
	case "source-id":
		return strconv.FormatUint(uint64(r.Source.ID), 10)
	case "file":
		return r.Source.Name
	case "digest":
		return r.Source.Digest
	case "line":
		return strconv.Itoa(r.Span.Start.Line)
	case "column":
		return strconv.Itoa(r.Span.Start.Column)
	case "offset":
		return strconv.Itoa(r.Span.Start.Offset)
	case "end-offset":
		return strconv.Itoa(r.Span.End.Offset)
	case "kind":
		return string(r.Kind)
	case "name":
		return r.Name
	case "headings":
		names := make([]string, len(r.Headings))
		for i, h := range r.Headings {
			names[i] = h.Name
		}
		return strings.Join(names, " / ")
	case "term":
		return r.Term
	case "description":
		return r.Description
	case "text":
		return r.Text
	case "normalized-text":
		return r.NormalizedText
	case "raw":
		return r.Raw
	case "unsupported":
		return strconv.FormatBool(r.Unsupported)
	}
	return ""
}

type output struct {
	terminal  error
	ctx       context.Context
	writer    io.Writer
	remaining int
}

func (w *output) Write(data []byte) (n int, err error) {
	if w.terminal != nil {
		return 0, w.terminal
	}
	defer func() {
		if err != nil {
			w.terminal = err
		}
	}()
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(data) > w.remaining {
		return 0, ErrLimit
	}
	n, err = w.writer.Write(data)
	if n < 0 || n > len(data) {
		return 0, errors.Join(fmt.Errorf("invalid output writer count %d", n), err)
	}
	w.remaining -= n
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}

func nilValue(v any) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}
