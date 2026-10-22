package groff

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrNoTokenMatch  = errors.New("no token match")
	ErrUnclosedQuote = errors.New("quote wasn't closed")
)

type Token interface {
	String() string
}

type TokenText string

func (t TokenText) String() string {
	return string(t)
}

var _ Token = (TokenText)("")

var wsRe = regexp.MustCompile(`\s+`)

type TokenComment string

func (t TokenComment) String() string {
	return `.\" ` + string(t)
}

var _ Token = (TokenComment)("")

func ParseComment(s string) (Token, error) {
	return TokenComment(s), nil
}

func ArgumentParser(l string) ([]string, int, error) {
	stack := []rune{}
	result := []string{}
	var argument strings.Builder
	quotedArg := false
	acomment := false
	i := 0
	rs := []rune(l)
	for ; i < len(rs) && !acomment; i++ {
		switch rs[i] {
		case ' ', '\t':
			if len(stack) == 0 {
				if argument.Len() != 0 || quotedArg {
					result = append(result, argument.String())
				}
				argument.Reset()
				quotedArg = false
				continue
			}
		case '\\':
			if i+1 >= len(rs) {
				break
			}
			i++
			if len(stack) == 0 && rs[i] == '"' {
				acomment = true
				i -= 2
				continue
			}
			if rs[i] != ' ' && rs[i] != '"' {
				argument.WriteRune('\\')
			}
		case '"':
			if argument.Len() > 0 && len(stack) == 0 {
				break
			}
			if len(stack) > 0 && stack[len(stack)-1] == '"' {
				// Check for doubled quote inside quotes: ""
				if i+1 < len(rs) && rs[i+1] == '"' {
					argument.WriteRune('"')
					i++
					continue
				}
				stack = stack[:len(stack)-1]
				if len(stack) == 0 {
					if argument.Len() != 0 || quotedArg {
						result = append(result, argument.String())
						argument.Reset()
						quotedArg = false
					}
				}
				continue
			} else if argument.Len() == 0 {
				stack = append(stack, '"')
				quotedArg = true
				continue
			}
		}
		argument.WriteRune(rs[i])
	}
	if len(stack) > 0 {
		return nil, i, fmt.Errorf("%w: stack %s", ErrUnclosedQuote, string(stack))
	}
	if argument.Len() > 0 || quotedArg {
		result = append(result, argument.String())
	}
	return result, i, nil
}
