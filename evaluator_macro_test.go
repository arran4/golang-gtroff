package groff

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestEvaluationMacroParameterForms(t *testing.T) {
	env := NewEnvironment()
	env.DefineMacro("M", []string{`\$0|\$1|\$(02|\$[10]|\$[11]|\n[.$]`})
	out, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M", RawArgs: `"two words" "" c d e f g h i j`}}, env)
	if err != nil || len(out) != 1 || out[0] != TokenText("M|two words||j||10") {
		t.Fatalf("parameter indexing: %v %v", out, err)
	}
	for _, reference := range []string{`\$@`, `\$*`, `\$[-1]`, `\$(00`, `\$[0]`, `\$(a0`, `\$[x]`, `\$[]`, `\$`} {
		env := NewEnvironment()
		env.DefineMacro("M", []string{reference})
		if _, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M"}}, env); err == nil {
			t.Fatalf("unsupported parameter %q accepted", reference)
		}
	}
	env = NewEnvironment()
	env.DefineMacro("M", []string{"unused"})
	out, err = Evaluate([]Token{TokenText("prefix"), &TokenUnknownControl{Name: ".M", RawArgs: `"unclosed`}}, env)
	var evaluated *EvaluationError
	if !errors.Is(err, ErrUnclosedQuote) || !errors.As(err, &evaluated) || len(evaluated.Expansion) == 0 || evaluated.Expansion[0] != "macro M" || len(out) != 1 {
		t.Fatalf("macro argument error lost identity/path/prefix: %v %v", out, err)
	}
}
func TestEvaluationMacroArgumentValuesAreOpaque(t *testing.T) {
	env := NewEnvironment()
	env.SetRegister("X", 0, 1)
	env.SetString("S", `\n+[X]`)
	env.DefineMacro("M", []string{`\$1 \$1`, `.B "\$1"`})
	out, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M", RawArgs: `\*[S]`}}, env)
	if err != nil || len(out) != 2 || out[0] != TokenText(`\n+[X] \n+[X]`) || env.GetRegister("X") != 0 {
		t.Fatalf("argument was re-interpolated: %v %v", out, err)
	}
	bold, ok := out[1].(*TokenManB)
	if !ok || bold.Text != `\n+[X]` {
		t.Fatalf("typed argument changed semantics: %T %v", out[1], out[1])
	}
	env = NewEnvironment()
	env.DefineMacro("M", []string{`\$1`})
	out, err = Evaluate([]Token{&TokenUnknownControl{Name: ".M", RawArgs: `".sy arbitrary-command"`}}, env)
	if err != nil || len(out) != 1 || out[0] != TokenText(".sy arbitrary-command") {
		t.Fatalf("parameter constructed a control request: %v %v", out, err)
	}
}
func TestEvaluationMacroAndStateBudgets(t *testing.T) {
	env := NewEnvironment()
	env.DefineMacro("M", []string{"unused"})
	env.MaxLineBytes = 8
	_, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M", RawArgs: strings.Repeat(" ", 1000)}}, env)
	var limit *EvaluationLimitError
	if !errors.As(err, &limit) || limit.Kind != "line byte" {
		t.Fatalf("macro arguments bypassed line bound: %v", err)
	}
	env = NewEnvironment()
	env.SetString("S", "123456")
	env.DefineMacro("M", []string{"unused"})
	env.MaxOutputBytes = 8
	_, err = Evaluate([]Token{&TokenUnknownControl{Name: ".M", RawArgs: `\*[S] \*[S]`}}, env)
	if !errors.As(err, &limit) || limit.Kind != "output byte" {
		t.Fatalf("macro frame bypassed byte bound: %v", err)
	}
	for _, tokens := range [][]Token{{&TokenRequestNr{Ident: ".$", Value: "7"}}, {&TokenRequestRr{Ident: ".$"}}, {TokenText(`\n+[.$]`)}} {
		if _, err := Evaluate(tokens, nil); err == nil {
			t.Fatal("parameter count was mutable")
		}
	}
	env = NewEnvironment()
	env.SetRegister(".$", 7, 0)
	if _, err := Evaluate(nil, env); err == nil {
		t.Fatal("stored parameter count shadows scoped count")
	}
	ctx := &evaluationCancelContext{Context: context.Background(), remaining: 30}
	env = NewEnvironment()
	env.DefineMacro("M", []string{"unused"})
	if _, err := EvaluateContext(ctx, []Token{&TokenUnknownControl{Name: ".M", RawArgs: strings.Repeat("arg ", 100)}}, env); !errors.Is(err, context.Canceled) {
		t.Fatalf("argument cancellation lost: %v", err)
	}
}

func TestEvaluationFormattingCopiesAndUnsupportedControls(t *testing.T) {
	env := NewEnvironment()
	env.SetString("S", "two words")
	env.SetRegister("X", 0, 1)
	bold := &TokenManB{Text: `\*[S]`}
	include := &TokenRequestSo{File: `\n+[X]`}
	out, err := Evaluate([]Token{bold, include}, env)
	if err != nil || len(out) != 2 {
		t.Fatalf("formatting: %v %v", out, err)
	}
	copy, ok := out[0].(*TokenManB)
	if !ok || copy == bold || copy.Text != "two words" || bold.Text != `\*[S]` {
		t.Fatal("formatting expansion mutated or reparsed the original token")
	}
	if out[1] != include || env.GetRegister("X") != 0 {
		t.Fatal("unsupported include interpolated or changed")
	}
}

func TestEvaluationGeneratedRequestShadowingIsVisible(t *testing.T) {
	env := NewEnvironment()
	env.DefineMacro("nr", []string{"shadowed"})
	request := &TokenRequestNr{Ident: "X", Value: "7"}
	out, err := Evaluate([]Token{request}, env)
	if err != nil || len(out) != 1 || out[0] != request || len(env.Diagnostics) != 1 || env.GetRegister("X") != 0 {
		t.Fatalf("unsupported shadow executed the built-in request: %v %v %+v", out, err, env.Diagnostics)
	}
}
