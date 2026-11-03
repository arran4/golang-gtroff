package groff

import (
	"errors"
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
