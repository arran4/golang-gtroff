package docdiff_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/docdiff"
)

func TestSerializationAndPartialOrdinals(t *testing.T) {
	r := compare(t, header+def("A", "Old."), header+def("A", "New."), docdiff.Options{})
	for _, format := range []docdiff.Format{docdiff.Text, docdiff.JSON, docdiff.JSONL} {
		var out bytes.Buffer
		if err := r.Write(context.Background(), &out, format, 0); err != nil {
			t.Fatal(err)
		}
		if format == docdiff.JSONL {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			if len(lines) != 2 {
				t.Fatal("missing JSONL summary/change")
			}
			for _, line := range lines {
				if !json.Valid([]byte(line)) {
					t.Fatal("invalid JSONL")
				}
			}
		}
		if err := r.Write(context.Background(), shortWriter{}, format, 0); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal("short write lost", err)
		}
		if err := r.Write(context.Background(), io.Discard, format, 1); !errors.Is(err, docdiff.ErrLimit) {
			t.Fatal("output cap ignored", err)
		}
	}
	r, err := docdiff.Compare(context.Background(), input(t, "old", header+def("A", "One.")), input(t, "new", header+def("B", "Two.")), docdiff.Options{MaxChanges: 1})
	if !errors.Is(err, docdiff.ErrLimit) || len(r.Changes) != 1 || r.Changes[0].Index != 1 || !r.Different || r.Complete {
		t.Fatal("partial prefix lost status/ordinal", r, err)
	}
}
func TestReferenceFontAndLineWrappingAreNotChanges(t *testing.T) {
	a := header + ".UR https://example.com\nlink text\n.UE\n"
	b := header + ".UR https://example.com\n.B link\ntext\n.UE\n"
	r := compare(t, a, b, docdiff.Options{})
	if r.Different || len(r.Ambiguities) > 0 {
		t.Fatalf("font/wrap false changes %+v %+v", r.Changes, r.Ambiguities)
	}
}
func TestInlineUnsupportedRemainsExplicit(t *testing.T) {
	r := compare(t, header+"A \\[old-unknown] value.\n", header+"A \\[new-unknown] value.\n", docdiff.Options{})
	if !r.Different || len(r.Diagnostics) == 0 {
		t.Fatal("unsupported inline hidden")
	}
	for _, change := range r.Changes {
		if change.Confidence != "uninterpreted-source" {
			t.Fatal("unsupported inline asserted modeled", change)
		}
	}
}
func TestSourceViewMalformedAndStrict(t *testing.T) {
	a, b := input(t, "old.1", ".TH\n"), input(t, "new.1", ".TH bad\n")
	r, err := docdiff.Compare(context.Background(), a, b, docdiff.Options{View: docdiff.Source})
	if err != nil || !r.Different {
		t.Fatal("captured malformed source was not compared", err)
	}
	r, err = docdiff.Compare(context.Background(), a, b, docdiff.Options{})
	if !errors.Is(err, docdiff.ErrInput) || len(r.Changes) != 0 {
		t.Fatal("malformed semantics compared", r, err)
	}
	r, err = docdiff.Compare(context.Background(), a, b, docdiff.Options{View: docdiff.Source, Strict: true})
	if !errors.Is(err, docdiff.ErrStrict) || !r.Complete || r.Status != "strict-invalid" {
		t.Fatal("strict diagnostic status", r, err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return 0, nil }

type invalidWriter struct{ err error }

func (w invalidWriter) Write(data []byte) (int, error) { return len(data) + 1, w.err }
func TestWriterContractFailures(t *testing.T) {
	r := compare(t, header, header, docdiff.Options{})
	sentinel := errors.New("underlying writer failure")
	if err := r.Write(context.Background(), invalidWriter{sentinel}, docdiff.JSON, 1024); !errors.Is(err, sentinel) {
		t.Fatal("invalid count swallowed underlying cause", err)
	}
	var buffer *bytes.Buffer
	if err := r.Write(context.Background(), buffer, docdiff.JSON, 1024); !errors.Is(err, docdiff.ErrOptions) {
		t.Fatal("typed nil writer accepted")
	}
}

func TestUnexpandedMetadataIsNotAssertedModeled(t *testing.T) {
	a := ".TH \\*[OLD] 1\n.SH DESCRIPTION\nText.\n"
	b := strings.Replace(a, "OLD", "NEW", 1)
	r := compare(t, a, b, docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Before.Kind != docdiff.Metadata || r.Changes[0].Confidence != "uninterpreted-source" || len(r.Diagnostics) == 0 {
		t.Fatal("unexpanded metadata falsely modeled", r)
	}
}
