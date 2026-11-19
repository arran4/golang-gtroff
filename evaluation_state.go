package groff

import "fmt"

// EvaluationStateKind distinguishes the namespaces of evaluator bindings.
type EvaluationStateKind string

const (
	EvaluationRegister  EvaluationStateKind = "register"
	EvaluationIncrement EvaluationStateKind = "increment"
	EvaluationString    EvaluationStateKind = "string"
	EvaluationMacro     EvaluationStateKind = "macro"
	EvaluationAlias     EvaluationStateKind = "alias"
)

// EvaluationValue holds one binding. Registers/increments use Number;
// strings/aliases use Text; macros use Body. Other fields must be empty.
type EvaluationValue struct {
	Number int
	Text   string
	Body   []string
}

// EvaluationState is the storage boundary for opt-in evaluation. Implementations
// must return consistent bindings, enumerate every binding once, and stop Walk
// when yield returns false. Store must copy Body; Load/Walk bodies are read-only.
// Missing bindings are not errors by themselves, including for Delete.
// Methods are trusted Go code, must not modify unrelated bindings, and are not
// interruptible by the evaluator. No methods are called by structural parsing.
// State errors preserve their identity through EvaluationError.
type EvaluationState interface {
	Load(kind EvaluationStateKind, name string) (EvaluationValue, bool, error)
	Store(kind EvaluationStateKind, name string, value EvaluationValue) error
	Delete(kind EvaluationStateKind, name string) error
	Walk(yield func(EvaluationStateKind, string, EvaluationValue) bool) error
}

var _ EvaluationState = (*Environment)(nil)

// Load reads one raw binding in this environment's maps. Aliases are resolved
// by the evaluator, not by this storage interface. State is not delegated here.
func (env *Environment) Load(kind EvaluationStateKind, name string) (EvaluationValue, bool, error) {
	switch kind {
	case EvaluationRegister:
		v, ok := env.Registers[name]
		return EvaluationValue{Number: v}, ok, nil
	case EvaluationIncrement:
		v, ok := env.AutoIncrs[name]
		return EvaluationValue{Number: v}, ok, nil
	case EvaluationString:
		v, ok := env.Strings[name]
		return EvaluationValue{Text: v}, ok, nil
	case EvaluationMacro:
		v, ok := env.Macros[name]
		return EvaluationValue{Body: v}, ok, nil
	case EvaluationAlias:
		v, ok := env.Aliases[name]
		return EvaluationValue{Text: v}, ok, nil
	}
	return EvaluationValue{}, false, fmt.Errorf("unknown evaluation state kind %q", kind)
}

// Store writes a raw binding and copies macro body slices. Like the existing
// setters, it does not enforce budgets when called outside evaluation.
func (env *Environment) Store(kind EvaluationStateKind, name string, value EvaluationValue) error {
	env.init()
	if err := evaluationValueFields(kind, value); err != nil {
		return err
	}
	switch kind {
	case EvaluationRegister:
		env.Registers[name] = value.Number
	case EvaluationIncrement:
		env.AutoIncrs[name] = value.Number
	case EvaluationString:
		env.Strings[name] = value.Text
	case EvaluationMacro:
		env.Macros[name] = append([]string(nil), value.Body...)
	case EvaluationAlias:
		env.Aliases[name] = value.Text
	default:
		return fmt.Errorf("unknown evaluation state kind %q", kind)
	}
	return nil
}
func (env *Environment) Delete(kind EvaluationStateKind, name string) error {
	switch kind {
	case EvaluationRegister:
		delete(env.Registers, name)
	case EvaluationIncrement:
		delete(env.AutoIncrs, name)
	case EvaluationString:
		delete(env.Strings, name)
	case EvaluationMacro:
		delete(env.Macros, name)
	case EvaluationAlias:
		delete(env.Aliases, name)
	default:
		return fmt.Errorf("unknown evaluation state kind %q", kind)
	}
	return nil
}
func (env *Environment) Walk(yield func(EvaluationStateKind, string, EvaluationValue) bool) error {
	for name, v := range env.Registers {
		if !yield(EvaluationRegister, name, EvaluationValue{Number: v}) {
			return nil
		}
	}
	for name, v := range env.AutoIncrs {
		if !yield(EvaluationIncrement, name, EvaluationValue{Number: v}) {
			return nil
		}
	}
	for name, v := range env.Strings {
		if !yield(EvaluationString, name, EvaluationValue{Text: v}) {
			return nil
		}
	}
	for name, v := range env.Macros {
		if !yield(EvaluationMacro, name, EvaluationValue{Body: v}) {
			return nil
		}
	}
	for name, v := range env.Aliases {
		if !yield(EvaluationAlias, name, EvaluationValue{Text: v}) {
			return nil
		}
	}
	return nil
}

func evaluationValueFields(kind EvaluationStateKind, value EvaluationValue) error {
	switch kind {
	case EvaluationRegister, EvaluationIncrement:
		if value.Text == "" && len(value.Body) == 0 {
			return nil
		}
	case EvaluationString, EvaluationAlias:
		if value.Number == 0 && len(value.Body) == 0 {
			return nil
		}
	case EvaluationMacro:
		if value.Number == 0 && value.Text == "" {
			return nil
		}
	default:
		return fmt.Errorf("unknown evaluation state kind %q", kind)
	}
	return fmt.Errorf("invalid fields for evaluation state kind %q", kind)
}
func evaluationStateKey(kind EvaluationStateKind, name string) string {
	return string(kind) + "\x00" + name
}
func (m *evaluation) valueSize(kind EvaluationStateKind, value EvaluationValue) (int, error) {
	if err := evaluationValueFields(kind, value); err != nil {
		return 0, err
	}
	if kind == EvaluationMacro {
		return m.bodySize(value.Body)
	}
	if len(value.Text) > m.stateLimit-32 {
		return 0, &EvaluationLimitError{Kind: "state byte", Limit: m.stateLimit}
	}
	return len(value.Text) + 32, nil
}
func (m *evaluation) load(kind EvaluationStateKind, name string) (EvaluationValue, bool, error) {
	if err := m.charge(len(name)); err != nil {
		return EvaluationValue{}, false, err
	}
	value, ok, err := m.store.Load(kind, name)
	if err != nil {
		return EvaluationValue{}, false, fmt.Errorf("load %s %q: %w", kind, name, err)
	}
	if ok {
		size, err := m.valueSize(kind, value)
		if err != nil {
			return EvaluationValue{}, false, err
		}
		if err := m.state(string(kind), name, size); err != nil {
			return EvaluationValue{}, false, err
		}
	}
	return value, ok, nil
}
func (m *evaluation) save(kind EvaluationStateKind, name string, value EvaluationValue) error {
	size, err := m.valueSize(kind, value)
	if err != nil {
		return err
	}
	if err := m.state(string(kind), name, size); err != nil {
		return err
	}
	if kind == EvaluationMacro {
		if err := m.charge(16 * len(value.Body)); err != nil {
			return err
		}
	}
	if err := m.store.Store(kind, name, value); err != nil {
		return fmt.Errorf("store %s %q: %w", kind, name, err)
	}
	return nil
}
func (m *evaluation) erase(kind EvaluationStateKind, name string) error {
	if err := m.charge(len(name)); err != nil {
		return err
	}
	if err := m.store.Delete(kind, name); err != nil {
		return fmt.Errorf("delete %s %q: %w", kind, name, err)
	}
	m.removeState(string(kind), name)
	return nil
}
