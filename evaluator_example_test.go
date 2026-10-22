package groff_test

import (
	"context"
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleEvaluateContext() {
	source := ".nr X 7\n.ds S Hello\n\\*[S] at \\n[X]\n"
	doc, err := groff.ParseDocument(strings.NewReader(source), groff.WithRegistry(groff.NewRegistry()))
	if err != nil {
		panic(err)
	}
	tokens := make([]groff.Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		tokens[i] = dt.Token
	}
	env := groff.NewEnvironment()
	env.MaxOutputBytes = 1024
	output, err := groff.EvaluateContext(context.Background(), tokens, env)
	if err != nil {
		panic(err)
	}
	for _, tok := range output {
		fmt.Println(tok.String())
	}
	fmt.Printf("diagnostics: %d; source preserved: %v\n", len(env.Diagnostics), doc.Source() == source)
	// Output:
	// Hello at 7
	// diagnostics: 0; source preserved: true
}
