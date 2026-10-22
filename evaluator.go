package groff

import "context"

// Environment holds mutable state for opt-in evaluation. A zero value is usable.
// It must not be shared by concurrent evaluations or modified during evaluation.
// Limits <= 0 select defaults; larger values are clamped to hard safety ceilings.
// StepCount is cumulative across calls; callers may explicitly reset it to zero.
type Environment struct {
	Registers map[string]int
	AutoIncrs map[string]int
	Strings   map[string]string
	Macros    map[string][]string

	MaxDepth       int
	MaxSteps       int
	MaxOutputBytes int
	MaxWorkBytes   int
	MaxStateBytes  int
	MaxTokens      int
	MaxLineBytes   int

	StepCount, Depth int
	// Diagnostics describes preserved, unevaluated tokens from the latest call.
	Diagnostics []EvaluationDiagnostic
}

// NewEnvironment creates an initialized evaluation environment with default limits.
func NewEnvironment() *Environment {
	env := &Environment{MaxDepth: 50, MaxSteps: 50000, MaxTokens: 50000,
		MaxOutputBytes: 8 << 20, MaxWorkBytes: 32 << 20, MaxStateBytes: 8 << 20, MaxLineBytes: 64 << 10}
	env.init()
	return env
}

func (env *Environment) init() {
	if env.Registers == nil {
		env.Registers = make(map[string]int)
	}
	if env.AutoIncrs == nil {
		env.AutoIncrs = make(map[string]int)
	}
	if env.Strings == nil {
		env.Strings = make(map[string]string)
	}
	if env.Macros == nil {
		env.Macros = make(map[string][]string)
	}
}

func (env *Environment) GetRegister(name string) int { return env.Registers[name] }
func (env *Environment) SetRegister(name string, val, incr int) {
	env.init()
	env.Registers[name] = val
	if incr != 0 {
		env.AutoIncrs[name] = incr
	}
}
func (env *Environment) RemoveRegister(name string) {
	delete(env.Registers, name)
	delete(env.AutoIncrs, name)
}
func (env *Environment) GetString(name string) (string, bool) {
	v, ok := env.Strings[name]
	return v, ok
}
func (env *Environment) SetString(name, val string)    { env.init(); env.Strings[name] = val }
func (env *Environment) AppendString(name, val string) { env.init(); env.Strings[name] += val }
func (env *Environment) DefineMacro(name string, lines []string) {
	env.init()
	env.Macros[name] = append([]string(nil), lines...)
}

// Evaluate runs the bounded, optional legacy evaluation profile. It returns the
// emitted prefix on error; state changes already applied remain in env. Parsing,
// AST construction and rendering do not invoke this function implicitly.
func Evaluate(tokens []Token, env *Environment) ([]Token, error) {
	return EvaluateContext(context.Background(), tokens, env)
}

// EvaluateContext is Evaluate with cancellation. Cancellation and limits are
// checked before interpretation, parsing selected lines and interpolation work.
// Diagnostics use zero-based input token indices and expansion paths, not source
// spans. See docs/evaluation.md for the deliberately limited language profile.
func EvaluateContext(ctx context.Context, tokens []Token, env *Environment) ([]Token, error) {
	m, err := newEvaluation(ctx, env)
	if err != nil {
		return nil, err
	}
	defer func() { m.env.Depth = 0 }()
	for i, tok := range tokens {
		m.index = i
		if err := m.token(tok); err != nil {
			return m.output, m.failure(err)
		}
	}
	return m.output, nil
}

// ExpandTextContext performs one bounded interpolation pass, returning errors
// rather than partial text. It shares the environment's cumulative step budget.
func (env *Environment) ExpandTextContext(ctx context.Context, text string) (string, error) {
	m, err := newEvaluation(ctx, env)
	if err != nil {
		return "", err
	}
	out, err := m.expand(text)
	if err != nil {
		return "", m.failure(err)
	}
	return out, nil
}

// ExpandText is the compatibility wrapper. On failure it returns the original
// text and records the error in Diagnostics. Use ExpandTextContext to handle errors.
func (env *Environment) ExpandText(text string) string {
	out, err := env.ExpandTextContext(context.Background(), text)
	if err == nil {
		return out
	}
	if env != nil {
		env.Diagnostics = append(env.Diagnostics, EvaluationDiagnostic{TokenIndex: -1, Message: err.Error()})
	}
	return text
}
