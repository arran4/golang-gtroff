package gen

import (
	"fmt"
	"sort"
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
	if len(c) > 0 {
		c = " " + c
	}
	if a.Optional {
		return "[" + a.Name + c + "]"
	}
	return a.Name + c
}

type Args []*ArgStruct

func (a Args) String() string {
	c := ""
	for ni, next := range a {
		if ni > 0 {
			c += " "
		}
		forceOptional := ni < len(a)-2
		ns := next.String()
		if forceOptional {
			c += "[" + ns + "]"
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

func (ar *argRun) Openings(n int) (r [][2]int) {
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[0] == n {
			r = append(r, e)
		}
	}
	return
}

func (ar *argRun) Closings(n int) (r [][2]int) {
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[1] == n {
			r = append(r, e)
		}
	}
	return
}

func ParseArgs(in string) (result Args, err error) {
	result = make(Args, 0)
	ar, err := parseArgRun(in)
	if err != nil {
		return nil, err
	}
	i := 0
	result, i = ar.parseArgs(i, len(ar.Words), 0)
	if i != len(ar.Words) {
		return nil, fmt.Errorf("end difference %d != %d", i, len(ar.Words))
	}
	return
}

func (ar *argRun) parseArgs(i int, end int, depth int) (Args, int) {
	var result []*ArgStruct
	for loops := 0; i < end; loops++ {
		as := &ArgStruct{
			Name:     ar.Words[i],
			Optional: ar.OptionalStart(i) > 0,
			Position: depth + loops,
		}
		result = append(result, as)
		i++
		if v := ar.OptionalStart(i); v >= i {
			as.Next, i = ar.parseArgs(i, v, depth+loops+1)
		} else {
			as.Next, i = ar.parseArgs(i, end, depth+loops+1)
			break
		}
	}
	return result, i
}

func (ar *argRun) OptionalStart(n int) int {
	var result []int
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[0] == n {
			result = append(result, e[1])
		}
	}
	if len(result) == 0 {
		return -1
	}
	sort.Ints(result)
	return result[len(result)-1]
}

func (ar *argRun) OptionalStartMany(n int) int {
	var result []int
	for i := range ar.OptionalRanges {
		e := ar.OptionalRanges[i]
		if e[0] == n && e[1] > n+1 {
			result = append(result, e[1])
		}
	}
	if len(result) == 0 {
		return -1
	}
	sort.Ints(result)
	return result[len(result)-1]
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
