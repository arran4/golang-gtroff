package renderimage_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
)

func ExampleLayout() {
	tree, err := groff.ParseAST(strings.NewReader(".TH TOOL 1\n.SH NAME\ntool - inspect\n"), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	doc, err := man.Build(tree, man.Options{})
	if err != nil {
		panic(err)
	}
	plan, err := renderimage.Layout(doc, renderimage.Options{Width: 600, PageHeight: 800})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d page, %d pixels\n", plan.Result().Pages, plan.Result().Pixels)
	// Output: 1 page, 480000 pixels
}
