package groff_test

import (
	"context"
	"embed"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

//go:embed testdata/environ_snippet.7
var queryRealCases embed.FS

func TestQueryRealEnvironSnippet(t *testing.T) {
	input, err := queryRealCases.ReadFile("testdata/environ_snippet.7")
	if err != nil {
		t.Fatal(err)
	}
	set, err := source.Read(context.Background(), strings.NewReader(string(input)), "environ_snippet.7", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, err := query.ParseSelector(`section[name="DESCRIPTION"] paragraph[normalized-text*="environment variable"]`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, selector, query.Options{})
	if err != nil || len(result.Matches) == 0 {
		t.Fatal("real environ content not found", err)
	}
	for _, match := range result.Matches {
		if match.Source.Name != "environ_snippet.7" || match.Raw != string(input[match.Span.Start.Offset:match.Span.End.Offset]) {
			t.Fatal("real fixture provenance changed")
		}
	}
	selector, err = query.ParseSelector(`macro[macro=".BR"][arg.0="getenv"]`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = query.Execute(context.Background(), set, selector, query.Options{View: query.Syntax})
	if err != nil || len(result.Matches) != 1 || !strings.Contains(result.Matches[0].Raw, "getenv") {
		t.Fatal("real syntax reference not found", err)
	}
}
