package script_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
	"github.com/d5/tengo/v2"
)

// This prototype exercises the approved engine with the actual document/query
// objects before choosing the optional CLI and persistent-session adapter.
func TestTengoDocumentPrototype(t *testing.T) {
	ctx := context.Background()
	set, e := source.Read(ctx, strings.NewReader(".SH ENVIRONMENT\n.TP\nTOOL_HOME\nOld directory.\n.TP\nPATH\nSearch paths.\n"), "tool.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	working, e := edit.New(set, edit.Options{})
	if e != nil {
		t.Fatal(e)
	}
	var emitted []string
	doc := &tengo.ImmutableMap{Value: map[string]tengo.Object{
		"query": &tengo.UserFunction{Name: "doc.query", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return nil, errors.New("selector must be string")
			}
			selector, e := query.ParseSelector(s.Value)
			if e != nil {
				return nil, e
			}
			result, e := working.Query(ctx, selector, query.Options{})
			if e != nil {
				return nil, e
			}
			matches := make([]tengo.Object, 0, len(result.Matches))
			for _, match := range result.Matches {
				matches = append(matches, &tengo.ImmutableMap{Value: map[string]tengo.Object{"term": &tengo.String{Value: match.Term}, "text": &tengo.String{Value: match.Text}, "source": &tengo.String{Value: match.Source.Name}, "index": &tengo.Int{Value: int64(match.Index)}}})
			}
			return &tengo.ImmutableArray{Value: matches}, nil
		}},
		"replace_description": &tengo.UserFunction{Name: "doc.replace_description", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			selector, ok := args[0].(*tengo.String)
			if !ok {
				return nil, errors.New("selector must be string")
			}
			text, ok := args[1].(*tengo.String)
			if !ok {
				return nil, errors.New("text must be string")
			}
			q, e := query.ParseSelector(selector.Value)
			if e != nil {
				return nil, e
			}
			p, e := working.Plan(ctx, edit.Command{Select: q, Operation: edit.ReplaceDescription, Text: text.Value})
			if e != nil {
				return nil, e
			}
			if e = working.Apply(p); e != nil {
				return nil, e
			}
			return tengo.UndefinedValue, nil
		}},
		"emit": &tengo.UserFunction{Name: "doc.emit", Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}
			s, ok := args[0].(*tengo.String)
			if !ok {
				return nil, errors.New("emission must be string")
			}
			emitted = append(emitted, s.Value)
			return tengo.UndefinedValue, nil
		}},
	}}
	code := `
 each := func(values, fn) { for value in values { fn(value) } }
 matches := doc.query("section[name=ENVIRONMENT] definition")
 each(matches, func(match) {
  if match.term == "TOOL_HOME" {
   doc.emit(match.term + ": " + match.source)
   doc.replace_description("definition[term=TOOL_HOME]", "New directory.")
  }
 })
 updated := doc.query("definition[term=TOOL_HOME]")
 doc.emit(updated[0].text)
 `
	program := tengo.NewScript([]byte(code))
	program.SetImports(tengo.NewModuleMap())
	program.EnableFileImport(false)
	program.SetMaxAllocs(10000)
	program.SetMaxConstObjects(4096)
	if e = program.Add("doc", doc); e != nil {
		t.Fatal(e)
	}
	compiled, e := program.RunContext(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(emitted) != 2 || emitted[0] != "TOOL_HOME: tool.1" || !strings.Contains(emitted[1], "New directory.") {
		t.Fatal(emitted)
	}
	if len(working.Changes()) != 1 {
		t.Fatal("missing in-memory change")
	}
	t.Logf("compiled globals %d; emitted %d values", len(compiled.GetAll()), len(emitted))
}
func TestTengoResourceAndCapabilityPrototype(t *testing.T) {
	for _, name := range []string{"os", "net", "fmt", "json", "../source"} {
		program := tengo.NewScript([]byte(`mod := import("` + name + `")`))
		program.EnableFileImport(false)
		program.SetImports(tengo.NewModuleMap())
		if _, e := program.Run(); e == nil {
			t.Fatal("unexpected module", name)
		}
	}
	if runtime.GOOS != "js" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		if _, e := tengo.NewScript([]byte(`for {}`)).RunContext(ctx); !errors.Is(e, context.DeadlineExceeded) {
			t.Fatal("infinite loop cancellation", e)
		}
	}
	program := tengo.NewScript([]byte(`x := 0; for { x++ }`))
	program.SetMaxAllocs(100)
	if _, e := program.Run(); !errors.Is(e, tengo.ErrObjectAllocLimit) {
		t.Fatal("allocation budget", e)
	}
}
