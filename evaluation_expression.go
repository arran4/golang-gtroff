package groff

import (
	"fmt"
	"strconv"
	"strings"
)

// expression evaluates the integer profile with roff's left-to-right order.
// Interpolation precedes arithmetic; it may mutate auto-incremented registers.
func (m *evaluation) expression(text string) (int, error) {
	if err := m.charge(len(text)); err != nil {
		return 0, err
	}
	expanded, err := m.expand(text)
	if err != nil {
		return 0, err
	}
	return m.parseExpression(expanded)
}

func (m *evaluation) parseExpression(expanded string) (int, error) {
	if err := m.charge(len(expanded)); err != nil {
		return 0, err
	}
	p := evaluationExpression{m: m, text: expanded}
	value, err := p.sequence()
	if err != nil {
		return 0, err
	}
	p.space()
	if p.pos != len(p.text) {
		return 0, p.invalid("unexpected input")
	}
	return value, nil
}

type evaluationExpression struct {
	m    *evaluation
	text string
	pos  int
}

func (p *evaluationExpression) invalid(reason string) error {
	return fmt.Errorf("invalid numeric expression at byte %d: %s", p.pos, reason)
}
func (p *evaluationExpression) space() {
	for p.pos < len(p.text) && (p.text[p.pos] == ' ' || p.text[p.pos] == '\t' || p.text[p.pos] == '\r' || p.text[p.pos] == '\n') {
		p.pos++
	}
}
func (p *evaluationExpression) sequence() (int, error) {
	left, err := p.operand()
	if err != nil {
		return 0, err
	}
	for {
		p.space()
		if p.pos == len(p.text) || p.text[p.pos] == ')' {
			return left, nil
		}
		if err := p.m.tick(); err != nil {
			return 0, err
		}
		start := p.pos
		p.pos++
		op := p.text[start:p.pos]
		if strings.ContainsAny(op, "<>=") && p.pos < len(p.text) && p.text[p.pos] == '=' {
			p.pos++
			op = p.text[start:p.pos]
		}
		switch op {
		case "+", "-", "*", "/", "%", "<", ">", "<=", ">=", "=", "==", "&", ":":
		default:
			return 0, p.invalid("unsupported operator")
		}
		right, err := p.operand()
		if err != nil {
			return 0, err
		}
		left, err = evaluationOperation(op, left, right)
		if err != nil {
			return 0, fmt.Errorf("numeric operator %s: %w", op, err)
		}
	}
}
func (p *evaluationExpression) operand() (int, error) {
	if err := p.m.tick(); err != nil {
		return 0, err
	}
	p.space()
	negative := false
	for p.pos < len(p.text) && (p.text[p.pos] == '+' || p.text[p.pos] == '-') {
		if err := p.m.tick(); err != nil {
			return 0, err
		}
		if p.text[p.pos] == '-' {
			negative = !negative
		}
		p.pos++
		p.space()
	}
	if p.pos == len(p.text) {
		return 0, p.invalid("missing operand")
	}
	if p.text[p.pos] == '(' {
		p.pos++
		var value int
		err := p.m.nested("numeric parentheses", func() error {
			var err error
			value, err = p.sequence()
			if err != nil {
				return err
			}
			p.space()
			if p.pos == len(p.text) || p.text[p.pos] != ')' {
				return p.invalid("missing closing parenthesis")
			}
			p.pos++
			return nil
		})
		if err != nil {
			return 0, err
		}
		if negative {
			return evaluationSubtract(0, value)
		}
		return value, nil
	}
	start := p.pos
	// Consume digits with work checkpoints, without building an AST or slice.
	for p.pos < len(p.text) && p.text[p.pos] >= '0' && p.text[p.pos] <= '9' {
		if err := p.m.tick(); err != nil {
			return 0, err
		}
		p.pos++
	}
	if p.pos == start {
		return 0, p.invalid("expected decimal integer or parenthesis")
	}
	magnitude, err := strconv.ParseUint(p.text[start:p.pos], 10, strconv.IntSize)
	if err != nil {
		return 0, p.invalid("integer overflow")
	}
	max := uint64(^uint(0) >> 1)
	if negative {
		if magnitude > max+1 {
			return 0, p.invalid("integer overflow")
		}
		if magnitude == max+1 {
			return -int(max) - 1, nil
		}
		return -int(magnitude), nil
	}
	if magnitude > max {
		return 0, p.invalid("integer overflow")
	}
	return int(magnitude), nil
}

func evaluationSubtract(a, b int) (int, error) {
	max := int(^uint(0) >> 1)
	min := -max - 1
	if b > 0 && a < min+b || b < 0 && a > max+b {
		return 0, fmt.Errorf("integer overflow")
	}
	return a - b, nil
}
func evaluationOperation(op string, a, b int) (int, error) {
	truth := func(value bool) int {
		if value {
			return 1
		}
		return 0
	}
	switch op {
	case "+":
		return evaluationAdd(a, b)
	case "-":
		return evaluationSubtract(a, b)
	case "*":
		max := int(^uint(0) >> 1)
		min := -max - 1
		if a == 0 || b == 0 {
			return 0, nil
		}
		if a == -1 && b == min || b == -1 && a == min {
			return 0, fmt.Errorf("integer overflow")
		}
		product := a * b
		if product/b != a {
			return 0, fmt.Errorf("integer overflow")
		}
		return product, nil
	case "/", "%":
		if b == 0 {
			return 0, fmt.Errorf("division or modulus by zero")
		}
		min := -int(^uint(0)>>1) - 1
		if a == min && b == -1 {
			if op == "%" {
				return 0, nil
			}
			return 0, fmt.Errorf("integer overflow")
		}
		if op == "/" {
			return a / b, nil
		}
		return a % b, nil
	case "<":
		return truth(a < b), nil
	case ">":
		return truth(a > b), nil
	case "<=":
		return truth(a <= b), nil
	case ">=":
		return truth(a >= b), nil
	case "=", "==":
		return truth(a == b), nil
	case "&":
		return truth(a > 0 && b > 0), nil
	case ":":
		return truth(a > 0 || b > 0), nil
	}
	return 0, fmt.Errorf("unsupported numeric operator %q", op)
}
