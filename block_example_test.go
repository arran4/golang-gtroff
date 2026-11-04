package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

type literalBlock struct{}

func (literalBlock) CanHandle(cmd, pkg string) bool { return cmd == ".LITERAL" && pkg == "man" }
func (literalBlock) Begin(header groff.BlockHeader) (groff.BlockRule, error) {
	return groff.BlockRule{Ends: []string{".ENDLITERAL"}}, nil
}
func (literalBlock) ParseBlock(source groff.BlockSource) (groff.Token, error) {
	// Inspection only: no request evaluation, include resolution or process launch.
	return groff.TokenText(source.BodySource()), nil
}

var _ groff.BlockHandler = literalBlock{}

func ExampleWithBlockHandler() {
	input := ".LITERAL\n.B stored as raw text\n雪\n.ENDLITERAL\n"
	doc, err := groff.ParseDocument(strings.NewReader(input), groff.WithPackage("man"), groff.WithRegistry(groff.NewRegistry()), groff.WithBlockHandler(literalBlock{}), groff.WithStrict(true))
	if err != nil {
		panic(err)
	}
	block := doc.Tokens[0].Block
	fmt.Printf("body=%q\n", block.BodySource())
	fmt.Printf("closed=%v sourceExact=%v\n", block.Closed, block.Source() == input)
	// Output:
	// body=".B stored as raw text\n雪\n"
	// closed=true sourceExact=true
}
