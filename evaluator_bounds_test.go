package groff

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEvaluationResourceLimits(t *testing.T) {
	cases := []struct {
		name, kind string
		setup      func(*Environment) []Token
	}{
		{"steps", "step", func(e *Environment) []Token { e.MaxSteps = 1; return []Token{TokenText("x")} }},
		{"work", "work byte", func(e *Environment) []Token { e.MaxWorkBytes = 3; return []Token{TokenText("1234")} }},
		{"branch whitespace", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenRequestIf{Cond: "1", Anything: strings.Repeat(" ", 1000)}}
		}},
		{"register whitespace", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenRequestNr{Ident: "X", Value: strings.Repeat(" ", 1000)}}
		}},
		{"increment whitespace", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenRequestNr{Ident: "X", Value: "1", Incr: strings.Repeat(" ", 1000)}}
		}},
		{"control name", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenUnknownControl{Name: strings.Repeat(".", 1000)}}
		}},
		{"append name", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenRequestAs{Name: strings.Repeat("x", 1000), StringVal: "a"}}
		}},
		{"relative register name", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			return []Token{&TokenRequestNr{Ident: strings.Repeat("x", 1000), Value: "+1"}}
		}},
		{"macro argument whitespace", "work byte", func(e *Environment) []Token {
			e.MaxWorkBytes = 20
			e.DefineMacro("M", []string{"text"})
			return []Token{&TokenUnknownControl{Name: ".M", RawArgs: strings.Repeat(" ", 1000)}}
		}},
		{"output bytes", "output byte", func(e *Environment) []Token { e.MaxOutputBytes = 3; return []Token{TokenText("1234")} }},
		{"output tokens", "output token", func(e *Environment) []Token { e.MaxTokens = 1; return []Token{TokenComment("a"), TokenComment("b")} }},
		{"new state", "state byte", func(e *Environment) []Token {
			e.MaxStateBytes = 40
			return []Token{&TokenRequestDs{Name: "S", StringVal: "12345678"}}
		}},
		{"initial state", "state byte", func(e *Environment) []Token {
			e.MaxStateBytes = 40
			e.SetString("S", strings.Repeat("x", 40))
			return nil
		}},
		{"empty body lines", "state byte", func(e *Environment) []Token {
			e.MaxStateBytes = 40
			return []Token{&TokenMacroDefinition{Name: "M", Body: []string{""}}}
		}},
		{"reparse line", "line byte", func(e *Environment) []Token {
			e.MaxLineBytes = 3
			return []Token{&TokenRequestIf{Cond: "1", Anything: "1234"}}
		}},
		{"nested conditionals", "recursion depth", func(e *Environment) []Token {
			e.MaxDepth = 4
			return []Token{&TokenRequestIf{Cond: "1", Anything: strings.Repeat(".if 1 ", 2000) + "text"}}
		}},
		{"unary work", "step", func(e *Environment) []Token {
			e.MaxSteps = 30
			return []Token{&TokenRequestIf{Cond: strings.Repeat("!", 100000) + "1", Anything: "text"}}
		}},
		{"macro cycle", "recursion depth", func(e *Environment) []Token {
			e.MaxDepth = 4
			e.DefineMacro("M", []string{".M"})
			return []Token{&TokenUnknownControl{Name: ".M"}}
		}},
		{"fanout", "output token", func(e *Environment) []Token {
			e.MaxTokens = 16
			e.DefineMacro("M0", []string{"text"})
			for i := 1; i < 10; i++ {
				line := fmt.Sprintf(".M%d", i-1)
				e.DefineMacro(fmt.Sprintf("M%d", i), []string{line, line})
			}
			return []Token{&TokenUnknownControl{Name: ".M9"}}
		}},
		{"append state", "state byte", func(e *Environment) []Token {
			e.MaxStateBytes = 40
			e.SetString("S", "1234")
			return []Token{&TokenRequestAs{Name: "S", StringVal: "5678"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := NewEnvironment()
			tokens := tc.setup(env)
			out, err := Evaluate(tokens, env)
			var limit *EvaluationLimitError
			if !errors.As(err, &limit) || limit.Kind != tc.kind {
				t.Fatalf("want %s budget, got output %v, error %v", tc.kind, out, err)
			}
			if env.Depth != 0 {
				t.Fatalf("depth leaked: %d", env.Depth)
			}
			if len(out) > env.MaxTokens {
				t.Fatal("output token bound exceeded")
			}
		})
	}
}

func TestEvaluationSelectedParseErrors(t *testing.T) {
	bad := `.B "unclosed`
	cases := [][]Token{
		{&TokenRequestIf{Cond: "1", Anything: bad}},
		{&TokenRequestIe{Cond: "1", Anything: bad}},
		{&TokenRequestIe{Cond: "0"}, &TokenRequestEl{Anything: bad}},
		{&TokenMacroDefinition{Name: "M", Body: []string{bad}}, &TokenUnknownControl{Name: ".M"}},
	}
	for i, tokens := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			env := NewEnvironment()
			out, err := Evaluate(append([]Token{TokenText("prefix")}, tokens...), env)
			var evalErr *EvaluationError
			if !errors.Is(err, ErrUnclosedQuote) || !errors.As(err, &evalErr) || len(evalErr.Expansion) == 0 {
				t.Fatalf("lost parse error/path: %v", err)
			}
			if len(out) != 1 || out[0] != TokenText("prefix") {
				t.Fatalf("partial output lost: %v", out)
			}
			if env.Depth != 0 {
				t.Fatal("depth leaked after failure")
			}
		})
	}
	out, err := Evaluate([]Token{&TokenRequestIf{Cond: "0", Anything: bad}, TokenText("after")}, nil)
	if err != nil || len(out) != 1 || out[0] != TokenText("after") {
		t.Fatalf("disabled branch interpreted: %v %v", out, err)
	}
}

func TestEvaluationInvalidStateAndIntegers(t *testing.T) {
	for _, tokens := range [][]Token{{nil}, {(*TokenRequestDs)(nil)}, {&TokenRequestEl{Anything: "orphan"}}} {
		if _, err := Evaluate(tokens, nil); err == nil {
			t.Fatal("expected invalid-token/pairing error")
		}
	}
	max := int(^uint(0) >> 1)
	for _, tc := range []struct {
		value, incr string
		initial     int
	}{
		{"bad", "", 7}, {"2+3", "", 7}, {"10", "invalid", 7}, {"+1", "", max}, {"-1", "", -max - 1},
	} {
		env := NewEnvironment()
		env.SetRegister("X", tc.initial, 0)
		_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: tc.value, Incr: tc.incr}}, env)
		if err == nil || env.GetRegister("X") != tc.initial {
			t.Fatalf("invalid register assignment applied: %+v, %d, %v", tc, env.GetRegister("X"), err)
		}
	}
}

// An Err-driven context makes mid-evaluation cancellation deterministic without
// timing-dependent sleeps or a worker goroutine that can outlive a failed test.
type evaluationCancelContext struct {
	context.Context
	remaining int
}

func (c *evaluationCancelContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func TestEvaluationCancellationAndLifetime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := EvaluateContext(ctx, []Token{TokenText("x")}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	env := NewEnvironment()
	ctx2 := &evaluationCancelContext{Context: context.Background(), remaining: 25}
	_, err := EvaluateContext(ctx2, []Token{TokenText(strings.Repeat(`\n[X]`, 100))}, env)
	if !errors.Is(err, context.Canceled) || env.StepCount == 0 {
		t.Fatalf("mid-interpolation cancellation lost: %v", err)
	}
	var evaluated *EvaluationError
	if !errors.As(err, &evaluated) {
		t.Fatal("cancellation lacks evaluation context")
	}
	if _, err := EvaluateContext(nil, nil, env); err == nil {
		t.Fatal("nil context accepted")
	}

	var zero Environment
	out, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: "7"}, &TokenRequestDs{Name: "S", StringVal: "zero"}, TokenText(`\*[S] \nX`)}, &zero)
	if err != nil || len(out) != 1 || out[0] != TokenText("zero 7") {
		t.Fatalf("zero value not usable: %v %v", out, err)
	}
	zero.MaxSteps = zero.StepCount
	if _, err := Evaluate([]Token{TokenText("x")}, &zero); err == nil {
		t.Fatal("shared step budget was reset")
	}
	zero.StepCount = 0
	zero.MaxSteps = 0
	if _, err := Evaluate([]Token{TokenText("x")}, &zero); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluationInterpolationProfile(t *testing.T) {
	env := NewEnvironment()
	env.SetRegister("X", 7, 2)
	env.SetRegister("XY", 8, 0)
	env.SetString("µ", "unicode")
	env.SetString("C", `\*[C]`)
	input := `\nX \n(XY \n[missing] \*[µ] \*[unknown] \*[C] \\nX \fB` + string([]byte{0xff})
	want := `7 8 0 unicode \*[unknown] \*[C] \\nX \fB` + string([]byte{0xff})
	out, err := env.ExpandTextContext(context.Background(), input)
	if err != nil || out != want {
		t.Fatalf("single-pass profile: %q, %v; want %q", out, err, want)
	}
	if env.GetRegister("X") != 7 {
		t.Fatal("plain interpolation incremented register")
	}
	for _, input := range []string{`\n`, `\*[]`, `\n(X`, `\*[unclosed`, `\n+[X]`} {
		if _, err := env.ExpandTextContext(context.Background(), input); err == nil {
			t.Fatalf("invalid/unsupported interpolation %q accepted", input)
		}
	}
	env.MaxOutputBytes = 3
	if out := env.ExpandText("longer"); out != "longer" || len(env.Diagnostics) != 1 {
		t.Fatalf("legacy wrapper hid failure: %q %+v", out, env.Diagnostics)
	}
}

func TestEvaluationDefinitionCopyAndPassthrough(t *testing.T) {
	env := NewEnvironment()
	body := []string{"original"}
	env.DefineMacro("M", body)
	body[0] = "changed"
	unknown := &TokenRequestSo{File: "/etc/passwd"}
	out, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M"}, unknown}, env)
	if err != nil || !reflect.DeepEqual(out, []Token{TokenText("original"), unknown}) || len(env.Diagnostics) != 1 {
		t.Fatalf("copy/preservation failed: %v %v %+v", out, err, env.Diagnostics)
	}
	if _, err := Evaluate(nil, env); err != nil || len(env.Diagnostics) != 0 {
		t.Fatal("diagnostics not reset on next invocation")
	}
}

func TestEvaluationIgnoresGlobalHandlers(t *testing.T) {
	for _, arg := range os.Args {
		if arg != "evaluation-global-helper" {
			continue
		}
		h := &registryTestHandler{}
		RegisterCustomHandler(registryGlobalHandler{h})
		env := NewEnvironment()
		out, err := Evaluate([]Token{&TokenRequestIf{Cond: "1", Anything: ".GLOBAL selected"}}, env)
		if err != nil || h.calls.Load() != 0 || len(out) != 1 || len(env.Diagnostics) != 1 {
			t.Fatalf("global handler leaked into evaluation: %v %v %+v", out, err, env.Diagnostics)
		}
		if _, ok := out[0].(*TokenUnknownControl); !ok {
			t.Fatalf("expected preserved control, got %T", out[0])
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestEvaluationIgnoresGlobalHandlers$", "--", "evaluation-global-helper")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("handler isolation subprocess: %v\n%s", err, out)
	}
}

func FuzzEvaluateBounded(f *testing.F) {
	for _, seed := range []string{".nr X 1\n\\n[X]\n", ".if !0 visible\n", ".de M\n.M\n..\n.M\n", "\\*[unclosed"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 {
			t.Skip()
		}
		doc, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), WithDocumentBytes(4096))
		if err != nil {
			return
		}
		tokens := make([]Token, len(doc.Tokens))
		for i, dt := range doc.Tokens {
			tokens[i] = dt.Token
		}
		env := NewEnvironment()
		env.MaxSteps = 200
		env.MaxDepth = 8
		env.MaxOutputBytes = 4096
		env.MaxWorkBytes = 8192
		env.MaxStateBytes = 4096
		env.MaxTokens = 100
		out, _ := Evaluate(tokens, env)
		if len(out) > 100 || env.Depth != 0 || env.StepCount > 200 {
			t.Fatal("evaluation escaped a budget")
		}
		bytes := 0
		for _, tok := range out {
			if nilInterface(tok) {
				t.Fatal("nil output")
			}
			if text, ok := tok.(TokenText); ok {
				bytes += len(text)
			}
		}
		if bytes > 4096 {
			t.Fatal("emitted text exceeded budget")
		}
	})
}

func TestEvaluationLimitsCannotBeDisabled(t *testing.T) {
	max := int(^uint(0) >> 1)
	for _, depth := range []int{-1, 0, max} {
		env := NewEnvironment()
		env.MaxDepth = depth
		env.DefineMacro("M", []string{".M"})
		_, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M"}}, env)
		var limit *EvaluationLimitError
		want := 50
		if depth == max {
			want = 128
		}
		if !errors.As(err, &limit) || limit.Kind != "recursion depth" || limit.Limit != want {
			t.Fatalf("depth %d disabled bound: %v", depth, err)
		}
	}
	env := NewEnvironment()
	env.MaxLineBytes = max
	_, err := Evaluate([]Token{&TokenRequestIf{Cond: "1", Anything: strings.Repeat("x", (1<<20)+1)}}, env)
	var limit *EvaluationLimitError
	if !errors.As(err, &limit) || limit.Kind != "line byte" || limit.Limit != 1<<20 {
		t.Fatalf("line safety ceiling disabled: %v", err)
	}
	env = NewEnvironment()
	env.StepCount = -1
	if _, err := Evaluate(nil, env); err == nil {
		t.Fatal("negative work counter accepted")
	}
}
