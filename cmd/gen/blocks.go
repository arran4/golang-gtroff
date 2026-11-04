package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type BlockPolicy struct {
	Kind string   `json:"kind"`
	Ends []string `json:"ends,omitempty"`
}
type BlockDefinition struct {
	Command, Kind string
	Ends          []string
	Preprocessor  bool
}

var blockKinds = map[string]string{
	".de": "definition", ".am": "append", ".ig": "ignore", ".TS": "table", ".EQ": "equation", ".PS": "picture", ".[": "reference",
	".de1": "definition_variant", ".dei": "definition_variant", ".dei1": "definition_variant",
	".am1": "append_variant", ".ami": "append_variant", ".ami1": "append_variant",
}

func validateBlockPolicy(e *NormalizedEntry) error {
	if e.Block == nil {
		if _, known := blockKinds[e.Token]; known {
			return fmt.Errorf("missing block declaration for %s", e.Token)
		}
		return nil
	}
	p := e.Block
	if expected, known := blockKinds[e.Token]; !known || p.Kind != expected || (e.Kind != KindMacro && e.Kind != KindRequest) {
		return fmt.Errorf("unregistered block implementation %q for %s", p.Kind, e.Token)
	}

	dynamic := p.Kind == "definition" || p.Kind == "append" || p.Kind == "ignore" || p.Kind == "definition_variant" || p.Kind == "append_variant"
	if dynamic && len(p.Ends) != 0 || !dynamic && (len(p.Ends) == 0 || len(p.Ends) > 128) {
		return fmt.Errorf("invalid block closing declaration")
	}
	seen := map[string]bool{}
	for _, end := range p.Ends {
		if len(end) < 2 || end[0] != '.' || !utf8.ValidString(end) || strings.ContainsAny(end, " \t\r\n\x00") || seen[end] {
			return fmt.Errorf("invalid/duplicate block delimiter %q", end)
		}
		seen[end] = true
	}
	return nil
}
func prepareBlockDefinitions(cat *Catalog) []BlockDefinition {
	defs := map[string]BlockDefinition{}
	for _, e := range cat.Entries {
		if e == nil || e.Block == nil {
			continue
		}
		p := e.Block
		defs[e.Token] = BlockDefinition{Command: e.Token, Kind: p.Kind, Ends: append([]string(nil), p.Ends...), Preprocessor: p.Kind != "definition" && p.Kind != "append" && p.Kind != "ignore" && p.Kind != "definition_variant" && p.Kind != "append_variant"}
	}
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []BlockDefinition
	for _, name := range names {
		out = append(out, defs[name])
	}
	return out
}
