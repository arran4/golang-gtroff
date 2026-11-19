package groff_test

import (
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

func ExampleWithBlockNameResolver() {
	input := ".dei TARGET ENDREF\n.B raw body\n.STOP\n"
	names := map[string]string{"ENDREF": "STOP"}
	resolver := groff.BlockNameResolverFunc(func(header groff.BlockHeader, reference string) (string, error) {
		name, ok := names[reference]
		if !ok {
			return "", fmt.Errorf("unknown delimiter reference %q", reference)
		}
		return name, nil
	})
	doc, err := groff.ParseDocument(strings.NewReader(input), groff.WithRegistry(groff.NewRegistry()), groff.WithBlockNameResolver(resolver), groff.WithStrict(true))
	if err != nil {
		panic(err)
	}
	token := doc.Tokens[0].Token.(*groff.TokenMacroDefinitionVariant)
	fmt.Printf("request=%s nameReference=%s endReference=%s delimiter=%s\n", token.Request, token.Name, token.End, token.Delimiter)
	fmt.Printf("sourceExact=%v\n", doc.Tokens[0].Block.Source() == input)
	// Output:
	// request=.dei nameReference=TARGET endReference=ENDREF delimiter=.STOP
	// sourceExact=true
}
