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
		if kind != 'n' && kind != '*' {
			if err := write(text[start:i]); err != nil {
				return "", err
			}
			continue
		}
		if kind == 'n' && i < len(text) && (text[i] == '+' || text[i] == '-') {
			return "", fmt.Errorf("register auto-increment escapes are unsupported")
		}
		if i == len(text) {
			return "", fmt.Errorf("missing interpolation name")
		}
		var name string
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
		if kind == 'n' {
			value = strconv.Itoa(m.env.GetRegister(name))
		} else if defined, ok := m.env.GetString(name); ok {
			value = defined
		}
		if err := write(value); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}
