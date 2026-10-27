package groff

import (
	"strings"
	"testing"
)

func TestEvaluate_RegistersAndStrings(t *testing.T) {
	input := `.nr PO 72
.ds HF Helvetica-Bold
Line with offset \n[PO] and font \*[HF]
.nr PO +10
.as HF -Extension
Updated \n[PO] and \*[HF]
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	rawTokens := make([]Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		rawTokens[i] = dt.Token
	}

	env := NewEnvironment()
	evaluated, err := Evaluate(rawTokens, env)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	// Verify register was tracked
	if env.GetRegister("PO") != 82 {
		t.Errorf("expected register PO=82, got %d", env.GetRegister("PO"))
	}
	// Verify string was tracked and appended
	if str, _ := env.GetString("HF"); str != "Helvetica-Bold-Extension" {
		t.Errorf("expected string HF='Helvetica-Bold-Extension', got %q", str)
	}

	// Check evaluated text tokens
	foundFirst := false
	foundSecond := false
	for _, tok := range evaluated {
		if text, ok := tok.(TokenText); ok {
			if strings.Contains(string(text), "Line with offset 72 and font Helvetica-Bold") {
				foundFirst = true
			}
			if strings.Contains(string(text), "Updated 82 and Helvetica-Bold-Extension") {
				foundSecond = true
			}
		}
	}

	if !foundFirst {
		t.Errorf("expected first line text to be expanded with initial register/string values")
	}
	if !foundSecond {
		t.Errorf("expected second line text to be expanded with updated values")
	}
}

func TestEvaluate_Conditionals(t *testing.T) {
	input := `.ds OS Linux
.ie 'Linux'\*[OS] .B "Is Linux"
.el .B "Other OS"
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(true))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	rawTokens := make([]Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		rawTokens[i] = dt.Token
	}

	env := NewEnvironment()
	evaluated, err := Evaluate(rawTokens, env)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	foundLinux := false
	for _, tok := range evaluated {
		if b, ok := tok.(*TokenMacroB); ok {
			if b.Text == "Is Linux" {
				foundLinux = true
			}
		}
	}
	if !foundLinux {
		t.Errorf("expected conditional true branch to execute .B 'Is Linux'")
	}
}

func TestEvaluate_MacroDefinitionAndExpansion(t *testing.T) {
	input := `.de GREET
Hello World
..
.GREET
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(false))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	rawTokens := make([]Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		rawTokens[i] = dt.Token
	}

	env := NewEnvironment()
	evaluated, err := Evaluate(rawTokens, env)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	foundHello := false
	for _, tok := range evaluated {
		if text, ok := tok.(TokenText); ok {
			if string(text) == "Hello World" {
				foundHello = true
			}
		}
	}
	if !foundHello {
		t.Errorf("expected dynamic macro .GREET expansion to produce 'Hello World'")
	}
}

func TestEvaluate_RecursionLimit(t *testing.T) {
	// Infinite recursive macro
	input := `.de LOOP
.LOOP
..
.LOOP
`
	doc, err := ParseDocument(strings.NewReader(input), WithStrict(false))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	rawTokens := make([]Token, len(doc.Tokens))
	for i, dt := range doc.Tokens {
		rawTokens[i] = dt.Token
	}

	env := NewEnvironment()
	env.MaxDepth = 5

	_, err = Evaluate(rawTokens, env)
	if err == nil {
		t.Errorf("expected recursion depth error, got nil")
	} else if !strings.Contains(err.Error(), "recursion depth exceeded") {
		t.Errorf("unexpected error message: %v", err)
	}
}
