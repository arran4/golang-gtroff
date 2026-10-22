package groff_test

import (
	"fmt"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleParseArguments() {
	input := `  café\ mocha "世界" \" comment`
	arguments, err := groff.ParseArguments(input)
	if err != nil {
		panic(err)
	}
	lexemes, spans := arguments.Lexemes(), arguments.ByteSpans()
	for i, lexeme := range lexemes {
		fmt.Printf("%q bytes=%d:%d runes=%d:%d\n", lexeme.Value, spans[i].Start, spans[i].End, lexeme.Start, lexeme.End)
	}
	canonical, err := arguments.Canonical()
	if err != nil {
		panic(err)
	}
	fmt.Printf("source=%q\ncanonical=%q\n", arguments.Source(), canonical)
	// Output:
	// "café mocha" bytes=2:14 runes=2:13
	// "世界" bytes=15:23 runes=14:18
	// source="  café\\ mocha \"世界\" \\\" comment"
	// canonical="\"café mocha\" 世界"
}
