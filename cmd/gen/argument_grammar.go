package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/gen"
)

// ArgumentPolicy is source catalogue metadata. Omission means fixed arity from
// Args; aggregate and special request grammars always require explicit policy.
type ArgumentPolicy struct {
	Mode       string            `json:"mode"`
	Rest       string            `json:"rest,omitempty"`
	Optional   []bool            `json:"optional,omitempty"`
	Direction  string            `json:"direction,omitempty"`
	Minimum    *int              `json:"minimum,omitempty"`
	RepeatFrom int               `json:"repeat_from,omitempty"`
	Step       int               `json:"step,omitempty"`
	Flags      map[string]string `json:"flags,omitempty"`
	Numeric    []string          `json:"numeric,omitempty"`
}

type argumentPlan struct {
	Mode, Literal    string
	Minimum, Maximum int
}

func planArguments(entry *NormalizedEntry, words []*gen.Word) (argumentPlan, error) {
	p := entry.ArgumentGrammar
	if p == nil {
		p = &ArgumentPolicy{Mode: "fixed"}
	}
	mode := p.Mode
	if mode == "" {
		return argumentPlan{}, fmt.Errorf("argument grammar mode is required")
	}
	allowed := map[string]string{"fixed": "ArgumentsFixed", "joined": "ArgumentsJoined", "variadic": "ArgumentsVariadic", "raw": "ArgumentsRaw", "string": "ArgumentsString", "register": "ArgumentsRegister", "conditional": "ArgumentsConditional", "delimited": "ArgumentsDelimited"}
	constant, ok := allowed[mode]
	if !ok {
		return argumentPlan{}, fmt.Errorf("unknown argument grammar %q", mode)
	}
	n := len(words)
	optional := make([]bool, n)
	indices := map[string]int{}
	for i, word := range words {
		optional[i] = word.Optional
		indices[word.Id] = i
	}
	if p.Optional != nil {
		if len(p.Optional) != n {
			return argumentPlan{}, fmt.Errorf("argument grammar optional count differs from fields")
		}
		copy(optional, p.Optional)
	}
	minimum := 0
	for _, opt := range optional {
		if !opt {
			minimum++
		}
	}
	if p.Minimum != nil {
		if *p.Minimum < minimum {
			return argumentPlan{}, fmt.Errorf("minimum cannot omit required fields")
		}
		minimum = *p.Minimum
	}
	if minimum < 0 || p.RepeatFrom < 0 || p.Step < 0 {
		return argumentPlan{}, fmt.Errorf("negative argument grammar count")
	}
	reverse := false
	switch p.Direction {
	case "", "left":
	case "right":
		reverse = true
	default:
		return argumentPlan{}, fmt.Errorf("unknown binding direction %q", p.Direction)
	}
	rest := -1
	if p.Rest != "" {
		var ok bool
		rest, ok = indices[p.Rest]
		if !ok {
			return argumentPlan{}, fmt.Errorf("unknown aggregate field %q", p.Rest)
		}
	}
	switch mode {
	case "joined", "variadic", "raw", "string":
		if n == 0 || rest != n-1 || reverse {
			return argumentPlan{}, fmt.Errorf("aggregate must name the final forward field")
		}
	case "register":
		if n != 3 {
			return argumentPlan{}, fmt.Errorf("register grammar requires three fields")
		}
	case "conditional":
		if n != 2 {
			return argumentPlan{}, fmt.Errorf("conditional grammar requires two fields")
		}
	case "delimited":
		if n != 1 {
			return argumentPlan{}, fmt.Errorf("delimited grammar requires one field")
		}
	case "fixed":
		if rest != -1 || minimum > n {
			return argumentPlan{}, fmt.Errorf("invalid fixed grammar")
		}
	}
	if mode != "variadic" && (p.Step != 0 || p.RepeatFrom != 0) {
		return argumentPlan{}, fmt.Errorf("repetition controls require variadic grammar")
	}
	if p.RepeatFrom > n || mode == "variadic" && minimum < p.RepeatFrom {
		return argumentPlan{}, fmt.Errorf("invalid repeating prefix/minimum")
	}
	if mode != "fixed" && mode != "joined" && mode != "variadic" && reverse {
		return argumentPlan{}, fmt.Errorf("special grammar requires forward binding")
	}
	if mode != "joined" && mode != "variadic" && mode != "raw" && mode != "string" && p.Rest != "" {
		return argumentPlan{}, fmt.Errorf("unexpected aggregate field")
	}
	if mode == "register" && (optional[0] || optional[1] || !optional[2] || minimum != 2) {
		return argumentPlan{}, fmt.Errorf("register grammar requires name, value and optional increment")
	}
	if mode == "conditional" && (optional[0] || minimum != 1+btoi(!optional[1])) {
		return argumentPlan{}, fmt.Errorf("conditional grammar requires predicate and declared body")
	}
	if mode == "delimited" && (optional[0] || minimum != 1) {
		return argumentPlan{}, fmt.Errorf("delimited grammar requires one raw field")
	}
	if len(p.Numeric) > 0 && (mode != "fixed" || reverse) {
		return argumentPlan{}, fmt.Errorf("numeric fields require a forward fixed grammar")
	}
	if mode == "raw" || mode == "string" {
		required := 0
		for _, opt := range optional {
			if !opt {
				required++
			}
		}
		if minimum != required {
			return argumentPlan{}, fmt.Errorf("raw minimum must match required fields")
		}
		for _, opt := range optional[:rest] {
			if opt {
				return argumentPlan{}, fmt.Errorf("raw prefixes must be required")
			}
		}
	}
	if mode != "fixed" && len(p.Flags) > 0 {
		return argumentPlan{}, fmt.Errorf("literal flags require fixed grammar")
	}
	var opts []string
	for _, opt := range optional {
		opts = append(opts, strconv.FormatBool(opt))
	}
	literal := fmt.Sprintf("ArgumentGrammar{Mode:%s, Optional:[]bool{%s}, Reverse:%t, Rest:%d, Minimum:%d, RepeatFrom:%d, Step:%d", constant, strings.Join(opts, ","), reverse, rest, minimum, p.RepeatFrom, p.Step)
	var flagNames []string
	for name := range p.Flags {
		flagNames = append(flagNames, name)
	}
	sort.Strings(flagNames)
	if len(flagNames) > 0 {
		literal += ",Flags:map[int]string{"
		for _, name := range flagNames {
			i, ok := indices[name]
			if !ok || !optional[i] || p.Flags[name] == "" {
				return argumentPlan{}, fmt.Errorf("invalid optional flag %q", name)
			}
			literal += fmt.Sprintf("%d:%q,", i, p.Flags[name])
		}
		literal += "}"
	}
	if len(p.Numeric) > 0 {
		numeric := make([]bool, n)
		optionalSeen := false
		for _, opt := range optional {
			if opt {
				optionalSeen = true
			} else if optionalSeen {
				return argumentPlan{}, fmt.Errorf("numeric grammar requires trailing optional fields")
			}
		}
		for _, name := range p.Numeric {
			i, ok := indices[name]
			if !ok || numeric[i] {
				return argumentPlan{}, fmt.Errorf("invalid numeric field %q", name)
			}
			numeric[i] = true
		}
		literal += ",Numeric:[]bool{"
		for _, isNumeric := range numeric {
			literal += strconv.FormatBool(isNumeric) + ","
		}
		literal += "}"
	}
	literal += "}"
	maximum := -1
	if mode == "fixed" || mode == "register" {
		maximum = n
	}
	return argumentPlan{Mode: mode, Literal: literal, Minimum: minimum, Maximum: maximum}, nil
}
func entryArgumentWords(entry *NormalizedEntry) ([]*gen.Word, error) {
	signature := entry.ArgsRaw
	if override, ok := consolidatedSignatures[entry.Token]; ok {
		signature = override
	}
	args, err := gen.ParseArgs(signature)
	if err != nil {
		return nil, err
	}
	return args.Words, nil
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}
