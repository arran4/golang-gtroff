package man_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func ExampleBuild_capturedBlocks() {
	source := ".TH TOOL 1\n.de1 SAVED\n.TH HIDDEN 9\n.SH HIDDEN\n..\n.SH NAME\nvisible\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	doc, err := man.Build(tree, man.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("title=%s sourceExact=%v\n", doc.Metadata.Title, doc.Source() == source)
	for _, node := range doc.Nodes {
		if node.Kind == man.Unsupported {
			fmt.Printf("stored=%s\n", node.Value)
		}
		if node.Kind == man.Section {
			fmt.Printf("section=%s\n", node.Heading)
		}
	}
	fmt.Printf("diagnostics=%d\n", len(doc.Diagnostics))
	// Output:
	// title=TOOL sourceExact=true
	// stored=.de1
	// section=NAME
	// diagnostics=1
}
