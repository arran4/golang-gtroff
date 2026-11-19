package groff

import (
	"fmt"
	"strings"
)

func (m *evaluation) resolveDefinition(name string) (string, error) {
	seen := make(map[string]bool)
	for depth := 0; ; depth++ {
		if err := m.charge(len(name)); err != nil {
			return "", err
		}
		if depth >= m.depthLimit {
			return "", &EvaluationLimitError{Kind: "recursion depth", Limit: m.depthLimit}
		}
		if seen[name] {
			return "", fmt.Errorf("definition alias cycle at %q", name)
		}
		seen[name] = true
		alias, ok, err := m.load(EvaluationAlias, name)
		if err != nil {
			return "", err
		}
		if !ok {
			return name, nil
		}
		if alias.Text == "" {
			return "", fmt.Errorf("empty definition alias target")
		}
		name = alias.Text
	}
}
func (m *evaluation) definition(name string) (string, EvaluationStateKind, EvaluationValue, bool, error) {
	root, err := m.resolveDefinition(name)
	if err != nil {
		return "", "", EvaluationValue{}, false, err
	}
	value, ok, err := m.load(EvaluationString, root)
	if err != nil {
		return "", "", value, false, err
	}
	if ok {
		return root, EvaluationString, value, true, nil
	}
	value, ok, err = m.load(EvaluationMacro, root)
	return root, EvaluationMacro, value, ok, err
}
func (m *evaluation) validateDefinitions() error {
	var visitErr error
	err := m.store.Walk(func(kind EvaluationStateKind, name string, value EvaluationValue) bool {
		if visitErr != nil {
			return false
		}
		if visitErr = m.tick(); visitErr != nil {
			return false
		}
		switch kind {
		case EvaluationAlias:
			_, _, _, ok, err := m.definition(name)
			if err != nil {
				visitErr = err
				return false
			}
			if !ok {
				visitErr = fmt.Errorf("undefined alias target for %q", name)
				return false
			}
			_, chained, err := m.load(EvaluationAlias, value.Text)
			if err != nil {
				visitErr = err
				return false
			}
			if chained {
				visitErr = fmt.Errorf("stored aliases must target definitions directly")
				return false
			}
			for _, other := range []EvaluationStateKind{EvaluationString, EvaluationMacro} {
				_, exists, err := m.load(other, name)
				if err != nil {
					visitErr = err
					return false
				}
				if exists {
					visitErr = fmt.Errorf("alias %q also has a definition", name)
					return false
				}
			}
		case EvaluationString:
			_, exists, err := m.load(EvaluationMacro, name)
			if err != nil {
				visitErr = err
				return false
			}
			if exists {
				visitErr = fmt.Errorf("both string and macro stored for %q", name)
				return false
			}
		}
		return true
	})
	if visitErr != nil {
		return visitErr
	}
	return err
}
func (m *evaluation) setDefinition(kind EvaluationStateKind, name string, value EvaluationValue) error {
	if name == "" {
		return fmt.Errorf("definition requires a name")
	}
	root, err := m.resolveDefinition(name)
	if err != nil {
		return err
	}
	if err := m.save(kind, root, value); err != nil {
		return err
	}
	other := EvaluationMacro
	if kind == EvaluationMacro {
		other = EvaluationString
	}
	return m.erase(other, root)
}
func (m *evaluation) appendString(name, text string) error {
	if name == "" {
		return fmt.Errorf(".as requires a name")
	}
	root, kind, old, exists, err := m.definition(name)
	if err != nil {
		return err
	}
	if exists && kind != EvaluationString {
		return fmt.Errorf(".as on macro %q is unsupported", name)
	}
	if len(text) > m.stateLimit-len(old.Text)-32 {
		return &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
	}
	if err := m.state(string(EvaluationString), root, len(old.Text)+len(text)+32); err != nil {
		return err
	}
	if err := m.charge(len(old.Text) + len(text)); err != nil {
		return err
	}
	return m.save(EvaluationString, root, EvaluationValue{Text: old.Text + text})
}
func (m *evaluation) appendMacro(name string, body []string) error {
	if name == "" {
		return fmt.Errorf(".am requires a name")
	}
	root, kind, old, exists, err := m.definition(name)
	if err != nil {
		return err
	}
	if exists && kind != EvaluationMacro {
		return fmt.Errorf(".am on string %q is unsupported", name)
	}
	oldSize, err := m.bodySize(old.Body)
	if err != nil {
		return err
	}
	newSize, err := m.bodySize(body)
	if err != nil {
		return err
	}
	if newSize-32 > m.stateLimit-oldSize {
		return &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
	}
	if err := m.state(string(EvaluationMacro), root, oldSize+newSize-32); err != nil {
		return err
	}
	if err := m.charge(16 * (len(old.Body) + len(body))); err != nil {
		return err
	}
	combined := make([]string, 0, len(old.Body)+len(body))
	combined = append(combined, old.Body...)
	combined = append(combined, body...)
	return m.save(EvaluationMacro, root, EvaluationValue{Body: combined})
}
func (m *evaluation) aliasDefinition(tok *TokenRequestAls) error {
	root, _, _, exists, err := m.definition(tok.Old)
	if err != nil {
		return err
	}
	if !exists {
		return m.preserve(tok, "alias target is undefined")
	}
	if tok.New == "" {
		return fmt.Errorf("empty alias name")
	}
	newRoot, err := m.resolveDefinition(tok.New)
	if err != nil {
		return err
	}
	if newRoot == root {
		return nil
	}
	if err := m.removeDefinition(tok.New); err != nil {
		return err
	}
	return m.save(EvaluationAlias, tok.New, EvaluationValue{Text: root})
}
func (m *evaluation) removeDefinition(name string) error {
	_, isAlias, err := m.load(EvaluationAlias, name)
	if err != nil {
		return err
	}
	if isAlias {
		return m.erase(EvaluationAlias, name)
	}
	_, kind, value, exists, err := m.definition(name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	// Alias targets are always live canonical names. Promote the least alias
	// when removing its root, preserving the shared object under remaining names.
	var aliases []string
	var visitErr error
	err = m.store.Walk(func(k EvaluationStateKind, n string, v EvaluationValue) bool {
		if visitErr != nil {
			return false
		}
		if visitErr = m.charge(len(n) + len(v.Text)); visitErr != nil {
			return false
		}
		if k == EvaluationAlias && v.Text == name {
			aliases = append(aliases, n)
		}
		return true
	})
	if visitErr != nil {
		return visitErr
	}
	if err != nil {
		return err
	}
	if len(aliases) == 0 {
		return m.erase(kind, name)
	}
	promoted := aliases[0]
	for _, alias := range aliases[1:] {
		if alias < promoted {
			promoted = alias
		}
	}
	if err := m.save(kind, promoted, value); err != nil {
		return err
	}
	if err := m.erase(EvaluationAlias, promoted); err != nil {
		return err
	}
	for _, alias := range aliases {
		if alias != promoted {
			if err := m.save(EvaluationAlias, alias, EvaluationValue{Text: promoted}); err != nil {
				return err
			}
		}
	}
	return m.erase(kind, name)
}
func (m *evaluation) names(text string, apply func(string) error) error {
	if err := m.charge(len(text)); err != nil {
		return err
	}
	for text != "" {
		text = strings.TrimLeft(text, " \t\r\n")
		if text == "" {
			break
		}
		end := strings.IndexAny(text, " \t\r\n")
		if end < 0 {
			end = len(text)
		}
		if err := m.tick(); err != nil {
			return err
		}
		if err := apply(text[:end]); err != nil {
			return err
		}
		text = text[end:]
	}
	return nil
}
