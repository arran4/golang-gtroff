package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleASTNode_Token() {
	source := ".SH ENVIRONMENT\n.TP\n.B TOOL_HOME\nDirectory.\n.de SAVED\n.SH STORED\n..\n.MYSTERY value\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	fmt.Println("root:", tree.Root().Kind())
	section := tree.Root().FindSection("ENVIRONMENT", false)[0]
	fmt.Println("tag openers:", len(section.FindMacro("TP")))
	for _, node := range section.FindMacro("B") {
		token, err := node.Token()
		if err != nil {
			panic(err)
		}
		// Typed tokens are copies; Raw and the span still identify original bytes.
		fmt.Printf("typed: %T\n", token)
		span := node.Span()
		fmt.Println("source slice:", tree.Source()[span.Start.Offset:span.End.Offset] == node.Raw())
	}
	capture := section.FindKind(groff.ASTBlock)[0]
	fmt.Println("opaque:", capture.Attributes().Opaque)
	fmt.Println("live stored headings:", len(tree.Root().FindSection("STORED", false)))
	unknown := section.FindMacro("MYSTERY")[0]
	fmt.Printf("unknown retained: %s", unknown.Raw())
	// Output:
	// root: document
	// tag openers: 1
	// typed: *groff.TokenManB
	// source slice: true
	// opaque: true
	// live stored headings: 0
	// unknown retained: .MYSTERY value
}
