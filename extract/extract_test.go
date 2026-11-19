package extract_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

const example = ".TH TOOL 1\n.SH ENVIRONMENT\n.TP\n.B TOOL_HOME\nFirst directory.\n.TP\n.B TOOL_HOME\nSecond directory.\n.SS EXTRA\n.TP\n.B TOOL_PATH\nSearch path.\n"

func read(t *testing.T, text string) *source.Set {
	t.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(text), "tool.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return set
}
func selectRecords(t *testing.T, text, expression string, o extract.Options) *extract.Report {
	t.Helper()
	selector, err := query.ParseSelector(expression)
	if err != nil {
		t.Fatal(err)
	}
	r, err := extract.Select(context.Background(), read(t, text), selector, o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestDefinitionRecordsAndRepeatedTerms(t *testing.T) {
	r := selectRecords(t, example, `section[name="ENVIRONMENT"] definition`, extract.Options{})
	if !r.Complete || !r.QueryComplete || len(r.Records) != 3 {
		t.Fatal("record selection", r)
	}
	for i, record := range r.Records {
		if record.Index != i+1 || record.QueryIndex != i+1 || !record.Semantic || record.Metadata.Title != "TOOL" || record.Context == nil || record.Source.Digest == "" || record.Unsupported || len(record.Blocks) < 2 || len(record.TermRuns) == 0 {
			t.Fatalf("missing semantics %+v", record)
		}
	}
	if r.Records[0].Term != "TOOL_HOME" || r.Records[1].Term != "TOOL_HOME" || r.Records[0].Description != "First directory.\n" || r.Records[0].Text != "TOOL_HOME\nFirst directory.\n" || r.Records[0].NormalizedText != "TOOL_HOME First directory." || len(r.Records[2].Headings) != 2 {
		t.Fatal("term/body/heading separation", r.Records)
	}
	if r.Records[0].Raw != ".TP\n.B TOOL_HOME\nFirst directory.\n" {
		t.Fatal("raw not exact")
	}
	r.Records[0].Metadata.Title = "CHANGED"
	r.Records[0].Blocks[0].Tag[0].Text = "CHANGED"
	if r.Records[1].Metadata.Title != "TOOL" || r.Records[1].TermRuns[0].Text != "TOOL_HOME" {
		t.Fatal("record mutable data aliases")
	}
}
func TestSyntaxAndFilteredQueryResults(t *testing.T) {
	set := read(t, example)
	selected, err := query.Execute(context.Background(), set, query.Select(query.Definition), query.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selected.Matches = selected.Matches[1:2]
	selected.Matches[0].Text = "forged text"
	selected.Matches[0].Raw = "forged raw"
	selected.Matches[0].Term = "forged term"
	r, err := extract.FromQuery(context.Background(), set, selected, extract.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Records[0].Index != 1 || r.Records[0].QueryIndex != 2 || r.Records[0].Term != "TOOL_HOME" || strings.Contains(r.Records[0].Raw, "forged") {
		t.Fatal("caller text was trusted", r)
	}
	r = selectRecords(t, example, `macro[macro=".B"]`, extract.Options{Query: query.Options{View: query.Syntax}})
	if len(r.Records) != 3 || r.Records[0].Semantic || r.Records[0].Text != "TOOL_HOME\n" || r.Records[0].Raw != ".B TOOL_HOME\n" {
		t.Fatal("syntax extraction changed contract", r)
	}
	selected.Matches[0].Source.Digest = "stale"
	if _, err := extract.FromQuery(context.Background(), set, selected, extract.Options{}); !errors.Is(err, extract.ErrSnapshot) {
		t.Fatal("stale source accepted", err)
	}
}
func TestIncludesAndExactBytes(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".TH TOOL 1\r\n.SH ENVIRONMENT\r\n.so terms.roff\r\n")}, "terms.roff": {Data: []byte(".TP\r\n.B BYTE\r\nInvalid \xff byte.\r\n")}}
	set, err := source.Load(context.Background(), fsys, "main.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, _ := query.ParseSelector(`section[name="ENVIRONMENT"] definition`)
	r, err := extract.Select(context.Background(), set, selector, extract.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Records) != 1 || len(r.Records[0].Headings) != 1 || r.Records[0].Headings[0].Source != 1 || r.Records[0].Headings[0].Name != "ENVIRONMENT" || r.Records[0].Source.ID != 2 || r.Records[0].Span.Start.Filename != "terms.roff" || r.Records[0].Raw != string(fsys["terms.roff"].Data) {
		t.Fatal("include provenance/raw lost", r)
	}
	var out bytes.Buffer
	if err := r.Write(context.Background(), &out, extract.Raw, extract.WriteOptions{}); err != nil || !bytes.Equal(out.Bytes(), fsys["terms.roff"].Data) {
		t.Fatal("raw bytes changed", err)
	}
	out.Reset()
	if err := r.Write(context.Background(), &out, extract.JSON, extract.WriteOptions{}); err != nil || !json.Valid(out.Bytes()) {
		t.Fatal("invalid json", err)
	}
}
func TestUnsupportedStrictAndPartial(t *testing.T) {
	text := example + ".UNKNOWN literal\n"
	r := selectRecords(t, text, "section", extract.Options{})
	if !r.Records[0].Unsupported || len(r.Diagnostics) == 0 || !strings.Contains(r.Records[0].Raw, ".UNKNOWN") {
		t.Fatal("unsupported content dropped", r)
	}
	selector := query.Select(query.Section)
	r, err := extract.Select(context.Background(), read(t, text), selector, extract.Options{Strict: true})
	if !errors.Is(err, extract.ErrStrict) || !r.Complete || len(r.Records) != 1 {
		t.Fatal("strict lost diagnostic/data", r, err)
	}
	selected, err := query.Execute(context.Background(), read(t, example), query.Select(query.Definition), query.Options{MaxMatches: 1})
	if !errors.Is(err, query.ErrLimit) {
		t.Fatal(err)
	}
	r, err = extract.FromQuery(context.Background(), read(t, example), selected, extract.Options{})
	if !errors.Is(err, extract.ErrIncomplete) || r.QueryComplete || len(r.Records) != 1 {
		t.Fatal("partial query hidden", r, err)
	}
}
func TestHTMLSelectionAndVirtualTerm(t *testing.T) {
	for _, expression := range []string{"definition", "term", "section"} {
		r := selectRecords(t, example, expression, extract.Options{HTML: true})
		var out bytes.Buffer
		if err := r.Write(context.Background(), &out, extract.HTML, extract.WriteOptions{}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "TOOL_HOME") {
			t.Fatal("html missing selection")
		}
		if expression == "term" && strings.Contains(out.String(), "First directory") {
			t.Fatal("term rendering added unselected description")
		}
	}
	r := selectRecords(t, example, "definition", extract.Options{})
	if err := r.Write(context.Background(), io.Discard, extract.HTML, extract.WriteOptions{}); !errors.Is(err, extract.ErrOptions) {
		t.Fatal("unprepared html accepted")
	}
}
func TestSerializersAndBounds(t *testing.T) {
	r := selectRecords(t, example, "definition", extract.Options{})
	for _, format := range []extract.Format{extract.JSON, extract.JSONL, extract.Text, extract.Raw, extract.Grep, extract.CSV, extract.TSV} {
		var out bytes.Buffer
		if err := r.Write(context.Background(), &out, format, extract.WriteOptions{}); err != nil {
			t.Fatal(err)
		}
		if format == extract.JSONL {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			if len(lines) != 4 {
				t.Fatal("missing jsonl summary")
			}
			for _, line := range lines {
				if !json.Valid([]byte(line)) {
					t.Fatal("invalid record")
				}
			}
		}
		if format == extract.CSV {
			rows, err := csv.NewReader(&out).ReadAll()
			if err != nil || len(rows) != 4 || rows[1][4] != "TOOL_HOME" || rows[1][5] != "First directory.\n" {
				t.Fatal("csv field separation", rows, err)
			}
		}
		if err := r.Write(context.Background(), io.Discard, format, extract.WriteOptions{MaxBytes: 1}); !errors.Is(err, extract.ErrLimit) {
			t.Fatal("output budget", err)
		}
		if err := r.Write(context.Background(), shortWriter{}, format, extract.WriteOptions{}); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal("short writer lost", err)
		}
	}
	selector := query.Select(query.Definition)
	for _, options := range []extract.Options{{MaxNodes: 1}, {MaxWork: 1}, {MaxBytes: 1}, {MaxRecords: 1}} {
		report, err := extract.Select(context.Background(), read(t, example), selector, options)
		if err == nil || report.Complete && report.QueryComplete {
			t.Fatal("limit accepted", report, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := extract.Select(ctx, read(t, example), selector, extract.Options{}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel lost", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return 0, nil }
