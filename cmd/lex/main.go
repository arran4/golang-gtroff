package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	groff "github.com/arran4/golang-gtroff"
)

var (
	pkgFlag    = flag.String("package", "", "Macro package context: man, ms, or roff")
	strictFlag = flag.Bool("strict", false, "Strict mode: fail on unrecognized commands")
)

func main() {
	flag.Parse()
	files := flag.Args()

	if len(files) == 0 {
		if err := processReader("<stdin>", os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	hasError := false
	for _, fn := range files {
		if err := processFile(fn); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", fn, err)
			hasError = true
		}
	}

	if hasError {
		os.Exit(1)
	}
}

func processFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("open file %s: %w", fn, err)
	}
	defer f.Close()

	return processReader(fn, f)
}

func processReader(sourceName string, r io.Reader) error {
	doc, err := groff.ParseDocument(r,
		groff.WithPackage(*pkgFlag),
		groff.WithStrict(*strictFlag),
		groff.WithSourceName(sourceName),
	)
	if err != nil {
		return err
	}

	for _, dt := range doc.Tokens {
		fmt.Printf("%s:%d:%d: %s\n", dt.Span.Start.Filename, dt.Span.Start.Line, dt.Span.Start.Column, dt.Token.String())
	}

	for _, diag := range doc.Diagnostics {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", diag.Position.Filename, diag.Position.Line, diag.Position.Column, diag.Severity, diag.Message)
	}

	return nil
}

// ParseFile provides backwards compatibility for programmatic use.
func ParseFile(fn string) ([]groff.Token, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("open file: %s: %w", fn, err)
	}
	defer f.Close()

	doc, err := groff.ParseDocument(f,
		groff.WithSourceName(fn),
		groff.WithStrict(false),
	)
	if err != nil {
		return nil, err
	}

	tokens := make([]groff.Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		tokens[i] = dt.Token
	}
	return tokens, nil
}
