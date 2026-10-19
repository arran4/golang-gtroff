package groff

import (
	"strings"
	"testing"
)

func TestParseDocument_FinalLinePreserved(t *testing.T) {
	// Document without newline on the final line
	input := ".TH TEST 1\n.SH NAME\ntest - test command"
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	if len(doc.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(doc.Tokens))
	}

	last := doc.Tokens[2]
	if last.RawLine != "test - test command" {
		t.Errorf("last line raw text mismatch: got %q", last.RawLine)
	}
	if last.Span.Start.Line != 3 {
		t.Errorf("expected line 3, got %d", last.Span.Start.Line)
	}
}

func TestParseDocument_BlankLinesAndComments(t *testing.T) {
	input := ".TH TEST 1\n\n.\\\" Comment line\nSome text\n"
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	if len(doc.Tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(doc.Tokens))
	}

	if _, ok := doc.Tokens[0].Token.(*TokenMacroTH); !ok {
		t.Errorf("token 0 should be *TokenMacroTH")
	}
	if text, ok := doc.Tokens[1].Token.(TokenText); !ok || string(text) != "" {
		t.Errorf("token 1 should be empty TokenText for blank line, got %v", doc.Tokens[1].Token)
	}
	if _, ok := doc.Tokens[2].Token.(TokenComment); !ok {
		t.Errorf("token 2 should be TokenComment, got %T", doc.Tokens[2].Token)
	}
	if text, ok := doc.Tokens[3].Token.(TokenText); !ok || string(text) != "Some text" {
		t.Errorf("token 3 should be 'Some text', got %v", doc.Tokens[3].Token)
	}
}

func TestParseDocument_PreprocessorBlocks(t *testing.T) {
	input := `.TH TEST 1
.SH SYNOPSIS
.TS
allbox tab(;);
c s
l l.
Name;Value
.TE
.PP
Text after table.
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	foundTable := false
	for _, tok := range doc.Tokens {
		if tb, ok := tok.Token.(*TokenTableBlock); ok {
			foundTable = true
			if len(tb.Lines) != 4 {
				t.Errorf("expected 4 table lines, got %d", len(tb.Lines))
			}
			if tb.Lines[0] != "allbox tab(;);" {
				t.Errorf("table line 0 mismatch: %q", tb.Lines[0])
			}
		}
	}
	if !foundTable {
		t.Errorf("expected to find TokenTableBlock in parsed document")
	}
}

func TestParseDocument_MultilineMacroDefinition(t *testing.T) {
	input := `.de MYMAC
.B Hello
..
.MYMAC
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(false))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	if len(doc.Tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(doc.Tokens))
	}

	def, ok := doc.Tokens[0].Token.(*TokenMacroDefinition)
	if !ok {
		t.Fatalf("expected TokenMacroDefinition, got %T", doc.Tokens[0].Token)
	}
	if def.Name != "MYMAC" {
		t.Errorf("expected macro name MYMAC, got %q", def.Name)
	}
	if len(def.Body) != 1 || def.Body[0] != ".B Hello" {
		t.Errorf("macro body mismatch: %v", def.Body)
	}
}

func TestParseDocument_IgnoredBlock(t *testing.T) {
	input := `.ig
This is ignored text
which should not be parsed as macros
.TH FAKE 1
..
.PP
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	if len(doc.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(doc.Tokens))
	}

	ig, ok := doc.Tokens[0].Token.(*TokenIgnoredBlock)
	if !ok {
		t.Fatalf("expected TokenIgnoredBlock, got %T", doc.Tokens[0].Token)
	}
	if len(ig.Body) != 3 {
		t.Errorf("expected 3 ignored lines, got %d", len(ig.Body))
	}
}

func TestParseDocument_PermissiveUnknownControl(t *testing.T) {
	input := ".TH TEST 1\n.XYZUNKNOWN arg1 arg2\n"
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(false))
	if err != nil {
		t.Fatalf("ParseDocument permissive error: %v", err)
	}

	if len(doc.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(doc.Tokens))
	}

	unk, ok := doc.Tokens[1].Token.(*TokenUnknownControl)
	if !ok {
		t.Fatalf("expected TokenUnknownControl, got %T", doc.Tokens[1].Token)
	}
	if unk.Name != ".XYZUNKNOWN" {
		t.Errorf("expected .XYZUNKNOWN, got %q", unk.Name)
	}
	if unk.RawArgs != "arg1 arg2" {
		t.Errorf("expected 'arg1 arg2', got %q", unk.RawArgs)
	}
}
