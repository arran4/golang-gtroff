package groff

import (
	"strings"
)

// InlineKind classifies inline text spans and escape sequences.
type InlineKind string

const (
	InlineText        InlineKind = "Text"
	InlineStringRef   InlineKind = "StringRef"
	InlineRegisterRef InlineKind = "RegisterRef"
	InlineSpecialChar InlineKind = "SpecialChar"
	InlineFontChange  InlineKind = "FontChange"
	InlineSizeChange  InlineKind = "SizeChange"
	InlineEscape      InlineKind = "Escape"
)

// InlineToken represents an inline segment of text or an escape sequence.
type InlineToken struct {
	Kind  InlineKind
	Raw   string
	Name  string
	Value string
}

func (t InlineToken) String() string {
	return t.Raw
}

// EscapeDecoder is an interface for custom inline escape expansion.
type EscapeDecoder interface {
	DecodeEscape(tok InlineToken) (string, bool)
}

// ParseInline parses an input text string into inline tokens (text spans, escapes, references).
func ParseInline(text string) ([]InlineToken, error) {
	var tokens []InlineToken
	rs := []rune(text)
	n := len(rs)

	var textBuf strings.Builder

	flushText := func() {
		if textBuf.Len() > 0 {
			raw := textBuf.String()
			tokens = append(tokens, InlineToken{
				Kind:  InlineText,
				Raw:   raw,
				Value: raw,
			})
			textBuf.Reset()
		}
	}

	i := 0
	for i < n {
		if rs[i] != '\\' {
			textBuf.WriteRune(rs[i])
			i++
			continue
		}

		// Found backslash
		if i+1 >= n {
			// Lone trailing backslash
			textBuf.WriteRune('\\')
			i++
			continue
		}

		flushText()
		startIdx := i
		i++ // skip '\'
		next := rs[i]
		i++ // consume next rune

		switch next {
		case '*': // String reference: \*[name], \*(na, \*n
			name := parseEscapeName(rs, &i)
			tokens = append(tokens, InlineToken{
				Kind: InlineStringRef,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		case 'n': // Register reference: \n[name], \n(na, \nn
			name := parseEscapeName(rs, &i)
			tokens = append(tokens, InlineToken{
				Kind: InlineRegisterRef,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		case '[': // Composite/named character: \[name]
			name := readUntilBracket(rs, &i)
			tokens = append(tokens, InlineToken{
				Kind: InlineSpecialChar,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		case '(': // 2-character special glyph: \(bu
			start := i
			for count := 0; count < 2 && i < n; count++ {
				i++
			}
			name := string(rs[start:i])
			tokens = append(tokens, InlineToken{
				Kind: InlineSpecialChar,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		case 'f': // Font change: \f[name], \f(na, \fn
			name := parseEscapeName(rs, &i)
			tokens = append(tokens, InlineToken{
				Kind: InlineFontChange,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		case 's': // Size change: \s[size], \s(sz, \ss
			name := parseEscapeName(rs, &i)
			tokens = append(tokens, InlineToken{
				Kind: InlineSizeChange,
				Raw:  string(rs[startIdx:i]),
				Name: name,
			})

		default:
			// General single or common escape: \e, \\, \-, \~, \., \ , etc.
			tokens = append(tokens, InlineToken{
				Kind: InlineEscape,
				Raw:  string(rs[startIdx:i]),
				Name: string(next),
			})
		}
	}

	flushText()
	return tokens, nil
}

func parseEscapeName(rs []rune, i *int) string {
	n := len(rs)
	if *i >= n {
		return ""
	}

	r := rs[*i]
	if r == '[' {
		*i++
		return readUntilBracket(rs, i)
	} else if r == '(' {
		*i++
		start := *i
		for count := 0; count < 2 && *i < n; count++ {
			*i++
		}
		return string(rs[start:*i])
	}

	// Single character name
	*i++
	return string(r)
}

func readUntilBracket(rs []rune, i *int) string {
	n := len(rs)
	start := *i
	for *i < n && rs[*i] != ']' {
		*i++
	}
	name := string(rs[start:*i])
	if *i < n && rs[*i] == ']' {
		*i++ // skip closing bracket
	}
	return name
}

// DecodeDefaultSpecialChar converts standard groff special character names to Unicode.
func DecodeDefaultSpecialChar(name string) (string, bool) {
	switch name {
	case "bu":
		return "•", true
	case "em":
		return "—", true
	case "en":
		return "–", true
	case "hy":
		return "‐", true
	case "co":
		return "©", true
	case "rg":
		return "®", true
	case "tm":
		return "™", true
	case "lq":
		return "“", true
	case "rq":
		return "”", true
	case "oq":
		return "‘", true
	case "cq":
		return "’", true
	case "aq":
		return "'", true
	case "dq":
		return "\"", true
	case "ga":
		return "`", true
	case "aa":
		return "´", true
	case "ha":
		return "^", true
	case "ti":
		return "~", true
	default:
		return "", false
	}
}
