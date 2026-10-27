package groff

import (
	"fmt"
	"strings"
)

// ArgumentLexeme represents a parsed argument with its position and raw lexeme.
type ArgumentLexeme struct {
	Raw    string
	Value  string
	Start  int
	End    int
	Quoted bool
}

// LexArguments tokenizes roff arguments with full Unicode and quote awareness.
func LexArguments(l string) ([]ArgumentLexeme, int, error) {
	var lexemes []ArgumentLexeme
	rs := []rune(l)
	n := len(rs)

	i := 0
	for i < n {
		// Skip unquoted leading whitespace
		for i < n && (rs[i] == ' ' || rs[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		// Check for unquoted comment \"
		if rs[i] == '\\' && i+1 < n && rs[i+1] == '"' {
			// Comment starts here; consumes remainder of line
			return lexemes, i, nil
		}

		argStart := i
		var valBuilder strings.Builder
		var rawBuilder strings.Builder
		isQuoted := false

		if rs[i] == '"' {
			isQuoted = true
			rawBuilder.WriteRune(rs[i])
			i++ // skip opening quote

			closed := false
			for i < n {
				r := rs[i]
				rawBuilder.WriteRune(r)

				if r == '\\' {
					if i+1 < n {
						next := rs[i+1]
						if next == '"' {
							// Literal escaped quote inside quotes: \"
							rawBuilder.WriteRune(next)
							valBuilder.WriteRune('"')
							i += 2
							continue
						} else if next == ' ' {
							// Escaped space inside quotes
							rawBuilder.WriteRune(next)
							valBuilder.WriteRune(' ')
							i += 2
							continue
						} else {
							// Keep backslash and next char
							rawBuilder.WriteRune(next)
							valBuilder.WriteRune('\\')
							valBuilder.WriteRune(next)
							i += 2
							continue
						}
					} else {
						// Lone trailing backslash
						valBuilder.WriteRune('\\')
						i++
						continue
					}
				}

				if r == '"' {
					// Check for doubled quote: "" inside quotes is literal quote
					if i+1 < n && rs[i+1] == '"' {
						rawBuilder.WriteRune(rs[i+1])
						valBuilder.WriteRune('"')
						i += 2
						continue
					}
					// Closing quote
					closed = true
					i++
					break
				}

				valBuilder.WriteRune(r)
				i++
			}

			if !closed {
				return nil, i, fmt.Errorf("%w: unterminated quote starting at index %d", ErrUnclosedQuote, argStart)
			}
		} else {
			// Unquoted argument
			for i < n {
				r := rs[i]
				if r == ' ' || r == '\t' {
					break
				}

				// Check for comment
				if r == '\\' && i+1 < n && rs[i+1] == '"' {
					// Comment starts
					break
				}

				if r == '\\' {
					rawBuilder.WriteRune(r)
					if i+1 < n {
						next := rs[i+1]
						rawBuilder.WriteRune(next)
						if next == ' ' {
							// Escaped space
							valBuilder.WriteRune(' ')
						} else if next == '"' {
							valBuilder.WriteRune('"')
						} else {
							valBuilder.WriteRune('\\')
							valBuilder.WriteRune(next)
						}
						i += 2
						continue
					} else {
						valBuilder.WriteRune('\\')
						i++
						continue
					}
				}

				rawBuilder.WriteRune(r)
				valBuilder.WriteRune(r)
				i++
			}
		}

		lexemes = append(lexemes, ArgumentLexeme{
			Raw:    rawBuilder.String(),
			Value:  valBuilder.String(),
			Start:  argStart,
			End:    i,
			Quoted: isQuoted,
		})
	}

	return lexemes, i, nil
}
