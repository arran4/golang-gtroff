package groff

import (
	"fmt"
	"strings"
)

// Built-in registrations and preprocessor delimiter pairs are generated from
// catalogue Block declarations. Implementations remain handwritten here.
type builtinBlockHandler struct {
	command, kind string
	ends          []string
}

func (h builtinBlockHandler) CanHandle(cmd, pkg string) bool {
	return normalizedBlockCommand(cmd) == h.command
}
func (h builtinBlockHandler) Begin(header BlockHeader) (BlockRule, error) {
	rule := BlockRule{Ends: append([]string(nil), h.ends...)}
	switch h.kind {
	case "definition", "append", "ignore":
		_, end, err := blockHeaderNames(h.kind, header.Args)
		if err != nil {
			return BlockRule{}, err
		}
		rule.Ends = []string{".."}
		if end != "" {
			rule.Ends[0] = "." + end
		}
	default:
		rule.Nested = generatedPreprocessorPairs()
	}
	return rule, nil
}
func blockHeaderNames(kind, args string) (string, string, error) {
	optional := []bool{false, true}
	minimum := 1
	if kind == "ignore" {
		optional = []bool{true}
		minimum = 0
	}
	fields, err := BindArguments(args, ArgumentGrammar{Mode: ArgumentsFixed, Optional: optional, Minimum: minimum})
	if err != nil {
		return "", "", err
	}
	name, end := "", ""
	if kind == "ignore" {
		end = fields[0]
	} else {
		name, end = fields[0], fields[1]
		if name == "" {
			return "", "", fmt.Errorf("macro name must not be empty")
		}
	}
	for _, value := range []string{name, end} {
		if strings.ContainsAny(value, " \t\r\n\x00") {
			return "", "", fmt.Errorf("block name contains whitespace or NUL")
		}
	}
	return name, end, nil
}
func (h builtinBlockHandler) ParseBlock(source BlockSource) (Token, error) {
	body := make([]string, 0, len(source.Body))
	for _, line := range source.Body {
		body = append(body, line.Text())
	}
	switch h.kind {
	case "definition", "append", "ignore":
		name, end, err := blockHeaderNames(h.kind, source.Header.Args)
		if err != nil {
			return nil, err
		}
		switch h.kind {
		case "definition":
			return &TokenMacroDefinition{Name: name, End: end, Body: body}, nil
		case "append":
			return &TokenMacroAppend{Name: name, End: end, Body: body}, nil
		default:
			return &TokenIgnoredBlock{End: end, Body: body}, nil
		}
	case "table":
		return &TokenTableBlock{Options: source.Header.Args, Lines: body}, nil
	case "equation":
		return &TokenEquationBlock{Args: source.Header.Args, Lines: body}, nil
	case "picture":
		return &TokenPictureBlock{Args: source.Header.Args, Lines: body}, nil
	case "reference":
		return &TokenReferenceBlock{Lines: body}, nil
	}
	return nil, fmt.Errorf("%w: unknown built-in block kind %q", ErrInvalidHandler, h.kind)
}

var _ BlockHandler = builtinBlockHandler{}
