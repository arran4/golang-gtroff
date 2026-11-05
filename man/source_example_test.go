package man_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func ExampleDocument_SourceNodes() {
	source := ".TH TOOL 1\n.SH ENVIRONMENT\n.\\\" retained comment\n.TP\n.B TOOL_HOME\nDirectory for tool data.\n.so local.roff\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"), groff.WithSourceName("tool.1"))
	if err != nil {
		panic(err)
	}
	document, err := man.Build(tree, man.Options{})
	if err != nil {
		panic(err)
	}
	section, err := document.SelectSection("ENVIRONMENT", false)
	if err != nil {
		panic(err)
	}
	// Item tags are terms; the item's child paragraphs hold its description.
	for _, node := range section.Nodes {
		if node.Kind == man.Item {
			for _, run := range node.Tag {
				fmt.Printf("term: %s\n", run.Text)
			}
		}
	}
	lines, err := document.SourceNodes(section.Roots[0])
	if err != nil {
		panic(err)
	}
	for _, line := range lines {
		switch line.Attributes().TokenKind {
		case groff.KindComment:
			fmt.Printf("comment at %s:%d\n", line.Span().Start.Filename, line.Span().Start.Line)
		}
		if line.Attributes().Name == ".so" {
			// Recognition does not load a file. The unresolved instruction is source.
			fmt.Printf("include retained: %s", line.Raw())
		}
		if document.Syntax().Node(line.ID()) != line {
			panic("lost source identity")
		}
	}
	// Output:
	// term: TOOL_HOME
	// comment at tool.1:3
	// include retained: .so local.roff
}
