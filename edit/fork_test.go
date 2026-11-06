package edit_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func TestWorkingForkRetainsBaselineAndIsolatesPlans(t *testing.T) {
	ctx := context.Background()
	original := ".SH NAME\nold\n"
	set, e := source.Read(ctx, strings.NewReader(original), "page.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	w, e := edit.New(set, edit.Options{})
	if e != nil {
		t.Fatal(e)
	}
	p, e := w.Plan(ctx, edit.Command{Select: query.Select(query.Paragraph), Operation: edit.FindReplaceSource, Find: "old", Text: "new"})
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Apply(p); e != nil {
		t.Fatal(e)
	}
	fork := w.Fork()
	if fork.Version() != w.Version() || string(fork.Changes()[0].Before) != original {
		t.Fatal("fork reset original baseline")
	}
	p, e = fork.Plan(ctx, edit.Command{Select: query.Select(query.Paragraph), Operation: edit.FindReplaceSource, Find: "new", Text: "fork"})
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Apply(p); e != edit.ErrStale {
		t.Fatal("foreign plan accepted", e)
	}
	if e = fork.Apply(p); e != nil {
		t.Fatal(e)
	}
	data, _ := w.Data("page.1")
	if strings.Contains(string(data), "fork") {
		t.Fatal("fork changed parent")
	}
	if e = fork.Commit(ctx, func(context.Context, edit.CommitBatch) error { return nil }); e != nil {
		t.Fatal(e)
	}
	if len(fork.Changes()) != 0 || len(w.Changes()) != 1 {
		t.Fatal("commit acknowledgement escaped fork")
	}
}
