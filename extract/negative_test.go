package extract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func TestSnapshotIdentityAndSpanValidation(t *testing.T) {
	set := read(t, example)
	original, err := query.Execute(context.Background(), set, query.Select(query.Definition), query.Options{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(original)
	for _, mutate := range []func(*query.Match){func(m *query.Match) { m.Span.Start.Offset++ }, func(m *query.Match) { m.SemanticID = 9999 }, func(m *query.Match) { m.ASTIDs[0] = 9999 }, func(m *query.Match) { m.Source.Name = "other.1" }} {
		var selected query.Result
		if err := json.Unmarshal(data, &selected); err != nil {
			t.Fatal(err)
		}
		mutate(&selected.Matches[0])
		r, err := extract.FromQuery(context.Background(), set, &selected, extract.Options{})
		if !errors.Is(err, extract.ErrSnapshot) || r.Complete || len(r.Records) != 0 {
			t.Fatal("foreign identity accepted", r, err)
		}
	}
}
func TestContributedExporterCannotHideWriterFailures(t *testing.T) {
	r := selectRecords(t, example, "definition", extract.Options{})
	ignore := extract.ExporterFunc(func(ctx context.Context, w io.Writer, r *extract.Report) error {
		_, _ = io.WriteString(w, "first")
		_, _ = io.WriteString(w, "second")
		return nil
	})
	var out bytes.Buffer
	if err := r.Export(context.Background(), &out, ignore, 1); !errors.Is(err, extract.ErrLimit) || out.Len() != 0 {
		t.Fatal("ignored budget failure vanished", err)
	}
	if err := r.Export(context.Background(), shortWriter{}, ignore, 1024); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal("ignored short write vanished", err)
	}
	var nilExporter extract.ExporterFunc
	if err := r.Export(context.Background(), io.Discard, nilExporter, 1024); !errors.Is(err, extract.ErrOptions) {
		t.Fatal("typed nil exporter accepted")
	}
	for _, format := range []extract.Format{extract.JSON, extract.JSONL, extract.Text} {
		out.Reset()
		if err := r.Write(context.Background(), &out, format, extract.WriteOptions{Fields: []string{"term"}}); !errors.Is(err, extract.ErrOptions) || out.Len() != 0 {
			t.Fatal("incompatible fields ignored")
		}
	}
}
func FuzzSourceBackedRecords(f *testing.F) {
	for _, s := range []string{example, ".SH EMPTY\n", ".TP\n.B HOME\nValue \\fBbold\\fP.\n", ".de SAVED\n.SH STORED\n..\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 4096 {
			t.Skip()
		}
		set, _ := source.Read(context.Background(), strings.NewReader(text), "fuzz.1", source.Options{MaxBytes: 4096, MaxNodes: 1024})
		if set == nil {
			return
		}
		before := set.File(1).AST().Source()
		r, _ := extract.Select(context.Background(), set, query.Select(query.Any), extract.Options{MaxNodes: 4096, MaxRecords: 32, MaxBytes: 65536, MaxWork: 100000})
		for i, record := range r.Records {
			if record.Index != i+1 || record.Raw != before[record.Span.Start.Offset:record.Span.End.Offset] {
				t.Fatal("record provenance changed")
			}
		}
		if set.File(1).AST().Source() != before {
			t.Fatal("source mutated")
		}
	})
}
func BenchmarkSelect(b *testing.B) {
	for _, n := range []int{10, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var text strings.Builder
			text.WriteString(".TH TOOL 1\n.SH ENVIRONMENT\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&text, ".TP\n.B TOOL_VALUE_%d\nDescription.\n", i)
			}
			set, err := source.Read(context.Background(), strings.NewReader(text.String()), "bench.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			selector := query.Select(query.Section).Where(query.Name, query.Equal, "ENVIRONMENT").Descendant(query.Select(query.Definition))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r, err := extract.Select(context.Background(), set, selector, extract.Options{})
				if err != nil || len(r.Records) != n {
					b.Fatal("benchmark extraction", err)
				}
			}
		})
	}
}

func TestStoredDefinitionsAndEscapedContent(t *testing.T) {
	r := selectRecords(t, ".de SAVED\n.TP\n.B HIDDEN\nStored.\n..\n"+example, "definition", extract.Options{})
	if len(r.Records) != 3 {
		t.Fatal("stored definition became live", r)
	}
	r = selectRecords(t, ".TP\n.B \"HOME PATH\"\nA \\fBbold\\fP value and \\[unmodeled].\n", "definition", extract.Options{})
	if r.Records[0].Term != "HOME PATH" || !strings.Contains(r.Records[0].Description, "bold value") || !r.Records[0].Unsupported || !strings.Contains(r.Records[0].Raw, `\[unmodeled]`) {
		t.Fatal("quoted/unsupported content lost", r.Records)
	}
}

func TestExtractionWorkAccountingForDiff(t *testing.T) {
	set := read(t, example)
	report, err := extract.Select(context.Background(), set, query.Select(query.Document), extract.Options{MaxWork: 100000})
	if err != nil || report.WorkUnits <= 0 || report.WorkUnits > 100000 {
		t.Fatal("extraction work accounting", report.WorkUnits, err)
	}
	data, err := json.Marshal(report)
	if err != nil || strings.Contains(string(data), "workUnits") {
		t.Fatal("execution accounting changed JSON")
	}
	bounded, err := extract.Select(context.Background(), set, query.Select(query.Document), extract.Options{MaxWork: report.WorkUnits - 1})
	if err == nil || bounded.WorkUnits > report.WorkUnits-1 {
		t.Fatal("composed extraction budget", err)
	}
}
