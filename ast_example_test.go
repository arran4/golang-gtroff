package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleParseAST() {
	source := ".TH DEMO 1\n.SH NAME\ndemo \\- inspect roff\n.SH OPTIONS\n.TP\n.B \\-v\nVerbose output.\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		panic(err)
	}
	options := tree.Root().FindSection("OPTIONS", false)[0]
	fmt.Print(options.Text())
	line := options.FindMacro("B")[0]
	token, err := line.Token()
	if err != nil {
		panic(err)
	}
	fmt.Printf("typed: %s\n", token.(*groff.TokenManB).Text)
	fmt.Println("source preserved:", tree.Source() == source)
	// Output:
	// OPTIONS
	// \-v
	// Verbose output.
	// typed: \-v
	// source preserved: true
}
