package gen

import (
	"fmt"
)

type Word struct {
	Word     string
	Optional bool
	Id       string
}

type ArgStruct struct {
	Words          []*Word
	OptionalRanges [][2]int
}

func (ar *ArgStruct) HasOptional() bool {
	return len(ar.OptionalRanges) > 0
}

func (ar *ArgStruct) AllOptional() bool {
	return !ar.AllRequired()
}

func (ar *ArgStruct) AllRequired() bool {
	for _, w := range ar.Words {
		if w.Optional {
			return false
		}
	}
	return true
}

func (ar *ArgStruct) Forwards() bool {
	if len(ar.Words) == 0 {
		return false
	}
	return len(ar.Closings(len(ar.Words))) > len(ar.Openings(0)) || len(ar.Words) == len(ar.OptionalRanges)
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
	result, err = parseArgRun(in)
	if err != nil {
		return
	}
	m := map[string][]int{}
	for i, w := range result.Words {
		word := w.Word // TODO make safe
		countPos := append(m[w.Word], i)
		m[word] = countPos
		switch len(countPos) {
		case 1:
			result.Words[countPos[0]].Id = word
		case 2:
			result.Words[countPos[0]].Id += "0"
			fallthrough
		default:
			result.Words[countPos[len(countPos)-1]].Id = fmt.Sprintf("%s%d", word, len(countPos)-1)
		}
	}
	return
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
