package docdiff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Format string

const (
	Text  Format = "text"
	JSON  Format = "json"
	JSONL Format = "jsonl"
)

func ValidateFormat(format Format) error {
	if format != Text && format != JSON && format != JSONL {
		return fmt.Errorf("%w: format must be text, json or jsonl", ErrOptions)
	}
	return nil
}

// Write exports the full JSON report, JSONL summary/change/ambiguity envelopes,
// or positioned text. Serialization has an actual-byte budget independent of
// comparison data. Writer errors can leave a prefix; JSON uses UTF-8 replacement.
func (r *Report) Write(ctx context.Context, writer io.Writer, format Format, maxBytes int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if writer == nil || r == nil || nilWriter(writer) {
		return fmt.Errorf("%w: nil writer/report", ErrOptions)
	}
	if err := ValidateFormat(format); err != nil {
		return err
	}
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	if maxBytes > 64<<20 {
		return fmt.Errorf("%w: output byte limit exceeds ceiling", ErrOptions)
	}
	out := &output{ctx, writer, maxBytes}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch format {
	case JSON:
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(r)
	case JSONL:
		summary := *r
		summary.Changes = nil
		summary.Ambiguities = nil
		encoder := json.NewEncoder(out)
		if err := encoder.Encode(struct {
			SchemaVersion int    `json:"schemaVersion"`
			Type          string `json:"type"`
			Changes       int    `json:"changeCount"`
			Ambiguities   int    `json:"ambiguityCount"`
			Report        Report `json:"report"`
		}{1, "summary", len(r.Changes), len(r.Ambiguities), summary}); err != nil {
			return err
		}
		for _, change := range r.Changes {
			if err := encoder.Encode(struct {
				SchemaVersion int    `json:"schemaVersion"`
				Type          string `json:"type"`
				Change        Change `json:"change"`
			}{1, "change", change}); err != nil {
				return err
			}
		}
		for _, ambiguity := range r.Ambiguities {
			if err := encoder.Encode(struct {
				SchemaVersion int       `json:"schemaVersion"`
				Type          string    `json:"type"`
				Ambiguity     Ambiguity `json:"ambiguity"`
			}{1, "ambiguity", ambiguity}); err != nil {
				return err
			}
		}
		return nil
	case Text:
		if _, err := fmt.Fprintf(out, "diff: %s (assessment=%s complete=%t changes=%d ambiguities=%d)\nbefore: %q sources=%d objects=%d\nafter: %q sources=%d objects=%d\n", r.Status, r.Assessment, r.Complete, len(r.Changes), len(r.Ambiguities), r.Before.Name, len(r.Before.Files), r.Before.Objects, r.After.Name, len(r.After.Files), r.After.Objects); err != nil {
			return err
		}
		for _, change := range r.Changes {
			object := changeObject(change)
			if _, err := fmt.Fprintf(out, "%d %s %s %q [confidence=%s moved=%t content-changed=%t]: %s\n", change.Index, change.Kind, object.Kind, object.Identity, change.Confidence, change.Moved, change.ContentChanged, change.Reason); err != nil {
				return err
			}
			for _, side := range []struct {
				name   string
				object *Object
			}{{"before", change.Before}, {"after", change.After}} {
				if side.object == nil {
					continue
				}
				location := side.object.Location
				if _, err := fmt.Fprintf(out, "  %s %q:%d:%d source=%d [%d,%d) value=%q\n", side.name, location.Source.Name, location.Span.Start.Line, location.Span.Start.Column, location.Source.ID, location.Span.Start.Offset, location.Span.End.Offset, side.object.Value); err != nil {
					return err
				}
			}
		}
		for _, ambiguity := range r.Ambiguities {
			if _, err := fmt.Fprintf(out, "ambiguous %s %q: before=%d after=%d: %s\n", ambiguity.Kind, ambiguity.Identity, len(ambiguity.Before), len(ambiguity.After), ambiguity.Reason); err != nil {
				return err
			}
		}
		for _, ambiguity := range r.Ambiguities {
			for _, side := range []struct {
				name    string
				objects []Object
			}{{"before", ambiguity.Before}, {"after", ambiguity.After}} {
				for _, object := range side.objects {
					location := object.Location
					if _, err := fmt.Fprintf(out, "  ambiguous %s %q:%d:%d source=%d identity=%q\n", side.name, location.Source.Name, location.Span.Start.Line, location.Span.Start.Column, location.Source.ID, object.Identity); err != nil {
						return err
					}
				}
			}
		}
		for _, side := range []struct {
			name string
			info InputInfo
		}{{"before", r.Before}, {"after", r.After}} {
			for _, d := range side.info.Diagnostics {
				if _, err := fmt.Fprintf(out, "%s %q:%d:%d: input %s %s: %s\n", side.name, d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Code, d.Message); err != nil {
					return err
				}
			}
			if side.info.Error != "" {
				if _, err := fmt.Fprintf(out, "%s input error: %s\n", side.name, side.info.Error); err != nil {
					return err
				}
			}
		}
		for _, d := range r.Diagnostics {
			if _, err := fmt.Fprintf(out, "%s %q:%d:%d: %s: %s\n", d.Side, d.Position.Filename, d.Position.Line, d.Position.Column, d.Code, d.Message); err != nil {
				return err
			}
		}
		if r.Error != "" {
			if _, err := fmt.Fprintf(out, "diff error: %s\n", r.Error); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

type output struct {
	ctx       context.Context
	writer    io.Writer
	remaining int
}

func (w *output) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(data) > w.remaining {
		return 0, ErrLimit
	}
	n, err := w.writer.Write(data)
	if n < 0 || n > len(data) {
		return 0, errors.Join(fmt.Errorf("invalid diff writer count %d", n), err)
	}
	w.remaining -= n
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}

func nilWriter(writer io.Writer) bool {
	value := reflect.ValueOf(writer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}
