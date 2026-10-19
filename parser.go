package groff

import (
	"errors"
	"fmt"
	"regexp"
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
	a := ""
	acomment := false
	i := 0
	rs := []rune(l)
	for ; i < len(rs) && !acomment; i++ {
		switch rs[i] {
		case ' ':
			if len(stack) == 0 {
				if len(a) != 0 {
					result = append(result, a)
				}
				a = ""
				continue
			}
		case '\\':
			i++
			if len(stack) == 0 && rs[i] == '"' {
				acomment = true
				i -= 2
				continue
			}
			if rs[i] != ' ' && rs[i] != '"' {
				a += "\\"
			}
		case '"':
			if len(a) > 0 && len(stack) == 0 {
				break
			}
			if len(stack) > 0 && stack[len(stack)-1] == '"' {
				stack = stack[:len(stack)-1]
				if len(stack) == 0 && len(a) != 0 {
					result = append(result, a)
					a = ""
				}
				continue
			} else if len(a) == 0 {
				stack = append(stack, '"')
				continue
			}
		}
		a += string(rs[i])
	}
	if len(stack) > 0 {
		return nil, i, fmt.Errorf("%w: stack %s", ErrUnclosedQuote, string(stack))
	}
	if len(a) > 0 {
		result = append(result, a)
	}
	return result, i, nil
}
