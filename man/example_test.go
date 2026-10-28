package man_test

import (
	"fmt"
	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"strings"
)

func ExampleBuild() {
	source := ".TH DEMO 1\n.SH OPTIONS\n.TP\n.B \\-v\nVerbose output.\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	document, err := man.Build(tree, man.Options{Strict: true})
	if err != nil {
		panic(err)
	}
	options, err := document.SelectSection("OPTIONS", false)
	if err != nil {
		panic(err)
	}
	fmt.Println(options.Metadata.Title)
	fmt.Println(options.Text(man.TextOptions{Normalize: true}))
	fmt.Println("source preserved:", document.Source() == source)
	// Output:
	// DEMO
	// OPTIONS -v Verbose output.
	// source preserved: true
}
