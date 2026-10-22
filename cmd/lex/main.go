package main

import (
	"errors"
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
	return processReaderTo(sourceName, r, os.Stdout, os.Stderr,
		groff.WithPackage(*pkgFlag),
		groff.WithStrict(*strictFlag),
	)
}

func processReaderTo(sourceName string, r io.Reader, out, errOut io.Writer, opts ...groff.Option) error {
	opts = append(opts, groff.WithSourceName(sourceName))
	doc, parseErr := groff.ParseDocument(r, opts...)
	if doc == nil {
		return parseErr
	}

	for _, dt := range doc.Tokens {
		if _, err := fmt.Fprintf(out, "%s:%d:%d: %s\n", dt.Span.Start.Filename, dt.Span.Start.Line, dt.Span.Start.Column, dt.Token.String()); err != nil {
			return errors.Join(parseErr, err)
		}
	}

	for _, diag := range doc.Diagnostics {
		if _, err := fmt.Fprintf(errOut, "%s:%d:%d: %s: %s\n", diag.Position.Filename, diag.Position.Line, diag.Position.Column, diag.Severity, diag.Message); err != nil {
			return errors.Join(parseErr, err)
		}
	}

	return parseErr
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
	if doc == nil {
		return nil, err
	}

	tokens := make([]groff.Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		tokens[i] = dt.Token
	}
	return tokens, err
}
