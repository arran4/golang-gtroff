package edit_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleWorking_Plan() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".SH ENVIRONMENT\n.TP\nTOOL_HOME\nOld directory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	working, err := edit.New(set, edit.Options{Strict: true})
	if err != nil {
		panic(err)
	}
	selected := query.Select(query.Section).Where(query.Name, query.Equal, "ENVIRONMENT").Descendant(query.Select(query.Definition).Where(query.TermText, query.Equal, "TOOL_HOME"))
	plan, err := working.Plan(ctx, edit.Command{Select: selected, Operation: edit.ReplaceDescription, Text: "New directory."})
	if err != nil {
		panic(err)
	}
	fmt.Println("original:", strings.Contains(working.Sources().File(1).AST().Source(), "Old directory."))
	if err = working.Apply(plan); err != nil {
		panic(err)
	}
	report, err := working.Query(ctx, selected, query.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.Join(strings.Fields(report.Matches[0].Text), " "))
	fmt.Println("changed files:", len(working.Changes()))
	// Output:
	// original: true
	// TOOL_HOME New directory.
	// changed files: 1
}
func ExampleWorking_PlanPatches() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader("old\r\n"), "page.1", source.Options{})
	if err != nil {
		panic(err)
	}
	w, err := edit.New(set, edit.Options{})
	if err != nil {
		panic(err)
	}
	matches, err := w.Query(ctx, edit.Root(), query.Options{})
	if err != nil {
		panic(err)
	}
	plan, err := w.PlanPatches(ctx, []edit.Patch{{Source: "page.1", Digest: matches.Matches[0].Source.Digest, Start: 0, End: 3, Data: []byte("new")}})
	if err != nil {
		panic(err)
	}
	if err = w.Apply(plan); err != nil {
		panic(err)
	}
	b, err := w.Data("page.1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%q\n", b)
	// Output: "new\r\n"
}
func ExampleCreate() {
	ctx := context.Background()
	w, err := edit.Create(ctx, "new.1", man.Metadata{Title: "NEW", Section: "1"}, edit.Options{})
	if err != nil {
		panic(err)
	}
	p, err := w.Plan(ctx, edit.Command{Select: edit.Root(), Operation: edit.AppendSection, Heading: "NAME", Text: "new - demonstration"})
	if err != nil {
		panic(err)
	}
	if err = w.Apply(p); err != nil {
		panic(err)
	}
	result, err := w.Query(ctx, query.Select(query.Section), query.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Matches[0].Name)
	// Output: NAME
}
func ExampleWorking_Commit() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader("old\n"), "page.1", source.Options{})
	if err != nil {
		panic(err)
	}
	w, err := edit.New(set, edit.Options{})
	if err != nil {
		panic(err)
	}
	p, err := w.Plan(ctx, edit.Command{Select: edit.Root(), Operation: edit.FindReplaceSource, Find: "old", Text: "new"})
	if err != nil {
		panic(err)
	}
	if err = w.Apply(p); err != nil {
		panic(err)
	}
	// An application supplies its own explicitly authorized storage boundary.
	err = w.Commit(ctx, func(ctx context.Context, b edit.CommitBatch) error {
		fmt.Println("guards:", len(b.Guards), "changes:", len(b.Changes))
		fmt.Printf("%q\n", b.Changes[0].After)
		return nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("pending:", len(w.Changes()))
	// Output:
	// guards: 1 changes: 1
	// "new\n"
	// pending: 0
}

func ExampleWorking_Fork() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".SH NAME\nold\n"), "page.1", source.Options{})
	if err != nil {
		panic(err)
	}
	working, err := edit.New(set, edit.Options{})
	if err != nil {
		panic(err)
	}
	branch := working.Fork()
	plan, err := branch.Plan(ctx, edit.Command{Select: query.Select(query.Paragraph), Operation: edit.FindReplaceSource, Find: "old", Text: "new"})
	if err != nil {
		panic(err)
	}
	if err = branch.Apply(plan); err != nil {
		panic(err)
	}
	original, _ := working.Data("page.1")
	fmt.Println(strings.Contains(string(original), "old"), len(branch.Changes()))
	fmt.Println(branch.Guards()[0].Name, branch.Guards()[0].Bytes)
	// Output:
	// true 1
	// page.1 13
}
