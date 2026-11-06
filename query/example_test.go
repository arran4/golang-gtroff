package query_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleExecute() {
	input := ".SH ENVIRONMENT\n.TP\n.B TOOL_HOME\nDirectory for tool data.\n.TP\n.B TOOL_MODE\nOutput mode.\n"
	set, err := source.Read(context.Background(), strings.NewReader(input), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	selector, err := query.ParseSelector(`section[name="ENVIRONMENT"] definition`)
	if err != nil {
		panic(err)
	}
	result, err := query.Execute(context.Background(), set, selector, query.Options{})
	if err != nil {
		panic(err)
	}
	for _, match := range result.Matches {
		fmt.Printf("%d %s at %s:%d\n", match.Index, match.Term, match.Source.Name, match.Span.Start.Line)
	}
	// Output:
	// 1 TOOL_HOME at tool.1:2
	// 2 TOOL_MODE at tool.1:5
}
func ExampleSelect() {
	// Builders and parsed selectors compile to the same immutable representation.
	selector := query.Select(query.Section).Where(query.Name, query.Equal, "ENVIRONMENT").Descendant(query.Select(query.Definition).Where(query.TermText, query.Prefix, "TOOL_"))
	fmt.Println(selector.Validate())
	// Output: <nil>
}
func ExampleResult_Write() {
	set, err := source.Read(context.Background(), strings.NewReader(".SH NAME\ntool\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	result, err := query.Execute(context.Background(), set, query.Select(query.Section), query.Options{})
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	if err := result.Write(&out, query.CSV, []string{"index", "file", "line", "name"}); err != nil {
		panic(err)
	}
	fmt.Print(out.String())
	// Output:
	// index,file,line,name
	// 1,tool.1,1,NAME
}
