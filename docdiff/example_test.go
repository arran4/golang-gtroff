package docdiff_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arran4/golang-gtroff/docdiff"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleCompare() {
	ctx := context.Background()
	load := func(name, text string) docdiff.Input {
		set, err := source.Read(ctx, strings.NewReader(text), name, source.Options{})
		if err != nil {
			panic(err)
		}
		return docdiff.Input{Name: name, Sources: set}
	}
	before := load("before.1", ".TH TOOL 1\n.SH ENVIRONMENT\n.TP\n.B HOME\nOld directory.\n")
	after := load("after.1", ".TH TOOL 1\n.SH ENVIRONMENT\n.TP\n.B HOME\nNew directory.\n")
	report, err := docdiff.Compare(ctx, before, after, docdiff.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(report.Status)
	for _, change := range report.Changes {
		fmt.Printf("%s %s at %s:%d -> %s:%d\n", change.Kind, change.After.Identity, change.Before.Location.Source.Name, change.Before.Location.Span.Start.Line, change.After.Location.Source.Name, change.After.Location.Span.Start.Line)
	}
	// Output:
	// different
	// changed HOME at before.1:3 -> after.1:3
}
func ExampleReport_Write() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".TH TOOL 1\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := docdiff.Compare(ctx, docdiff.Input{Name: "before", Sources: set}, docdiff.Input{Name: "after", Sources: set}, docdiff.Options{View: docdiff.Source})
	if err != nil {
		panic(err)
	}
	if err := report.Write(ctx, os.Stdout, docdiff.Text, 1024); err != nil {
		panic(err)
	}
	// Output:
	// diff: equal (assessment=exact-source-bytes complete=true changes=0 ambiguities=0)
	// before: "before" sources=1 objects=1
	// after: "after" sources=1 objects=1
}
