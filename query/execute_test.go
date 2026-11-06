package query_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func environmentSet(t *testing.T) *source.Set {
	t.Helper()
	set, err := source.Load(context.Background(), fstest.MapFS{
		"tool.1":     {Data: []byte(".TH TOOL 1\n.SH ENVIRONMENT\n.so terms.roff\n.SH OTHER\n.TP\n.B OUTSIDE\nOther.\n")},
		"terms.roff": {Data: []byte(".TP\n.B TOOL_HOME\nHome directory.\n.TP\n.B TOOL_MODE\nMode.\n.de SAVED\n.TP\n.B NOT_LIVE\n..\n")},
	}, "tool.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return set
}
func TestEnvironmentQueryWithIncludedDefinitions(t *testing.T) {
	set := environmentSet(t)
	parsed, err := query.ParseSelector(`section[name="ENVIRONMENT"] definition`)
	if err != nil {
		t.Fatal(err)
	}
	builder := query.Select(query.Section).Where(query.Name, query.Equal, "ENVIRONMENT").Descendant(query.Select(query.Definition))
	first, err := query.Execute(context.Background(), set, parsed, query.Options{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := query.Execute(context.Background(), set, builder, query.Options{})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("builder and parser execute differently")
	}
	if !first.Complete || len(first.Matches) != 2 {
		t.Fatalf("wrong definition matches: %+v", first)
	}
	for i, match := range first.Matches {
		if match.Index != i+1 || match.Source.Name != "terms.roff" || match.Source.ID != 2 || match.Source.ParentEdge != 1 || len(match.Source.Digest) != 64 || !match.Source.Complete {
			t.Fatal("include provenance or ordinal lost")
		}
		if match.Raw != set.File(2).AST().Source()[match.Span.Start.Offset:match.Span.End.Offset] {
			t.Fatal("source was reconstructed")
		}
		for _, id := range match.ASTIDs {
			if set.File(2).AST().Node(id) == nil {
				t.Fatal("invented AST identity")
			}
		}
	}
	if first.Matches[0].Term != "TOOL_HOME" || !strings.Contains(first.Matches[0].NormalizedText, "home directory") || strings.Contains(first.Matches[1].Text, "NOT_LIVE") && len(first.Matches[1].Diagnostics) == 0 {
		t.Fatal("term/description or opaque warnings lost")
	}
}
func TestSyntaxAndSemanticViews(t *testing.T) {
	set := environmentSet(t)
	cases := []struct {
		selector string
		view     query.View
		count    int
	}{
		{`section[name="ENVIRONMENT"] definition > term[term="TOOL_HOME"]`, query.Semantic, 1},
		{`macro[macro=".B"][arg.0="TOOL_HOME"][source="terms.roff"][line=2]`, query.Syntax, 1},
		{`section[name="ENVIRONMENT"] macro[macro=".B"]`, query.Syntax, 2},
		{`source[include-depth>0] definition`, query.Semantic, 2},
		{`section[name="OTHER"] definition`, query.Semantic, 1},
		{`section[name="ENVIRONMENT"] > definition`, query.Semantic, 0},
		{`definition[normalized-text*="HOME   DIRECTORY"]`, query.Semantic, 1},
	}
	for _, tc := range cases {
		t.Run(tc.selector, func(t *testing.T) {
			selector, err := query.ParseSelector(tc.selector)
			if err != nil {
				t.Fatal(err)
			}
			result, err := query.Execute(context.Background(), set, selector, query.Options{View: tc.view})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Matches) != tc.count {
				t.Fatalf("got %d, want %d: %+v", len(result.Matches), tc.count, result.Matches)
			}
		})
	}
}
func TestQueryErrorsAndBudgets(t *testing.T) {
	set := environmentSet(t)
	selector := query.Select(query.Any)
	for _, o := range []query.Options{{MaxWork: 1}, {MaxNodes: 2}, {MaxMatches: 1}, {MaxResultBytes: 1}} {
		result, err := query.Execute(context.Background(), set, selector, o)
		if !errors.Is(err, query.ErrLimit) || result.Complete {
			t.Fatalf("query budget not enforced: %+v %v", o, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := query.Execute(ctx, set, selector, query.Options{})
	if !errors.Is(err, context.Canceled) || result.Complete {
		t.Fatal("cancellation lost", err)
	}
	if _, err := query.Execute(context.Background(), nil, selector, query.Options{}); err == nil {
		t.Fatal("nil source set accepted")
	}
	if _, err := query.Execute(context.Background(), set, query.Select(query.Line), query.Options{View: "made-up"}); err == nil {
		t.Fatal("invalid view accepted")
	}
}
func TestQueryRawBytesAndRecovery(t *testing.T) {
	for _, input := range []string{".SH NAME\nlast", ".SH NAME\r\n雪\r\n", ".SH NAME\ninvalid \xff\n.UNKNOWN\n", ".SH \"broken\n.de SAVED\n"} {
		set, err := source.Read(context.Background(), strings.NewReader(input), "bytes.1", source.Options{})
		if err != nil {
			t.Fatal(err)
		}
		result, err := query.Execute(context.Background(), set, query.Select(query.Line), query.Options{View: query.Syntax})
		if err != nil {
			t.Fatal(err)
		}
		var raw strings.Builder
		for _, match := range result.Matches {
			raw.WriteString(match.Raw)
			if match.Span.Start.Filename != "bytes.1" {
				t.Fatal("source attribution lost")
			}
		}
		if raw.String() != input {
			t.Fatal("raw syntax query changed bytes")
		}
	}
	set, err := source.Read(context.Background(), strings.NewReader(".SH NAME\nlong text\n"), "partial.1", source.Options{MaxBytes: 12})
	if !errors.Is(err, source.ErrLimit) {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, query.Select(query.Source), query.Options{})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Source.Complete || result.Matches[0].Raw != set.File(1).AST().Source() {
		t.Fatal("partial input facts lost", err)
	}
}

func TestDenseAncestorChainsStayBounded(t *testing.T) {
	input := ".SH ROOT\n" + strings.Repeat(".RS 2\n", 40) + ".TP\n.B TOOL_HOME\nDirectory.\n" + strings.Repeat(".RE\n", 40)
	set, err := source.Read(context.Background(), strings.NewReader(input), "deep.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, err := query.ParseSelector(strings.Repeat("indent ", 30) + "definition > term")
	if err != nil {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, selector, query.Options{MaxWork: 100000})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Term != "TOOL_HOME" {
		t.Fatal("dense ancestor combinations exceeded linear work bound", err)
	}
}

func TestRepeatedSectionsAndSubsections(t *testing.T) {
	input := ".SH ENVIRONMENT\n.TP\n.B FIRST\nOne.\n.SS DETAILS\n.TP\n.B SECOND\nTwo.\n.SH ENVIRONMENT\n.TP\n.B THIRD\nThree.\n"
	set, err := source.Read(context.Background(), strings.NewReader(input), "repeated.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, err := query.ParseSelector(`section[name="ENVIRONMENT"] definition`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, selector, query.Options{})
	if err != nil || len(result.Matches) != 3 {
		t.Fatal("repeated headings lost", err)
	}
	for i, term := range []string{"FIRST", "SECOND", "THIRD"} {
		if result.Matches[i].Index != i+1 || result.Matches[i].Term != term {
			t.Fatal("repeated heading results reordered")
		}
	}
	selector, err = query.ParseSelector(`section > subsection[name="DETAILS"] definition-list > definition > term[macro=".B"]`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = query.Execute(context.Background(), set, selector, query.Options{})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Term != "SECOND" || result.Matches[0].Raw != ".B SECOND\n" {
		t.Fatal("subsection/tag provenance changed", err)
	}
}

func TestInlineAndInvalidBytePredicates(t *testing.T) {
	set, err := source.Read(context.Background(), strings.NewReader("Font \\fBbold\\fP\ninvalid \xff\n"), "inline.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, err := query.ParseSelector(`inline[inline-kind="FontChange"][name="B"]`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, selector, query.Options{View: query.Syntax})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Raw != "\\fB" {
		t.Fatal("inline metadata or exact escape span lost", err)
	}
	selector, err = query.ParseSelector(`line[raw*="\xff"]`)
	if err != nil {
		t.Fatal(err)
	}
	result, err = query.Execute(context.Background(), set, selector, query.Options{View: query.Syntax})
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Raw != "invalid \xff\n" {
		t.Fatal("byte predicate did not match preserved invalid UTF-8", err)
	}
}

func TestIncludeWithinDefinitionKeepsLogicalMatchOrder(t *testing.T) {
	set, err := source.Load(context.Background(), fstest.MapFS{
		"main.1": {Data: []byte(".SH ENVIRONMENT\n.TP\n.so tag\nparent description\n")},
		"tag":    {Data: []byte(".TP\n.B CHILD\nchild description\n")},
	}, "main.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := query.Execute(context.Background(), set, query.Select(query.Paragraph), query.Options{})
	if err != nil || len(result.Matches) != 2 || result.Matches[0].Source.Name != "tag" || result.Matches[1].Source.Name != "main.1" {
		t.Fatal("included source moved after later parent content", err)
	}
}
