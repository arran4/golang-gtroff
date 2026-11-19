package groff

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// expand scans once without materializing an unbounded inline-token slice.
// Inserted string values are opaque: they are deliberately not re-interpolated.
func (m *evaluation) expand(text string) (string, error) {
	if err := m.charge(len(text)); err != nil {
		return "", err
	}
	var out strings.Builder
	write := func(value string) error {
		if len(value) > m.outputLimit-out.Len() {
			return &EvaluationLimitError{Kind: "output byte", Limit: m.outputLimit}
		}
		if err := m.charge(len(value)); err != nil {
			return err
		}
		out.WriteString(value)
		return nil
	}
	for i := 0; i < len(text); {
		start := i
		if text[i] != '\\' {
			end := strings.IndexByte(text[i:], '\\')
			if end < 0 {
				end = len(text) - i
			}
			i += end
			if err := write(text[start:i]); err != nil {
				return "", err
			}
			continue
		}
		if m.copiedReference(text, i) {
			i++
		}
		i++
		if i == len(text) {
			if err := write(text[start:i]); err != nil {
				return "", err
			}
			break
		}
		kind := text[i]
		_, n := utf8.DecodeRuneInString(text[i:])
		i += n
		if kind != 'n' && kind != '*' && (kind != '$' || len(m.frames) == 0) {
			if err := write(text[start:i]); err != nil {
				return "", err
			}
			continue
		}
		direction := byte(0)
		if kind == 'n' && i < len(text) && (text[i] == '+' || text[i] == '-') {
			direction = text[i]
			i++
		}
		if i == len(text) {
			return "", fmt.Errorf("missing interpolation name")
		}
		var name string
		parameterZero := text[i] == '0'
		switch text[i] {
		case '[':
			i++
			end := strings.IndexByte(text[i:], ']')
			if end < 0 {
				return "", fmt.Errorf("unclosed interpolation name")
			}
			name = text[i : i+end]
			i += end + 1
		case '(':
			i++
			begin := i
			for count := 0; count < 2; count++ {
				if i == len(text) {
					return "", fmt.Errorf("short two-character interpolation name")
				}
				_, n := utf8.DecodeRuneInString(text[i:])
				i += n
			}
			name = text[begin:i]
		default:
			_, n := utf8.DecodeRuneInString(text[i:])
			name = text[i : i+n]
			i += n
		}
		if name == "" {
			return "", fmt.Errorf("empty interpolation name")
		}
		value := text[start:i]
		if kind == '$' {
			parameter, err := m.parameter(name, parameterZero)
			if err != nil {
				return "", err
			}
			value = parameter
		} else if kind == 'n' {
			number, err := m.registerReference(name, direction)
			if err != nil {
				return "", err
			}
			value = strconv.Itoa(number)
		} else {
			_, kind, defined, ok, err := m.definition(name)
			if err != nil {
				return "", err
			}
			if ok && kind == EvaluationString {
				value = defined.Text
			}
		}
		if err := write(value); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

func (m *evaluation) registerReference(name string, direction byte) (int, error) {
	if name == ".$" {
		if direction != 0 {
			return 0, fmt.Errorf("parameter-count register is read-only")
		}
		if len(m.frames) == 0 {
			return 0, nil
		}
		return len(m.frames[len(m.frames)-1].args), nil
	}
	register, _, err := m.load(EvaluationRegister, name)
	if err != nil {
		return 0, err
	}
	increment, _, err := m.load(EvaluationIncrement, name)
	if err != nil {
		return 0, err
	}
	value, incr := register.Number, increment.Number
	if direction == 0 || incr == 0 {
		return value, nil
	}
	if direction == '+' {
		value, err = evaluationAdd(value, incr)
	} else {
		value, err = evaluationSubtract(value, incr)
	}
	if err != nil {
		return 0, fmt.Errorf("auto-increment register %q: %w", name, err)
	}
	if err := m.save(EvaluationRegister, name, EvaluationValue{Number: value}); err != nil {
		return 0, err
	}
	return value, nil
}
