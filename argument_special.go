package groff

import (
	"errors"
	"strings"
	"unicode/utf8"
)

type argumentCursor struct {
	source string
	pos    int
}

func bindNumericFields(source string, g ArgumentGrammar) ([]string, error) {
	c := argumentCursor{source: source}
	out := make([]string, len(g.Optional))
	count := 0
	for i := range out {
		before := c.pos
		read := c.word
		if g.Numeric[i] {
			read = c.numeric
		}
		value, present, err := read()
		if err != nil {
			return nil, err
		}
		if literal, flag := g.Flags[i]; flag && present && value != literal {
			c.pos = before
			continue
		}
		if !present {
			if !g.Optional[i] {
				return nil, &MissingArgumentsError{Actual: count, Minimum: g.Minimum}
			}
			continue
		}
		count++
		out[i] = value
	}
	if !c.done() {
		return nil, unexpectedNumericArguments(&c, count, len(out))
	}
	if count < g.Minimum {
		return nil, &MissingArgumentsError{Actual: count, Minimum: g.Minimum}
	}
	return out, nil
}

func unexpectedNumericArguments(c *argumentCursor, count, maximum int) error {
	for !c.done() {
		_, present, err := c.numeric()
		if err != nil {
			return err
		}
		if !present {
			break
		}
		count++
	}
	return &UnexpectedArgumentsError{Actual: count, Maximum: maximum}
}

func (c *argumentCursor) space() {
	for c.pos < len(c.source) && (c.source[c.pos] == ' ' || c.source[c.pos] == '\t') {
		c.pos++
	}
}
func argumentComment(source string, pos int) bool {
	return pos+1 < len(source) && source[pos] == '\\' && (source[pos+1] == '"' || source[pos+1] == '#')
}
func (c *argumentCursor) done() bool {
	c.space()
	return c.pos == len(c.source) || argumentComment(c.source, c.pos)
}
func (c *argumentCursor) word() (string, bool, error) {
	c.space()
	if c.done() {
		return "", false, nil
	}
	start := c.pos
	list, err := scanArgumentsLimit(c.source[start:], captureArgumentValues, 1)
	if err != nil {
		var syntax *ArgumentSyntaxError
		if errors.As(err, &syntax) {
			shift := utf8.RuneCountInString(c.source[:start])
			copy := *syntax
			copy.Span.Start += start
			copy.Span.End += start
			copy.RuneStart += shift
			copy.RuneEnd += shift
			return "", false, &copy
		}
		return "", false, err
	}
	c.pos += list.byteStop
	if len(list.values) == 0 {
		return "", false, nil
	}
	return list.values[0], true, nil
}
func (c *argumentCursor) numeric() (string, bool, error) {
	c.space()
	if c.done() {
		return "", false, nil
	}
	if c.source[c.pos] == '"' {
		return c.word()
	}
	start, depth := c.pos, 0
	for c.pos < len(c.source) {
		ch := c.source[c.pos]
		if argumentComment(c.source, c.pos) {
			break
		}
		if depth == 0 && (ch == ' ' || ch == '\t') {
			break
		}
		if ch == '\\' {
			c.pos = argumentReferenceEnd(c.source, c.pos)
			continue
		}
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth < 0 {
				return "", false, &ArgumentBindingError{Offset: c.pos, Reason: "unmatched numeric closing parenthesis"}
			}
		}
		c.pos++
	}
	if depth != 0 {
		return "", false, &ArgumentBindingError{Offset: start, Reason: "unclosed numeric parentheses"}
	}
	return c.source[start:c.pos], c.pos > start, nil
}
func bindRegisterArguments(source string) ([]string, error) {
	c := argumentCursor{source: source}
	out := make([]string, 3)
	name, ok, err := c.word()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &MissingArgumentsError{Minimum: 2}
	}
	value, ok, err := c.numeric()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &MissingArgumentsError{Actual: 1, Minimum: 2}
	}
	out[0], out[1] = name, value
	increment, present, err := c.numeric()
	if err != nil {
		return nil, err
	}
	if present {
		out[2] = increment
	}
	if !c.done() {
		return nil, unexpectedNumericArguments(&c, 3, 3)
	}
	return out, nil
}
func bindConditionalArguments(source string) ([]string, error) {
	c := argumentCursor{source: source}
	c.space()
	start := c.pos
	if c.done() {
		return nil, &MissingArgumentsError{Minimum: 1}
	}
	if c.source[c.pos] == '"' {
		cond, _, err := c.word()
		if err != nil {
			return nil, err
		}
		c.space()
		return []string{cond, c.source[c.pos:]}, nil
	}
	for c.pos < len(c.source) && c.source[c.pos] == '!' {
		c.pos++
		c.space()
	}
	prefix := c.source[start:c.pos]
	var predicate string
	if c.pos < len(c.source) && c.source[c.pos] == '\'' {
		begin := c.pos
		end := begin + 1
		delimiters := 1
		for end < len(c.source) && delimiters < 3 {
			if c.source[end] == '\\' {
				end = argumentReferenceEnd(c.source, end)
				continue
			}
			if c.source[end] == '\'' {
				delimiters++
			}
			end++
		}
		// Retain the earlier one-word missing-final-delimiter shorthand.
		if delimiters < 3 {
			word, ok, err := c.word()
			if err != nil {
				return nil, err
			}
			if !ok || strings.Count(word, "'") < 2 {
				return nil, &ArgumentBindingError{Offset: begin, Reason: "incomplete string comparison"}
			}
			predicate = word
		} else {
			predicate = c.source[begin:end]
			c.pos = end
		}
	} else {
		word, ok, err := c.numeric()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &MissingArgumentsError{Minimum: 1}
		}
		predicate = word
	}
	c.space()
	return []string{prefix + predicate, c.source[c.pos:]}, nil
}
func stripArgumentComment(source string) string {
	for pos := 0; pos < len(source); {
		if argumentComment(source, pos) {
			return source[:pos]
		}
		if source[pos] == '\\' {
			pos = argumentReferenceEnd(source, pos)
		} else {
			pos++
		}
	}
	return source
}
func bindRawArguments(source string, g ArgumentGrammar) ([]string, error) {
	c := argumentCursor{source: source}
	out := make([]string, len(g.Optional))
	for i := 0; i < g.Rest; i++ {
		value, ok, err := c.word()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &MissingArgumentsError{Actual: i, Minimum: g.Minimum}
		}
		out[i] = value
	}
	c.space()
	tail := source[c.pos:]
	if g.Mode == ArgumentsString {
		if strings.HasPrefix(tail, `"`) {
			tail = tail[1:]
		}
		tail = stripArgumentComment(tail)
	}
	if tail == "" && !g.Optional[g.Rest] {
		return nil, &MissingArgumentsError{Actual: g.Rest, Minimum: g.Minimum}
	}
	out[g.Rest] = tail
	return out, nil
}
func bindDelimitedArguments(source string) ([]string, error) {
	c := argumentCursor{source: source}
	c.space()
	start := c.pos
	if c.done() {
		return nil, &MissingArgumentsError{Minimum: 1}
	}
	delimiter, width := utf8.DecodeRuneInString(source[c.pos:])
	c.pos += width
	count := 1
	for c.pos < len(source) && count < 4 {
		if source[c.pos] == '\\' {
			c.pos = argumentReferenceEnd(source, c.pos)
			continue
		}
		rune, size := utf8.DecodeRuneInString(source[c.pos:])
		c.pos += size
		if rune == delimiter {
			count++
		}
	}
	if count != 4 {
		return nil, &ArgumentBindingError{Offset: start, Reason: "expected three delimiter-separated fields"}
	}
	end := c.pos
	if !c.done() {
		return nil, &UnexpectedArgumentsError{Actual: 2, Maximum: 1}
	}
	return []string{source[start:end]}, nil
}

// argumentReferenceEnd skips the spelling of one escape/reference, without
// resolving it. Parentheses or delimiters inside reference names are data.
func argumentReferenceEnd(text string, pos int) int {
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
