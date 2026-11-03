package groff

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type evaluationFrame struct {
	name string
	args []string
}

func (m *evaluation) generatedCollision(tok Token) (bool, error) {
	if _, unknown := tok.(*TokenUnknownControl); unknown {
		return false, nil
	}
	node, ok := tok.(Node)
	if !ok {
		return false, nil
	}
	switch node.TokenKind() {
	case KindText, KindComment, KindControl:
		return false, nil
	}
	name := node.TokenName()
	if err := m.charge(len(name)); err != nil {
		return false, err
	}
	if name == "" {
		return false, nil
	}
	_, _, _, defined, err := m.definition(strings.TrimLeft(name, ".'"))
	return defined, err
}

func (m *evaluation) invokeMacro(name string, call *TokenUnknownControl, body []string) error {
	return m.nested("macro "+name, func() error {
		if len(call.RawArgs) > m.lineLimit {
			return &EvaluationLimitError{Kind: "line byte", Limit: m.lineLimit}
		}
		if err := m.charge(len(call.RawArgs)); err != nil {
			return err
		}
		args, _, err := ArgumentParser(call.RawArgs)
		if err != nil {
			return fmt.Errorf("parse macro arguments: %w", err)
		}
		bytes := 0
		for i, arg := range args {
			expanded, err := m.expand(arg)
			if err != nil {
				return err
			}
			if len(expanded) > m.outputLimit-bytes {
				return &EvaluationLimitError{Kind: "output byte", Limit: m.outputLimit}
			}
			bytes += len(expanded)
			args[i] = expanded
		}
		m.frames = append(m.frames, evaluationFrame{name: name, args: args})
		defer func() { m.frames = m.frames[:len(m.frames)-1] }()
		for i, line := range body {
			if err := m.nested(fmt.Sprintf("body line %d", i+1), func() error { return m.line(line) }); err != nil {
				return err
			}
		}
		return nil
	})
}
func (m *evaluation) parameter(name string, allowZero bool) (string, error) {
	if err := m.charge(len(name)); err != nil {
		return "", err
	}
	index, err := strconv.Atoi(name)
	if err != nil || index < 0 {
		return "", fmt.Errorf("unsupported macro parameter %q", name)
	}
	frame := m.frames[len(m.frames)-1]
	if index == 0 {
		if !allowZero {
			return "", fmt.Errorf("macro parameter index must be positive")
		}
		return frame.name, nil
	}
	if index > len(frame.args) {
		return "", nil
	}
	return frame.args[index-1], nil
}

// Generated man/ms Node metadata selects formatting tokens. Clone their plain
// string fields instead of serializing/reparsing them, which would lose quoting
// or change token types. Never reflect over arbitrary external token graphs.
func (m *evaluation) formattingArguments(tok Token, node Node) (Token, bool, error) {
	if node.TokenKind() != KindMacro || node.PackageName() != "man" && node.PackageName() != "ms" {
		return tok, false, nil
	}
	value := reflect.ValueOf(tok)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct || value.Elem().Type().PkgPath() != reflect.TypeOf(TokenRequestNr{}).PkgPath() {
		return tok, false, nil
	}
	value = value.Elem()
	// Validate the complete shape before interpreting any field.
	for i := 0; i < value.NumField(); i++ {
		if value.Field(i).Kind() != reflect.String || !value.Type().Field(i).IsExported() {
			return tok, false, nil
		}
	}
	copy := reflect.New(value.Type())
	copy.Elem().Set(value)
	changed := false
	for i := 0; i < value.NumField(); i++ {
		original := value.Field(i).String()
		expanded, err := m.expand(original)
		if err != nil {
			return nil, false, err
		}
		if expanded != original {
			copy.Elem().Field(i).SetString(expanded)
			changed = true
		}
	}
	if !changed {
		return tok, false, nil
	}
	return copy.Interface().(Token), true, nil
}

func (m *evaluation) copiedReference(text string, start int) bool {
	return len(m.frames) > 0 && len(text)-start >= 3 && text[start] == '\\' && text[start+1] == '\\' && strings.ContainsRune("n*$", rune(text[start+2]))
}
