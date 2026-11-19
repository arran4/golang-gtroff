package source_test

import (
	"context"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/source"
)

func ExampleLoad() {
	fsys := fstest.MapFS{
		"tool.1":     {Data: []byte(".SH ENVIRONMENT\n.so terms.roff\n")},
		"terms.roff": {Data: []byte(".TP\n.B TOOL_HOME\nDirectory.\n")},
	}
	set, err := source.Load(context.Background(), fsys, "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	for _, edge := range set.Edges() {
		fmt.Printf("%s:%d includes %s (%s)\n", set.File(edge.From).Name(), edge.Span.Start.Line, set.File(edge.To).Name(), edge.State)
	}
	// Each logical line points into its original immutable, file-local AST.
	for _, line := range set.Lines() {
		node := set.Node(line)
		if node.Attributes().Name == ".B" {
			fmt.Printf("term at %s:%d: %s", node.Span().Start.Filename, node.Span().Start.Line, node.Text())
		}
	}
	// Output:
	// tool.1:2 includes terms.roff (resolved)
	// term at terms.roff:2: TOOL_HOME
}

func ExampleRead() {
	// No explicit filesystem: a reader cannot acquire ambient filesystem access.
	set, err := source.Read(context.Background(), strings.NewReader(".so local.roff\n"), "<stdin>", source.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(set.Edges()[0].State)
	fmt.Print(set.File(1).AST().Source())
	// Output:
	// unresolved
	// .so local.roff
}
