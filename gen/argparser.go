package gen

import (
	"fmt"
)

type Word struct {
	Word     string
	Optional bool
}

type ArgStruct struct {
	Words          []*Word
	OptionalRanges [][2]int
}

func (ar *ArgStruct) Openings(n int) (r [][2]int) {
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[0] == n {
			r = append(r, e)
		}
	}
	return
}

func (ar *ArgStruct) Closings(n int) (r [][2]int) {
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[1] == n {
			r = append(r, e)
		}
	}
	return
}

func ParseArgs(in string) (result *ArgStruct, err error) {
	return parseArgRun(in)
}

func parseArgRun(in string) (*ArgStruct, error) {
	ar := &ArgStruct{}
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
				ar.Words = append(ar.Words, &Word{
					Word:     string(rin[wordBegin:wordEnd]),
					Optional: len(openStack) > 0,
				})
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
