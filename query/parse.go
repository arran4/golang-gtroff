package query

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ParseSelector parses KIND[FIELD OP VALUE] steps joined by whitespace
// (descendant) or > (child). * matches any node. Values may be double-quoted Go
// strings, single-quoted strings with escaped quotes/backslashes/n/r/t, or bare
// ASCII identifiers/unsigned decimals. FIELD without an operator tests existence.
// There are no assignments, variables, callbacks, pseudo-classes or execution.
// Input is bounded to 16 KiB and must be valid UTF-8. Nested brackets, groups and
// unsupported operators are errors rather than silently weakened selectors.
func ParseSelector(input string) (Selector, error) {
	p := selectorParser{input: input}
	if len(input) > 16<<10 || !utf8.ValidString(input) {
		return Selector{}, p.fail("selector must be valid UTF-8 and at most 16 KiB")
	}
	var out Selector
	for {
		p.space()
		start := p.pos
		name := p.identifier()
		if p.peek() == '*' && name == "" {
			name = "*"
			p.pos++
		}
		if name == "" {
			return Selector{}, p.fail("expected node kind")
		}
		current := Select(Kind(name))
		for p.peek() == '[' {
			p.pos++
			p.space()
			field := p.identifier()
			if field == "" {
				return Selector{}, p.fail("expected attribute field")
			}
			p.space()
			operator := Exists
			value := ""
			if p.peek() != ']' {
				found := false
				for _, op := range []Operator{NotEqual, Contains, Prefix, Suffix, LessEqual, GreaterEqual, Equal, Less, Greater} {
					if strings.HasPrefix(input[p.pos:], string(op)) {
						p.pos += len(op)
						operator = op
						found = true
						break
					}
				}
				if !found {
					return Selector{}, p.fail("expected predicate operator or closing bracket")
				}
				p.space()
				var err error
				if p.peek() == '\'' || p.peek() == '"' {
					value, err = p.quoted()
				} else {
					value = p.identifier()
					if value == "" {
						err = p.fail("expected predicate value")
					}
				}
				if err != nil {
					return Selector{}, err
				}
				p.space()
			}
			if p.peek() != ']' {
				return Selector{}, p.fail("expected closing bracket")
			}
			p.pos++
			current = current.Where(Field(field), operator, value)
		}
		if len(out.steps) == 0 {
			out = current
		} else {
			current.steps[0].relation = p.pending
			out.steps = append(out.steps, current.steps...)
		}
		if err := out.Validate(); err != nil {
			return Selector{}, &SyntaxError{start, err.Error()}
		}
		spaced := p.space()
		if p.pos == len(input) {
			break
		}
		p.pending = descendant
		if p.peek() == '>' {
			p.pending = child
			p.pos++
			p.space()
		} else if !spaced {
			return Selector{}, p.fail("expected whitespace or > between steps")
		}
	}
	return out, nil
}

type selectorParser struct {
	input   string
	pos     int
	pending relation
}

func (p *selectorParser) fail(message string) error { return &SyntaxError{p.pos, message} }
func (p *selectorParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}
func (p *selectorParser) space() bool {
	start := p.pos
	for p.pos < len(p.input) && strings.ContainsRune(" \t\n\r", rune(p.input[p.pos])) {
		p.pos++
	}
	return p.pos != start
}
func (p *selectorParser) identifier() string {
	start := p.pos
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos]
}
func (p *selectorParser) quoted() (string, error) {
	quote, start := p.peek(), p.pos
	p.pos++
	if quote == '"' {
		escaped := false
		for p.pos < len(p.input) {
			c := p.input[p.pos]
			p.pos++
			if c == '"' && !escaped {
				value, err := strconv.Unquote(p.input[start:p.pos])
				if err != nil {
					return "", p.fail("invalid quoted string: " + err.Error())
				}
				return value, nil
			}
			if c == '\\' {
				escaped = !escaped
			} else {
				escaped = false
			}
		}
		return "", p.fail("unterminated quoted string")
	}
	var out strings.Builder
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		p.pos++
		if c == quote {
			return out.String(), nil
		}
		if c == '\\' {
			if p.pos == len(p.input) {
				break
			}
			c = p.input[p.pos]
			p.pos++
			switch c {
			case '\\', '\'':
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			default:
				return "", p.fail(fmt.Sprintf("unsupported single-quote escape %q", c))
			}
		} else if c == '\n' || c == '\r' {
			return "", p.fail("literal newline in quoted string")
		}
		out.WriteByte(c)
	}
	return "", p.fail("unterminated quoted string")
}
