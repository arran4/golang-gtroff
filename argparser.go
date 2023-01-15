package groff

import (
	"fmt"
	"strings"
)

type ArgStruct struct {
	Name     string
	Optional bool
	Position int
	Next     []*ArgStruct
}

var _ fmt.Stringer = (*ArgStruct)(nil)

func (a *ArgStruct) String() string {
	c := Args(a.Next).String()
	if a.Optional {
		return "[" + a.Name + c + "]"
	}
	return a.Name + c
}

type Args []*ArgStruct

func (a Args) String() string {
	c := ""
	for ni, next := range a {
		forceOptional := ni < len(a)-1
		ns := next.String()
		if forceOptional && !strings.HasPrefix(ns, "[") {
			c += " [" + ns + "]"
		} else {
			c += ns
		}
	}
	return c
}

var _ fmt.Stringer = (Args)(nil)

func ParseArgs(in string) (result Args, err error) {

}
