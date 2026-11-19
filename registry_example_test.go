package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

type noteHandler struct{}

func (noteHandler) CanHandle(cmd, pkg string) bool { return cmd == ".NOTE" && pkg == "man" }
func (noteHandler) Parse(cmd, args, pkg string) (groff.Token, error) {
	return groff.TokenText(args), nil
}

var _ groff.CustomHandler = noteHandler{}

func ExampleWithRegistry() {
	registry := groff.NewRegistry()
	registry.RegisterCustom(noteHandler{})
	doc, err := groff.ParseDocument(strings.NewReader(".NOTE local extension\n"),
		groff.WithRegistry(registry), groff.WithPackage("man"), groff.WithStrict(true))
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.Tokens[0].Token.String())
	fmt.Println(doc.Tokens[0].RawLine)
	// Output:
	// local extension
	// .NOTE local extension
}
