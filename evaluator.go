package groff

import (
	"fmt"
	"strconv"
	"strings"
)

// Environment holds the runtime state for register, string, and macro evaluation.
type Environment struct {
	Registers  map[string]int
	AutoIncrs  map[string]int
	Strings    map[string]string
	Macros     map[string][]string
	MaxDepth   int
	MaxSteps   int
	StepCount  int
	Depth      int
	lastIfCond bool
}

// NewEnvironment creates an initialized, bounded evaluation environment.
func NewEnvironment() *Environment {
	return &Environment{
		Registers: make(map[string]int),
		AutoIncrs: make(map[string]int),
		Strings:   make(map[string]string),
		Macros:    make(map[string][]string),
		MaxDepth:  50,
		MaxSteps:  50000,
	}
}

func (env *Environment) GetRegister(name string) int {
	return env.Registers[name]
}

func (env *Environment) SetRegister(name string, val int, incr int) {
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

func (env *Environment) SetString(name string, val string) {
	env.Strings[name] = val
}

func (env *Environment) AppendString(name string, val string) {
	env.Strings[name] = env.Strings[name] + val
}

func (env *Environment) DefineMacro(name string, lines []string) {
	copied := make([]string, len(lines))
	copy(copied, lines)
	env.Macros[name] = copied
}

// Evaluate performs an optional bounded semantic evaluation pass over a sequence of tokens.
func Evaluate(tokens []Token, env *Environment) ([]Token, error) {
	if env == nil {
		env = NewEnvironment()
	}

	var result []Token
	for _, tok := range tokens {
		env.StepCount++
		if env.StepCount > env.MaxSteps {
			return nil, fmt.Errorf("evaluation step budget exceeded (%d steps)", env.MaxSteps)
		}

		switch t := tok.(type) {
		case *TokenRequestNr:
			handleNr(t, env)

		case *TokenRequestRr:
			if t.Ident != "" {
				env.RemoveRegister(t.Ident)
			}

		case *TokenRequestDs:
			env.SetString(t.Name, t.StringVal)

		case *TokenRequestAs:
			env.AppendString(t.Name, t.StringVal)

		case *TokenMacroDefinition:
			env.DefineMacro(t.Name, t.Body)

		case *TokenRequestIf:
			branchToks, err := handleIf(t.Cond, t.Anything, env, false)
			if err != nil {
				return nil, err
			}
			result = append(result, branchToks...)

		case *TokenRequestIe:
			branchToks, err := handleIf(t.Cond, t.Anything, env, true)
			if err != nil {
				return nil, err
			}
			result = append(result, branchToks...)

		case *TokenRequestEl:
			if !env.lastIfCond && t.Anything != "" {
				parsed, err := ParseLinePermissive(t.Anything, "")
				if err == nil {
					subRes, err := Evaluate([]Token{parsed}, env)
					if err != nil {
						return nil, err
					}
					result = append(result, subRes...)
				}
			}

		case TokenText:
			expanded := env.ExpandText(string(t))
			result = append(result, TokenText(expanded))

		default:
			// Check if token represents dynamic macro invocation
			if n, ok := tok.(Node); ok {
				cmd := strings.TrimPrefix(n.TokenName(), ".")
				if body, hasMacro := env.Macros[cmd]; hasMacro {
					if env.Depth >= env.MaxDepth {
						return nil, fmt.Errorf("maximum macro recursion depth exceeded (%d)", env.MaxDepth)
					}
					env.Depth++
					var macroTokens []Token
					for _, line := range body {
						parsed, err := ParseLinePermissive(line, "")
						if err == nil {
							macroTokens = append(macroTokens, parsed)
						}
					}
					expanded, err := Evaluate(macroTokens, env)
					env.Depth--
					if err != nil {
						return nil, err
					}
					result = append(result, expanded...)
					continue
				}
			}
			result = append(result, tok)
		}
	}

	return result, nil
}

func handleNr(t *TokenRequestNr, env *Environment) {
	if t.Ident == "" {
		return
	}
	valStr := strings.TrimSpace(t.Value)
	if valStr == "" {
		return
	}

	var val int
	if strings.HasPrefix(valStr, "+") {
		delta, _ := strconv.Atoi(valStr[1:])
		val = env.GetRegister(t.Ident) + delta
	} else if strings.HasPrefix(valStr, "-") {
		delta, _ := strconv.Atoi(valStr[1:])
		val = env.GetRegister(t.Ident) - delta
	} else {
		val, _ = strconv.Atoi(valStr)
	}

	incr := 0
	if t.Incr != "" {
		incr, _ = strconv.Atoi(t.Incr)
	}
	env.SetRegister(t.Ident, val, incr)
}

func splitNameAndValue(nameArg, valArg string) (string, string) {
	if valArg != "" {
		return strings.TrimSpace(nameArg), strings.TrimSpace(valArg)
	}
	parts := strings.Fields(nameArg)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	name := parts[0]
	val := strings.TrimSpace(nameArg[len(name):])
	return name, val
}

func handleIf(condStr, rest string, env *Environment, isIe bool) ([]Token, error) {
	cond := evaluateCondition(condStr, env)
	if isIe {
		env.lastIfCond = cond
	}
	if !cond || rest == "" {
		return nil, nil
	}

	parsed, err := ParseLinePermissive(rest, "")
	if err != nil {
		return nil, nil
	}
	return Evaluate([]Token{parsed}, env)
}

func evaluateCondition(cond string, env *Environment) bool {
	cond = strings.TrimSpace(cond)
	if strings.HasPrefix(cond, "!") {
		return !evaluateCondition(strings.TrimSpace(cond[1:]), env)
	}

	// String comparison: 'str1'str2'
	if strings.HasPrefix(cond, "'") {
		parts := strings.Split(cond[1:], "'")
		if len(parts) >= 2 {
			s1 := env.ExpandText(parts[0])
			s2 := env.ExpandText(parts[1])
			return s1 == s2
		}
	}

	// Check register equality/comparison: \n[reg] > 0 or \n[reg]==0
	if strings.Contains(cond, ">") {
		p := strings.SplitN(cond, ">", 2)
		v1 := evaluateNumeric(p[0], env)
		v2 := evaluateNumeric(p[1], env)
		return v1 > v2
	}
	if strings.Contains(cond, "<") {
		p := strings.SplitN(cond, "<", 2)
		v1 := evaluateNumeric(p[0], env)
		v2 := evaluateNumeric(p[1], env)
		return v1 < v2
	}
	if strings.Contains(cond, "==") {
		p := strings.SplitN(cond, "==", 2)
		v1 := evaluateNumeric(p[0], env)
		v2 := evaluateNumeric(p[1], env)
		return v1 == v2
	}

	// Default roff built-ins: 'n' (nroff) mode is true, 't' (troff) mode is false
	if cond == "n" {
		return true
	}
	if cond == "t" {
		return false
	}

	// Evaluates as non-zero number
	n := evaluateNumeric(cond, env)
	return n > 0
}

func evaluateNumeric(s string, env *Environment) int {
	s = strings.TrimSpace(s)
	// Check if register expansion
	if strings.HasPrefix(s, `\n[`) && strings.HasSuffix(s, "]") {
		regName := s[3 : len(s)-1]
		return env.GetRegister(regName)
	}
	val, _ := strconv.Atoi(s)
	return val
}

// ExpandText replaces register and string references (\*[str] and \n[reg]) with their current environment values.
func (env *Environment) ExpandText(text string) string {
	tokens, err := ParseInline(text)
	if err != nil {
		return text
	}

	var sb strings.Builder
	for _, tok := range tokens {
		switch tok.Kind {
		case InlineStringRef:
			if val, ok := env.GetString(tok.Name); ok {
				sb.WriteString(val)
			} else {
				sb.WriteString(tok.Raw)
			}
		case InlineRegisterRef:
			val := env.GetRegister(tok.Name)
			sb.WriteString(strconv.Itoa(val))
		default:
			sb.WriteString(tok.Raw)
		}
	}
	return sb.String()
}
