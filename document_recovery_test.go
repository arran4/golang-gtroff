package groff

import (
	"errors"
	"strings"
	"testing"
)

// Refs #11/#12/#14: permissive EOF must preserve the consumed body rather than
// dereferencing a nil block token. Syntax parsing does not invent a delimiter.
func TestDocumentUnterminatedBlockPreserved(t *testing.T) {
	for _, header := range []string{".de X", ".am X", ".ig", ".TS", ".EQ", ".PS", ".["} {
		for _, ending := range []string{"", "\n"} {
			input := header + "\n雪\n.unknown content" + ending
			t.Run(header+ending, func(t *testing.T) {
				doc, err := ParseDocument(strings.NewReader(input))
				if err != nil {
					t.Fatal(err)
				}
				if len(doc.Tokens) != 1 || len(doc.Diagnostics) != 1 {
					t.Fatalf("tokens/diagnostics: %+v", doc)
				}
				if got := doc.Tokens[0].Token.String(); got != input {
					t.Fatalf("lost input: %q != %q", got, input)
				}
				if doc.Tokens[0].Span.End.Offset != len(input) {
					t.Fatal("wrong consumed span")
				}
				if _, err := ParseDocument(strings.NewReader(input), WithStrict(true)); err == nil {
					t.Fatal("strict mode accepted unterminated block")
				}
			})
		}
	}
}

func TestDocumentBlockInputFailureIsTerminal(t *testing.T) {
	for _, header := range []string{".TS", ".de X", ".ig"} {
		reader := &terminalBlockReader{data: header + "\n雪", err: errBlockReader}
		doc, err := ParseDocument(reader)
		if !errors.Is(err, errBlockReader) || doc == nil {
			t.Fatalf("reader failure: %v", err)
		}
		if len(doc.Tokens) != 1 || doc.Tokens[0].Token.String() != reader.data {
			t.Fatal("read bytes were discarded")
		}
		if doc.Tokens[0].Span.End.Column != 2 {
			t.Fatal("Unicode recovery column")
		}
		budgetDoc, budgetErr := ParseDocument(strings.NewReader(header+"\nbody\n..\n"), WithMaxLines(1))
		if budgetErr == nil || budgetDoc == nil || len(budgetDoc.Tokens) != 1 {
			t.Fatal("block budget ignored")
		}
	}
}

var errBlockReader = errors.New("block source reader failed")

type terminalBlockReader struct {
	data string
	err  error
	done bool
}

func (r *terminalBlockReader) Read(p []byte) (int, error) {
	if r.done {
		panic("parser retried terminal read failure")
	}
	r.done = true
	return copy(p, r.data), r.err
}

func TestArgumentParserTabWhitespace(t *testing.T) {
	tok, err := ParseLine(".TH\tDEMO\t1")
	if err != nil {
		t.Fatal(err)
	}
	if th := tok.(*TokenMacroTH); th.Title != "DEMO" || th.Section != "1" {
		t.Fatalf("tab arguments: %+v", th)
	}
	args, _, err := ArgumentParser("\"a\tb\"\tc")
	if err != nil || len(args) != 2 || args[0] != "a\tb" || args[1] != "c" {
		t.Fatalf("quoted tab lost: %q %v", args, err)
	}
}
