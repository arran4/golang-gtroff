package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleDocument_Source() {
	source := ".de\tX\r\n雪\r\n..\r\n"
	doc, err := groff.ParseDocument(strings.NewReader(source))
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.Source() == source, doc.SourceComplete)
	fmt.Printf("%q\n", doc.Tokens[0].RawSource)
	fmt.Printf("%q\n", doc.Tokens[0].Token.String())
	// Output:
	// true true
	// ".de\tX\r\n雪\r\n..\r\n"
	// ".de X\n雪\n.."
}
