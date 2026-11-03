package groff

import (
	"strings"
	"unicode/utf8"
)

// The legacy argument parser can split a parenthesized expression across the
// first/rest fields. Recover only that expression; never interpolate a disabled
// branch while finding its boundary. Source tokens remain unchanged.
func (m *evaluation) expressionParts(first, rest string) (string, string, error) {
	if err := m.charge(len(first)); err != nil {
		return "", "", err
	}
	depth := 0
	for i := 0; i < len(first); {
		if first[i] == '\\' {
			i = evaluationReferenceEnd(first, i)
			continue
		}
		if first[i] == '(' {
			depth++
		}
		if first[i] == ')' {
			depth--
		}
		i++
	}
	if depth <= 0 {
		return first, rest, nil
	}
	if len(first) >= m.lineLimit {
		return "", "", &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
	}
	scan := rest
	if len(scan) > m.lineLimit-len(first) {
		scan = scan[:m.lineLimit-len(first)]
	}
	end := 0
	for end < len(scan) {
		if len(first)+1+end >= m.lineLimit {
			return "", "", &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
		}
		if end%256 == 0 {
			if err := m.charge(256); err != nil {
				return "", "", err
			}
		}
		if depth == 0 && (scan[end] == ' ' || scan[end] == '\t') {
			break
		}
		if scan[end] == '\\' {
			start := end
			end = evaluationReferenceEnd(scan, end)
			if err := m.charge(end - start); err != nil {
				return "", "", err
			}
			continue
		}
		if rest[end] == '(' {
			depth++
		}
		if rest[end] == ')' {
			depth--
		}
		end++
	}
	if len(rest[:end]) > m.lineLimit-len(first)-1 {
		return "", "", &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
	}
	if err := m.charge(len(first) + 1 + end); err != nil {
		return "", "", err
	}
	return first + " " + rest[:end], rest[end:], nil
}
func evaluationReferenceEnd(text string, pos int) int {
	pos++
	if pos == len(text) {
		return pos
	}
	kind := text[pos]
	if kind == '\\' && pos+1 < len(text) && strings.ContainsRune("n*$", rune(text[pos+1])) {
		pos++
		kind = text[pos]
	}
	pos++
	if kind != 'n' && kind != '*' && kind != '$' {
		return pos
	}
	if kind == 'n' && pos < len(text) && (text[pos] == '+' || text[pos] == '-') {
		pos++
	}
	if pos == len(text) {
		return pos
	}
	switch text[pos] {
	case '[':
		end := strings.IndexByte(text[pos+1:], ']')
		if end < 0 {
			return len(text)
		}
		return pos + end + 2
	case '(':
		pos++
		for count := 0; count < 2 && pos < len(text); count++ {
			_, n := utf8.DecodeRuneInString(text[pos:])
			pos += n
		}
		return pos
	default:
		_, n := utf8.DecodeRuneInString(text[pos:])
		return pos + n
	}
}
func (m *evaluation) conditional(cond, body string) (bool, string, error) {
	if err := m.charge(len(cond)); err != nil {
		return false, "", err
	}
	// Quoted string comparison is distinct from numeric grouping.
	predicate := strings.TrimLeft(cond, "! \t")
	if strings.HasPrefix(predicate, "'") {
		value, err := m.condition(cond)
		return value, body, err
	}
	condition, rest, err := m.expressionParts(cond, body)
	if err != nil {
		return false, "", err
	}
	value, err := m.condition(condition)
	return value, rest, err
}
