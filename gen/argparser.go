package gen

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

type argRun struct {
	Words          []string
	OptionalRanges [][2]int
}

func ParseArgs(in string) (result Args, err error) {
	result = make(Args, 0)
	_, err = parseArgRun(in)
	if err != nil {
		return nil, err
	}
	return
}

func parseArgRun(in string) (*argRun, error) {
	ar := &argRun{}
	closeStack := []int{}
	openStack := []int{}
	wordBegin := 0
	wordEnd := 0
	rin := ([]rune)(in + " ")
	for rp, r := range rin {
		switch r {
		case '[':
			if len(closeStack) != 0 {
				closeStack = nil
			}
			if rp != wordBegin {
				break
			}
			openStack = append(openStack, len(ar.Words))
			wordBegin = rp + 1
			wordEnd = rp + 1
		case ']':
			closeStack = append(closeStack, rp)
		case ' ':
			if wordBegin != wordEnd {
				ar.Words = append(ar.Words, string(rin[wordBegin:wordEnd]))
				for len(closeStack) > len(openStack) {
					return nil, fmt.Errorf("more closes than opens")
				}
				for i := range closeStack {
					open := openStack[len(openStack)-len(closeStack)+i]
					ar.OptionalRanges = append(ar.OptionalRanges, [2]int{open, len(ar.Words)})
				}
				openStack = openStack[:len(openStack)-len(closeStack)]
				closeStack = nil
			}
			wordBegin = rp + 1
			wordEnd = rp + 1
		default:
			wordEnd = rp + 1
			if len(closeStack) != 0 {
				closeStack = nil
			}
		}
	}
	if len(closeStack) > 0 {
		return nil, fmt.Errorf("close stack has %d elements", len(closeStack))
	}
	return ar, nil
}
