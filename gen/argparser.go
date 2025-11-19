package gen

import (
	"fmt"
	"strings"
)

type Word struct {
	Word     string
	Optional bool
	Id       string
}

func (w *Word) Name() string {
	return w.Word
}

func (w *Word) Type() string {
	return "string"
}

func (w *Word) Comment() string {
	return ""
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

var (
	saferer = strings.NewReplacer(
		"'", "",
		"…", "Ellipse",
		"-", "",
	)
)

func ParseArgs(in string) (result *ArgStruct, err error) {
	result, err = parseArgRun(in)
	if err != nil {
		return
	}
	m := map[string][]int{}
	for i, w := range result.Words {
		word := w.Word
		if len(word) >= 2 && ((word[0] == '\'' && word[len(word)-1] == '\'') || (word[0] == '"' && word[len(word)-1] == '"')) {
			word = word[1 : len(word)-1]
		}
		word = saferer.Replace(word)
		if len(word) > 0 && word[0] >= '0' && word[0] <= '9' {
			word = "Arg" + word
		}
		// Ensure valid identifier characters (simple approach)
		cleanWord := ""
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				cleanWord += string(r)
			}
		}
		word = cleanWord
		if word == "" {
			word = fmt.Sprintf("Arg%d", i+1)
		}

		countPos := append(m[w.Word], i)
		m[w.Word] = countPos // Use original word for map key to track duplicates correctly based on input
		// But wait, if we have multiple args that map to the same sanitized name, we need to handle that.
		// The original code used m[w.Word] which seems to track duplicates of the *original* word.
		// But if 'a' and "a" both map to a, we have a collision in generated code.
		// We should track usage of the *sanitized* word.
	}

	// Re-implementing the loop to track usage of sanitized names
	usedIds := map[string]bool{}
	for i, w := range result.Words {
		word := w.Word
		if len(word) >= 2 && ((word[0] == '\'' && word[len(word)-1] == '\'') || (word[0] == '"' && word[len(word)-1] == '"')) {
			word = word[1 : len(word)-1]
		}
		word = saferer.Replace(word)
		if len(word) > 0 && word[0] >= '0' && word[0] <= '9' {
			word = "Arg" + word
		}
		cleanWord := ""
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				cleanWord += string(r)
			}
		}
		word = cleanWord
		if word == "" {
			word = fmt.Sprintf("Arg%d", i+1)
		}

		// Capitalize first letter
		if len(word) > 0 {
			word = strings.ToUpper(word[:1]) + word[1:]
		}

		// Ensure uniqueness
		base := word
		count := 1
		for usedIds[word] {
			count++
			word = fmt.Sprintf("%s%d", base, count)
		}
		usedIds[word] = true
		result.Words[i].Id = word
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
