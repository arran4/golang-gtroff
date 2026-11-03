package groff

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// EvaluationDiagnostic reports a token that remains visible without evaluation.
type EvaluationDiagnostic struct {
	TokenIndex int
	Expansion  []string
	Message    string
}

// EvaluationError identifies the outer token and nested expansion at a failure.
type EvaluationError struct {
	TokenIndex int
	Expansion  []string
	Err        error
}

func (e *EvaluationError) Error() string {
	return fmt.Sprintf("evaluate token %d (%s): %v", e.TokenIndex, strings.Join(e.Expansion, " / "), e.Err)
}
func (e *EvaluationError) Unwrap() error { return e.Err }

// EvaluationLimitError reports a resource budget failure. Kind is one of step,
// work byte, output byte, output token, state byte, line byte, or recursion depth.
type EvaluationLimitError struct {
	Kind  string
	Limit int
}

func (e *EvaluationLimitError) Error() string {
	return fmt.Sprintf("evaluation %s exceeded (%d)", e.Kind+" budget", e.Limit)
}

type evaluation struct {
	ctx     context.Context
	env     *Environment
	index   int
	path    []string
	pending []bool
	output  []Token

	work, outputBytes, stateBytes int
	sizes                         map[string]int

	depthLimit  int
	stepLimit   int
	workLimit   int
	outputLimit int
	stateLimit  int
	tokenLimit  int
	lineLimit   int

	registry Registry // Isolate expansion parsing from process-global handlers.
}

func evaluationBound(value, def, ceiling int) int {
	if value <= 0 {
		return def
	}
	if value > ceiling {
		return ceiling
	}
	return value
}

func newEvaluation(ctx context.Context, env *Environment) (*evaluation, error) {
	if env == nil {
		env = NewEnvironment()
	}
	env.init()
	env.Diagnostics = nil
	env.Depth = 0
	if ctx == nil {
		return nil, fmt.Errorf("nil evaluation context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m := &evaluation{ctx: ctx, env: env, index: -1, sizes: make(map[string]int),
		depthLimit:  evaluationBound(env.MaxDepth, 50, 128),
		stepLimit:   evaluationBound(env.MaxSteps, 50000, 1000000),
		workLimit:   evaluationBound(env.MaxWorkBytes, 32<<20, 256<<20),
		outputLimit: evaluationBound(env.MaxOutputBytes, 8<<20, 128<<20),
		stateLimit:  evaluationBound(env.MaxStateBytes, 8<<20, 128<<20),
		tokenLimit:  evaluationBound(env.MaxTokens, 50000, 1000000)}
	m.lineLimit = evaluationBound(env.MaxLineBytes, 64<<10, 1<<20)
	if env.StepCount < 0 {
		return nil, fmt.Errorf("negative evaluation StepCount")
	}
	// Account for caller-supplied state before interpreting any token. The fixed
	// overhead also bounds empty names, values, body lines and map entries.
	for name := range env.Registers {
		if err := m.state("r", name, 32); err != nil {
			return nil, m.failure(err)
		}
	}
	for name := range env.AutoIncrs {
		if err := m.state("i", name, 32); err != nil {
			return nil, m.failure(err)
		}
	}
	for name, value := range env.Strings {
		if err := m.state("s", name, len(value)+32); err != nil {
			return nil, m.failure(err)
		}
	}
	for name, body := range env.Macros {
		size, err := m.bodySize(body)
		if err != nil {
			return nil, m.failure(err)
		}
		if err := m.state("m", name, size); err != nil {
			return nil, m.failure(err)
		}
	}
	return m, nil
}

func (m *evaluation) failure(err error) error {
	return &EvaluationError{TokenIndex: m.index, Expansion: append([]string(nil), m.path...), Err: err}
}
func (m *evaluation) tick() error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if m.env.StepCount >= m.stepLimit {
		return &EvaluationLimitError{Kind: "step", Limit: m.stepLimit}
	}
	m.env.StepCount++
	return nil
}
func (m *evaluation) charge(n int) error {
	if err := m.tick(); err != nil {
		return err
	}
	if n > m.workLimit-m.work {
		return &EvaluationLimitError{Kind: "work byte", Limit: m.workLimit}
	}
	m.work += n
	return nil
}
func (m *evaluation) state(kind, name string, size int) error {
	if err := m.charge(len(name)); err != nil {
		return err
	}
	if size < 0 || len(name) > m.stateLimit || size > m.stateLimit-len(name) {
		return &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
	}
	size += len(name)
	key := kind + name
	if size-m.sizes[key] > m.stateLimit-m.stateBytes {
		return &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
	}
	m.stateBytes += size - m.sizes[key]
	m.sizes[key] = size
	return nil
}
func (m *evaluation) removeState(kind, name string) {
	key := kind + name
	m.stateBytes -= m.sizes[key]
	delete(m.sizes, key)
}
func (m *evaluation) bodySize(body []string) (int, error) {
	size := 32
	for _, line := range body {
		if err := m.tick(); err != nil {
			return 0, err
		}
		if len(line) > m.stateLimit-size-16 {
			return 0, &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
		}
		size += len(line) + 16
	}
	return size, nil
}
func (m *evaluation) emit(tok Token) error {
	if len(m.output) >= m.tokenLimit {
		return &EvaluationLimitError{Kind: "output token", Limit: m.tokenLimit}
	}
	if text, ok := tok.(TokenText); ok {
		if len(text) > m.outputLimit-m.outputBytes {
			return &EvaluationLimitError{Kind: "output byte", Limit: m.outputLimit}
		}
		m.outputBytes += len(text)
	}
	m.output = append(m.output, tok)
	return nil
}
func (m *evaluation) preserve(tok Token, message string) error {
	if err := m.emit(tok); err != nil {
		return err
	}
	m.env.Diagnostics = append(m.env.Diagnostics, EvaluationDiagnostic{TokenIndex: m.index, Expansion: append([]string(nil), m.path...), Message: message})
	return nil
}
func (m *evaluation) nested(label string, f func() error) error {
	if m.env.Depth >= m.depthLimit {
		return fmt.Errorf("maximum recursion depth exceeded (%d): %w", m.depthLimit, &EvaluationLimitError{Kind: "recursion depth", Limit: m.depthLimit})
	}
	m.env.Depth++
	m.path = append(m.path, label)
	err := f()
	m.env.Depth--
	if err != nil {
		return err
	} // Keep the failing path for the outer wrapper.
	m.path = m.path[:len(m.path)-1]
	return nil
}
func (m *evaluation) line(line string) error {
	if len(line) > m.lineLimit {
		return &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
	}
	if err := m.charge(len(line)); err != nil {
		return err
	}
	tok, err := m.registry.ParseLinePermissive(line, "")
	if err != nil {
		return fmt.Errorf("parse expansion: %w", err)
	}
	return m.token(tok)
}
func (m *evaluation) branch(body string) error {
	if len(body) > m.lineLimit {
		return &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
	}
	if err := m.charge(len(body)); err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(body), `\{`) {
		return fmt.Errorf("multiline conditional blocks are unsupported")
	}
	return m.nested("conditional", func() error { return m.line(body) })
}
func (m *evaluation) token(tok Token) error {
	if err := m.tick(); err != nil {
		return err
	}
	if nilInterface(tok) {
		return fmt.Errorf("nil evaluation token")
	}
	switch t := tok.(type) {
	case *TokenRequestNr:
		return m.register(t)
	case *TokenRequestRr:
		if err := m.charge(len(t.Ident)); err != nil {
			return err
		}
		m.env.RemoveRegister(t.Ident)
		m.removeState("r", t.Ident)
		m.removeState("i", t.Ident)
	case *TokenRequestDs:
		if err := m.charge(len(t.StringVal)); err != nil {
			return err
		}
		if err := m.state("s", t.Name, len(t.StringVal)+32); err != nil {
			return err
		}
		m.env.SetString(t.Name, t.StringVal)
	case *TokenRequestAs:
		if err := m.charge(len(t.Name)); err != nil {
			return err
		}
		if err := m.charge(len(t.StringVal)); err != nil {
			return err
		}
		old := m.env.Strings[t.Name]
		if len(t.StringVal) > m.stateLimit-len(old)-32 {
			return &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
		}
		if err := m.state("s", t.Name, len(old)+len(t.StringVal)+32); err != nil {
			return err
		}
		if err := m.charge(len(old)); err != nil {
			return err
		}
		m.env.AppendString(t.Name, t.StringVal)
	case *TokenMacroDefinition:
		size, err := m.bodySize(t.Body)
		if err != nil {
			return err
		}
		if err := m.state("m", t.Name, size); err != nil {
			return err
		}
		m.env.DefineMacro(t.Name, t.Body)
	case *TokenRequestIf:
		cond, err := m.condition(t.Cond)
		if err != nil {
			return err
		}
		if cond {
			return m.branch(t.Anything)
		}
	case *TokenRequestIe:
		cond, err := m.condition(t.Cond)
		if err != nil {
			return err
		}
		// The pending stack also consumes steps even when no output is emitted.
		m.pending = append(m.pending, cond)
		if cond {
			return m.branch(t.Anything)
		}
	case *TokenRequestEl:
		if len(m.pending) == 0 {
			return fmt.Errorf("unpaired .el")
		}
		last := len(m.pending) - 1
		cond := m.pending[last]
		m.pending = m.pending[:last]
		if !cond {
			return m.branch(t.Anything)
		}
	case TokenText:
		expanded, err := m.expand(string(t))
		if err != nil {
			return err
		}
		return m.emit(TokenText(expanded))
	case TokenComment, TokenEmptyControl:
		return m.emit(tok)
	default:
		if node, ok := tok.(Node); ok {
			rawName := node.TokenName()
			if err := m.charge(len(rawName)); err != nil {
				return err
			}
			name := strings.TrimLeft(rawName, ".'")
			if body, ok := m.env.Macros[name]; ok {
				// Argument substitution and request shadowing require a richer
				// grammar. Preserve argument-bearing calls instead of dropping args.
				unknown, ok := tok.(*TokenUnknownControl)
				if !ok {
					return m.preserve(tok, "macro invocation requires an unknown control without arguments")
				}
				if err := m.charge(len(unknown.RawArgs)); err != nil {
					return err
				}
				if strings.TrimSpace(unknown.RawArgs) != "" {
					return m.preserve(tok, "macro invocation requires an unknown control without arguments")
				}
				return m.nested("macro "+name, func() error {
					for i, line := range body {
						if err := m.nested(fmt.Sprintf("body line %d", i+1), func() error { return m.line(line) }); err != nil {
							return err
						}
					}
					return nil
				})
			}
		}
		return m.preserve(tok, "token preserved without evaluation")
	}
	return nil
}

func (m *evaluation) integer(text string) (int, error) {
	if err := m.charge(len(text)); err != nil {
		return 0, err
	}
	expanded, err := m.expand(strings.TrimSpace(text))
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(expanded)
	if err != nil {
		return 0, fmt.Errorf("unsupported or invalid integer %q: %w", expanded, err)
	}
	return value, nil
}
func evaluationAdd(a, b int) (int, error) {
	max := int(^uint(0) >> 1)
	min := -max - 1
	if b > 0 && a > max-b || b < 0 && a < min-b {
		return 0, fmt.Errorf("register arithmetic overflow")
	}
	return a + b, nil
}
func (m *evaluation) register(t *TokenRequestNr) error {
	if err := m.charge(len(t.Ident)); err != nil {
		return err
	}
	if err := m.charge(len(t.Value)); err != nil {
		return err
	}
	valueText := strings.TrimSpace(t.Value)
	if t.Ident == "" || valueText == "" {
		return fmt.Errorf(".nr requires name and value")
	}
	value, err := m.integer(valueText)
	if err != nil {
		return err
	}
	if strings.HasPrefix(valueText, "+") || strings.HasPrefix(valueText, "-") {
		value, err = evaluationAdd(m.env.GetRegister(t.Ident), value)
		if err != nil {
			return err
		}
	}
	incr := 0
	if t.Incr != "" {
		incr, err = m.integer(t.Incr)
		if err != nil {
			return err
		}
	}
	if err := m.state("r", t.Ident, 32); err != nil {
		return err
	}
	if incr != 0 {
		if err := m.state("i", t.Ident, 32); err != nil {
			return err
		}
	}
	m.env.SetRegister(t.Ident, value, incr)
	return nil
}
func (m *evaluation) condition(cond string) (bool, error) {
	if err := m.charge(len(cond)); err != nil {
		return false, err
	}
	cond = strings.TrimSpace(cond)
	negate := false
	for strings.HasPrefix(cond, "!") {
		if err := m.tick(); err != nil {
			return false, err
		}
		negate = !negate
		cond = strings.TrimSpace(cond[1:])
	}
	var value bool
	var err error
	switch {
	case cond == "n":
		value = true
	case cond == "t":
		value = false
	case strings.HasPrefix(cond, "'"):
		// Accept the historical missing-final-delimiter shorthand for compatibility.
		first, rest, ok := strings.Cut(cond[1:], "'")
		second, tail, closed := strings.Cut(rest, "'")
		if !ok || closed && tail != "" {
			return false, fmt.Errorf("invalid string comparison")
		}
		left, e := m.expand(first)
		if e != nil {
			return false, e
		}
		right, e := m.expand(second)
		if e != nil {
			return false, e
		}
		value = left == right
	default:
		matched := false
		for _, op := range []string{"==", ">", "<"} {
			if i := strings.Index(cond, op); i >= 0 {
				left, e := m.integer(cond[:i])
				if e != nil {
					return false, e
				}
				right, e := m.integer(cond[i+len(op):])
				if e != nil {
					return false, e
				}
				switch op {
				case "==":
					value = left == right
				case ">":
					value = left > right
				case "<":
					value = left < right
				}
				matched = true
				break
			}
		}
		if !matched {
			var n int
			n, err = m.integer(cond)
			value = n > 0
		}
	}
	if negate {
		value = !value
	}
	return value, err
}
