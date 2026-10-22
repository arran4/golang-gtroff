package groff

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func TestEvaluationRoffExpressions(t *testing.T) {
	cases := []struct {
		expression string
		want       int
	}{
		{"3+5*4", 32}, {"(3+5)*4", 32}, {"3+(5*4)", 23},
		{"199/100", 1}, {"5/-2", -2}, {"-5/2", -2}, {"5%-2", 1}, {"-5%2", -1},
		{"1+2<4*3", 3}, {"3<=3", 1}, {"3>=4", 0}, {"3=3", 1}, {"4==3", 0},
		{"(-1)&1", 0}, {"(-1):2", 1}, {"1&3", 1}, {"0:0", 0}, {"-0", 0},
		{" --3 + ( 2 * -4 ) ", -5},
	}
	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			env := NewEnvironment()
			_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: "(" + tc.expression + ")"}}, env)
			if err != nil || env.GetRegister("X") != tc.want {
				t.Fatalf("want %d, got %d, %v", tc.want, env.GetRegister("X"), err)
			}
		})
	}
	env := NewEnvironment()
	env.SetRegister("X", 4, 0)
	env.SetString("E", "3+5*4")
	out, err := Evaluate([]Token{&TokenRequestNr{Ident: "Y", Value: `\*[E]`}, &TokenRequestIf{Cond: `(\n[X]+1)*2>=10`, Anything: "yes"}}, env)
	if err != nil || env.GetRegister("Y") != 32 || len(out) != 1 || out[0] != TokenText("yes") {
		t.Fatalf("interpolated expressions: %v %v %+v", out, err, env.Registers)
	}
}

func TestEvaluationExpressionFailures(t *testing.T) {
	for _, expression := range []string{"", "()", "(1", "1)", "1+", "1 2", "1px", "1!=2", "1/0", "1%0", "1:(1/0)", "0&(1/0)", "(!1)"} {
		t.Run(expression, func(t *testing.T) {
			env := NewEnvironment()
			env.SetRegister("X", 7, 0)
			_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: expression}}, env)
			if err == nil || env.GetRegister("X") != 7 {
				t.Fatalf("invalid expression assigned: %q, %d, %v", expression, env.GetRegister("X"), err)
			}
		})
	}
	for _, expression := range []string{strings.Repeat("(", 10000) + "1" + strings.Repeat(")", 10000), strings.Repeat("-", 10000) + "1"} {
		env := NewEnvironment()
		env.MaxDepth = 4
		env.MaxSteps = 100
		_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: "(" + expression + ")"}}, env)
		var limit *EvaluationLimitError
		if !errors.As(err, &limit) || env.Depth != 0 {
			t.Fatalf("expression escaped budget: %v", err)
		}
	}
}

// Compare checked operators to unbounded arithmetic; this catches overflow in
// multiplication/division and subtraction without copying the checked formulas.
func TestEvaluationArithmeticAgainstBigIntegers(t *testing.T) {
	max := int(^uint(0) >> 1)
	min := -max - 1
	values := []int{min, min + 1, -max / 2, -100, -3, -1, 0, 1, 2, 3, 100, max / 2, max - 1, max}
	for _, a := range values {
		for _, b := range values {
			for _, op := range []string{"+", "-", "*", "/", "%"} {
				if b == 0 && (op == "/" || op == "%") {
					continue
				}
				left, right := big.NewInt(int64(a)), big.NewInt(int64(b))
				want := new(big.Int)
				switch op {
				case "+":
					want.Add(left, right)
				case "-":
					want.Sub(left, right)
				case "*":
					want.Mul(left, right)
				case "/":
					want.Quo(left, right)
				case "%":
					want.Rem(left, right)
				}
				got, err := evaluationOperation(op, a, b)
				fits := want.Cmp(big.NewInt(int64(min))) >= 0 && want.Cmp(big.NewInt(int64(max))) <= 0
				if fits && (err != nil || int64(got) != want.Int64()) || !fits && err == nil {
					t.Fatalf("%d %s %d: want %s (fits=%v), got %d, %v", a, op, b, want, fits, got, err)
				}
			}
		}
	}
	for _, value := range []int{min, max} {
		env := NewEnvironment()
		_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: "(" + strconv.Itoa(value) + ")"}}, env)
		if err != nil || env.GetRegister("X") != value {
			t.Fatalf("boundary literal %d: %v", value, err)
		}
	}
}

func TestEvaluationRelativeRegisters(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{
		{"-3+5", 2}, {"-(3+5)", 2}, {"+3+5", 18}, {"(-3+5)", 2}, {"--3", 13}, {`\n[Y]+5`, 2},
	} {
		env := NewEnvironment()
		env.SetRegister("X", 10, 0)
		env.SetRegister("Y", -3, 0)
		_, err := Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: tc.input}}, env)
		if err != nil || env.GetRegister("X") != tc.want {
			t.Fatalf("relative %q: want %d, got %d, %v", tc.input, tc.want, env.GetRegister("X"), err)
		}
	}
}

func TestEvaluationAutoIncrement(t *testing.T) {
	env := NewEnvironment()
	env.SetRegister("X", 5, 2)
	env.SetRegister("XY", 0, -2)
	out, err := env.ExpandTextContext(context.Background(), `\nX \n+X \n+[X] \n-X \n-(XY \n[XY] \n+[missing]`)
	if err != nil || out != "5 7 9 7 2 2 0" || env.GetRegister("X") != 7 {
		t.Fatalf("auto-increment profile: %q %v %+v", out, err, env.Registers)
	}
	_, err = Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: "6"}, TokenText(`\n+[X]`), &TokenRequestNr{Ident: "X", Value: "9", Incr: "0"}, TokenText(`\n+[X]`)}, env)
	if err != nil || env.GetRegister("X") != 9 || env.AutoIncrs["X"] != 0 {
		t.Fatalf("omitted/zero increment: %d %+v %v", env.GetRegister("X"), env.AutoIncrs, err)
	}
	max := int(^uint(0) >> 1)
	min := -max - 1
	for _, tc := range []struct {
		value, incr int
		text        string
		want        int
		failure     bool
	}{
		{max, 1, `\n+[X]`, max, true}, {min, 1, `\n-[X]`, min, true}, {0, min, `\n-[X]`, 0, true}, {min, min, `\n-[X]`, 0, false},
	} {
		env := NewEnvironment()
		env.SetRegister("X", tc.value, tc.incr)
		_, err := env.ExpandTextContext(context.Background(), tc.text)
		if (err != nil) != tc.failure || env.GetRegister("X") != tc.want {
			t.Fatalf("auto-increment overflow: %+v, register=%d, err=%v", tc, env.GetRegister("X"), err)
		}
	}
	for _, text := range []string{`\n+`, `\n-[]`, `\n+(X`} {
		if _, err := NewEnvironment().ExpandTextContext(context.Background(), text); err == nil {
			t.Fatalf("malformed increment %q accepted", text)
		}
	}
}

func FuzzEvaluationNumericGrammar(f *testing.F) {
	for _, seed := range []string{"3+5*4", "(3+5)*4", "(-5)%2", "1:(1/0)", "((((1))))", "\\n+[X]+2"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, expression string) {
		if len(expression) > 4096 {
			t.Skip()
		}
		env := NewEnvironment()
		env.MaxDepth = 8
		env.MaxSteps = 200
		env.MaxWorkBytes = 8192
		env.MaxOutputBytes = 4096
		_, _ = Evaluate([]Token{&TokenRequestNr{Ident: "X", Value: expression}}, env)
		if env.Depth != 0 || env.StepCount > 200 {
			t.Fatal("numeric grammar escaped limits")
		}
	})
}

func TestEvaluationPredicateBoundaryAndReferenceNames(t *testing.T) {
	env := NewEnvironment()
	env.SetRegister("µ(", 5, 0)
	env.DefineMacro("M", []string{`.if (\\n(µ( == 5) yes`})
	out, err := Evaluate([]Token{&TokenUnknownControl{Name: ".M"}}, env)
	if err != nil || len(out) != 1 || out[0] != TokenText("yes") {
		t.Fatalf("parentheses in two-rune reference name corrupted predicate: %v %v", out, err)
	}
	env = NewEnvironment()
	env.SetRegister("X", 0, 1)
	env.MaxWorkBytes = 1000
	out, err = Evaluate([]Token{&TokenRequestIf{Cond: "(", Anything: "0) " + strings.Repeat(`\n+[X]`, 10000)}}, env)
	if err != nil || len(out) != 0 || env.GetRegister("X") != 0 {
		t.Fatalf("predicate recovery inspected/executed disabled body: %v %v", out, err)
	}
}
