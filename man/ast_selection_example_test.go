package man_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func ExampleDocument_SelectAST() {
	source := ".SH EXAMPLES\n.nf\n  chosen\n.fi\noutside\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	doc, err := man.Build(tree, man.Options{})
	if err != nil {
		panic(err)
	}
	block := tree.Root().FindKind(groff.ASTBlock)[0]
	selected, err := doc.SelectAST(block.ID())
	if err != nil {
		panic(err)
	}
	fmt.Printf("kind=%s text=%q\n", doc.Node(selected.Roots[0]).Kind, selected.Text(man.TextOptions{Normalize: true}))
	fmt.Printf("raw=%q\n", selected.Text(man.TextOptions{Raw: true}))
	// Output:
	// kind=preformatted text="chosen"
	// raw=".nf\n  chosen\n.fi\n"
}
