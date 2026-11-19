package extract_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleSelect() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".SH ENVIRONMENT\n.TP\n.B TOOL_HOME\nHome directory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	selector := query.Select(query.Section).Where(query.Name, query.Equal, "ENVIRONMENT").Descendant(query.Select(query.Definition))
	report, err := extract.Select(ctx, set, selector, extract.Options{})
	if err != nil {
		panic(err)
	}
	for _, record := range report.Records {
		fmt.Printf("%s: %s (%s:%d)\n", record.Term, strings.TrimSpace(record.Description), record.Source.Name, record.Span.Start.Line)
	}
	// Output: TOOL_HOME: Home directory. (tool.1:2)
}
func ExampleFromQuery() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".SH ENVIRONMENT\n.TP\n.B HOME\nHome directory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	selected, err := query.Execute(ctx, set, query.Select(query.Definition), query.Options{})
	if err != nil {
		panic(err)
	}
	report, err := extract.FromQuery(ctx, set, selected, extract.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(report.Records[0].QueryIndex, report.Records[0].Term)
	// Output: 1 HOME
}
func ExampleReport_Write() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".TP\n.B HOME\nHome directory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := extract.Select(ctx, set, query.Select(query.Definition), extract.Options{})
	if err != nil {
		panic(err)
	}
	if err := report.Write(ctx, os.Stdout, extract.CSV, extract.WriteOptions{Fields: []string{"term", "description", "file", "line"}}); err != nil {
		panic(err)
	}
	// Output:
	// term,description,file,line
	// HOME,"Home directory.
	// ",tool.1,1
}
func ExampleReport_Export() {
	report := &extract.Report{SchemaVersion: 1, Command: "extract", Complete: true, QueryComplete: true, Records: []extract.Record{{Term: "HOME", Description: "Home directory."}}}
	exporter := extract.ExporterFunc(func(ctx context.Context, w io.Writer, r *extract.Report) error {
		for _, record := range r.Records {
			if _, err := fmt.Fprintf(w, "%s: %s\n", record.Term, record.Description); err != nil {
				return err
			}
		}
		return nil
	})
	if err := report.Export(context.Background(), os.Stdout, exporter, 1024); err != nil {
		panic(err)
	}
	// Output: HOME: Home directory.
}
