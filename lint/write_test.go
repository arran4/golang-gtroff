package lint_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/lint"
	"github.com/arran4/golang-gtroff/source"
)

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return 0, nil }

type failingWriter struct{ err error }

func (w failingWriter) Write(p []byte) (int, error) { return 0, w.err }
func TestReportSerializationAndFailures(t *testing.T) {
	r := run(t, ".TH TOOL 1\n", lint.Options{})
	for _, format := range []lint.Format{lint.Text, lint.JSON, lint.JSONL} {
		var out bytes.Buffer
		if err := r.Write(&out, format); err != nil {
			t.Fatal(err)
		}
		if out.Len() == 0 {
			t.Fatal("empty output")
		}
		if err := r.Write(shortWriter{}, format); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal("short writer swallowed", err)
		}
		sentinel := errors.New("writer failed")
		if err := r.Write(failingWriter{sentinel}, format); !errors.Is(err, sentinel) {
			t.Fatal("writer failure lost", err)
		}
		if format == lint.JSONL {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			if len(lines) != len(r.Findings)+1 {
				t.Fatal("jsonl records")
			}
			for _, line := range lines {
				if !json.Valid([]byte(line)) {
					t.Fatal("invalid jsonl")
				}
			}
		}
	}
	if err := r.Write(io.Discard, "bogus"); err == nil {
		t.Fatal("invalid format")
	}
}
func TestSubsectionsDoNotSatisfyRequiredSections(t *testing.T) {
	r := run(t, ".TH TOOL 1\n.SS NAME\ntool \\- description\n", lint.Options{})
	if len(findings(r, "man.required-section")) != 3 {
		t.Fatal("subsection became required section", r.Findings)
	}
}
func TestSourceBytesAndCumulativeBudgets(t *testing.T) {
	text := strings.ReplaceAll(good, "\n", "\r\n") + "comment \xff\n"
	d := document(t, text)
	before := d.Sources.File(1).AST().Source()
	r, err := lint.Run(context.Background(), []lint.Document{d}, lint.Options{})
	if err != nil || !r.Complete || d.Sources.File(1).AST().Source() != before || before != text {
		t.Fatal("source changed", err)
	}
	nodes := 0
	for _, s := range r.Documents[0].Sources {
		nodes += s.Nodes
	}
	r, err = lint.Run(context.Background(), []lint.Document{d, d}, lint.Options{MaxNodes: nodes})
	if !errors.Is(err, lint.ErrLimit) || r.Complete {
		t.Fatal("node budget reset for second document")
	}
	r, err = lint.Run(context.Background(), []lint.Document{d, d}, lint.Options{MaxWork: 100})
	if !errors.Is(err, lint.ErrLimit) || r.Complete {
		t.Fatal("work budget not enforced")
	}
}
func FuzzParsePolicy(f *testing.F) {
	for _, s := range []string{`{"schemaVersion":1}`, `{"schemaVersion":1,"policy":{"requiredSections":[]}}`, `{"schemaVersion":1,"policy":{"disable":["unknown"]}}`, `null`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		policy, err := lint.ParsePolicy(strings.NewReader(s))
		if err == nil {
			_ = lint.ValidatePolicy(policy, nil)
		}
	})
}
func BenchmarkRun(b *testing.B) {
	for _, n := range []int{10, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var text strings.Builder
			text.WriteString(good + ".SH OPTIONS\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&text, ".TP\n.B --option-%d\nDescription.\n", i)
			}
			set, err := source.Read(context.Background(), strings.NewReader(text.String()), "bench.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			docs := []lint.Document{{Name: "bench.1", Sources: set}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r, err := lint.Run(context.Background(), docs, lint.Options{})
				if err != nil || r.Failed() {
					b.Fatal(r, err)
				}
			}
		})
	}
}

type lintErrorReader struct{ data []byte }

func (r *lintErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, errors.New("input interrupted")
}
func TestIncompleteSourceFailsWithoutCallerErrorLabel(t *testing.T) {
	set, loadErr := source.Read(context.Background(), &lintErrorReader{[]byte(good)}, "partial.1", source.Options{})
	if loadErr == nil || set == nil {
		t.Fatal("partial fixture failed")
	}
	r, err := lint.Run(context.Background(), []lint.Document{{Name: "partial.1", Sources: set}}, lint.Options{FailOn: lint.Never})
	if err != nil || !r.Failed() || !r.Documents[0].InputInvalid || r.Documents[0].InputError == "" {
		t.Fatal("incomplete source became valid", r, err)
	}
}
func TestReporterRejectsForeignIdentity(t *testing.T) {
	for _, mutate := range []func(*lint.Finding){func(f *lint.Finding) { f.Rule = "another.rule" }, func(f *lint.Finding) { f.Source = 999 }, func(f *lint.Finding) { f.Node = 999 }, func(f *lint.Finding) { f.Span.Start.Filename = "other.1" }, func(f *lint.Finding) { f.Severity = "bogus" }} {
		registry, err := lint.NewRegistry(lint.Rule{Info: lint.RuleInfo{ID: "project.invalid", Severity: lint.Info, Category: lint.Heuristic, Description: "invalid fixture"}, Check: func(ctx context.Context, c *lint.Context, r lint.Reporter) error {
			f := lint.Finding{Source: 1, Span: c.Sources().File(1).AST().Root().Span(), Message: "fixture"}
			mutate(&f)
			return r.Emit(f)
		}})
		if err != nil {
			t.Fatal(err)
		}
		r, err := lint.Run(context.Background(), []lint.Document{document(t, good)}, lint.Options{Registry: registry})
		if !errors.Is(err, lint.ErrConfig) || r.Complete || !r.Failed() {
			t.Fatal("invalid callback finding accepted", r, err)
		}
	}
}
