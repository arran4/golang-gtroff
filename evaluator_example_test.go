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

func ExampleEnvironment_State() {
	backing := groff.NewEnvironment()
	config := groff.NewEnvironment()
	config.State = backing
	output, err := groff.EvaluateContext(context.Background(), []groff.Token{
		&groff.TokenRequestNr{Ident: "X", Value: "3+5*4"},
		groff.TokenText(`X=\n[X]`),
	}, config)
	if err != nil {
		panic(err)
	}
	fmt.Println(output[0].String())
	fmt.Printf("backing X=%d; config bindings=%d\n", backing.GetRegister("X"), len(config.Registers))
	// Output:
	// X=32
	// backing X=32; config bindings=0
}
