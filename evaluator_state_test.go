package groff

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Decorating the small storage interface can observe/reject writes without
// replacing the evaluator or mixing source parsing with state callbacks.
type evaluationObservedState struct {
	EvaluationState
	writes    int
	operation string
	failure   error
}

func (s *evaluationObservedState) Load(k EvaluationStateKind, n string) (EvaluationValue, bool, error) {
	if s.operation == "load" {
		return EvaluationValue{}, false, s.failure
	}
	return s.EvaluationState.Load(k, n)
}
func (s *evaluationObservedState) Store(k EvaluationStateKind, n string, v EvaluationValue) error {
	if s.operation == "store" {
		return s.failure
	}
	s.writes++
	return s.EvaluationState.Store(k, n, v)
}
func (s *evaluationObservedState) Delete(k EvaluationStateKind, n string) error {
	if s.operation == "delete" {
		return s.failure
	}
	return s.EvaluationState.Delete(k, n)
}
func (s *evaluationObservedState) Walk(yield func(EvaluationStateKind, string, EvaluationValue) bool) error {
	if s.operation == "walk" {
		return s.failure
	}
	return s.EvaluationState.Walk(yield)
}

func TestEvaluationComposableState(t *testing.T) {
	backing := NewEnvironment()
	backing.SetString("S", "stored")
	state := &evaluationObservedState{EvaluationState: backing}
	config := &Environment{State: state}
	out, err := EvaluateContext(context.Background(), []Token{&TokenRequestNr{Ident: "X", Value: "3+5*4"}, &TokenRequestAs{Name: "S", StringVal: "-tail"}, TokenText(`\*[S] \n[X]`)}, config)
	if err != nil || len(out) != 1 || out[0] != TokenText("stored-tail 32") || state.writes != 2 {
		t.Fatalf("state composition: %v, %v, writes=%d", out, err, state.writes)
	}
	if backing.GetRegister("X") != 32 || len(config.Registers) != 0 || len(config.Strings) != 0 {
		t.Fatal("evaluation mixed configured and legacy storage")
	}
	if config.Registers != nil || config.Strings != nil || config.Macros != nil || config.Aliases != nil {
		t.Fatal("custom storage initialized legacy maps")
	}
	if _, err := Evaluate([]Token{TokenText(`\n[X]`)}, config); err != nil {
		t.Fatal(err)
	}
}
func TestEvaluationStateErrors(t *testing.T) {
	failure := errors.New("state boundary failure")
	for _, operation := range []string{"load", "store", "delete", "walk"} {
		t.Run(operation, func(t *testing.T) {
			state := &evaluationObservedState{EvaluationState: NewEnvironment(), operation: operation, failure: failure}
			env := NewEnvironment()
			env.State = state
			tokens := []Token{TokenText("prefix"), &TokenRequestNr{Ident: "X", Value: "1"}}
			if operation == "delete" {
				tokens = append(tokens, &TokenRequestRr{Ident: "X"})
			}
			out, err := Evaluate(tokens, env)
			if !errors.Is(err, failure) {
				t.Fatalf("state error identity lost: %v", err)
			}
			if operation != "walk" && (len(out) != 1 || out[0] != TokenText("prefix")) {
				t.Fatalf("lost emitted prefix: %v", out)
			}
		})
	}
}
func TestEvaluationStateBudgetsAndAliases(t *testing.T) {
	for _, configure := range []func(*Environment){
		func(e *Environment) { e.Aliases["A"] = "B"; e.Aliases["B"] = "A" },
		func(e *Environment) { e.Aliases["A"] = "missing" },
		func(e *Environment) { e.Strings["S"] = "string"; e.Macros["S"] = []string{"macro"} },
		func(e *Environment) { e.Strings["S"] = "string"; e.Aliases["A"] = "S"; e.Strings["A"] = "conflict" },
	} {
		backing := NewEnvironment()
		configure(backing)
		env := NewEnvironment()
		env.State = backing
		if _, err := Evaluate(nil, env); err == nil {
			t.Fatal("inconsistent state accepted")
		}
	}
	backing := NewEnvironment()
	backing.SetString("large", strings.Repeat("x", 100))
	env := NewEnvironment()
	env.State = backing
	env.MaxStateBytes = 40
	_, err := Evaluate(nil, env)
	var limit *EvaluationLimitError
	if !errors.As(err, &limit) || limit.Kind != "state byte" {
		t.Fatalf("custom state bypassed initial budget: %v", err)
	}
	body := []string{"original"}
	if err := backing.Store(EvaluationMacro, "M", EvaluationValue{Body: body}); err != nil {
		t.Fatal(err)
	}
	body[0] = "changed"
	value, ok, err := backing.Load(EvaluationMacro, "M")
	if err != nil || !ok || value.Body[0] != "original" {
		t.Fatal("state Store borrowed mutable input")
	}
}

type evaluationEntry struct {
	kind  EvaluationStateKind
	name  string
	value EvaluationValue
}
type evaluationEntryState struct {
	EvaluationState
	entries []evaluationEntry
}

func (s evaluationEntryState) Walk(yield func(EvaluationStateKind, string, EvaluationValue) bool) error {
	for _, entry := range s.entries {
		if !yield(entry.kind, entry.name, entry.value) {
			break
		}
	}
	return nil
}
func TestEvaluationRejectsMalformedStores(t *testing.T) {
	for _, entries := range [][]evaluationEntry{
		{{EvaluationRegister, "X", EvaluationValue{}}, {EvaluationRegister, "X", EvaluationValue{}}},
		{{EvaluationString, "S", EvaluationValue{Number: 1}}},
		{{EvaluationStateKind("unknown"), "S", EvaluationValue{}}},
	} {
		env := NewEnvironment()
		env.State = evaluationEntryState{EvaluationState: NewEnvironment(), entries: entries}
		if _, err := Evaluate(nil, env); err == nil {
			t.Fatalf("malformed state accepted: %+v", entries)
		}
	}
}

func TestEvaluationAliasObjectLifetime(t *testing.T) {
	env := NewEnvironment()
	env.SetString("S", "value")
	tokens := []Token{&TokenRequestAls{New: "Z", Old: "S"}, &TokenRequestAls{New: "A", Old: "S"}, &TokenRequestAls{New: "Middle", Old: "S"}, &TokenRequestRm{Xx: "S"}, &TokenRequestAs{Name: "Z", StringVal: "-tail"}, &TokenRequestRm{Xx: "A"}, TokenText(`\*[Z]`)}
	out, err := Evaluate(tokens, env)
	if err != nil || len(out) != 1 || out[0] != TokenText("value-tail") || env.Aliases["Z"] != "Middle" {
		t.Fatalf("alias promotion/lifetime: %v %v %+v %+v", out, err, env.Aliases, env.Strings)
	}
	env.SetString("Z", "setter")
	if text, ok := env.GetString("Middle"); !ok || text != "setter" {
		t.Fatal("public setter bypassed alias object")
	}
	env.DefineMacro("Z", []string{"macro"})
	out, err = Evaluate([]Token{&TokenUnknownControl{Name: ".Middle"}}, env)
	if err != nil || len(out) != 1 || out[0] != TokenText("macro") {
		t.Fatalf("definition kind change broke aliases: %v %v", out, err)
	}
}

func TestEvaluationReplacingAnExistingAliasGroup(t *testing.T) {
	env := NewEnvironment()
	env.SetString("S", "first")
	env.SetString("T", "second")
	out, err := Evaluate([]Token{&TokenRequestAls{New: "Q", Old: "S"}, &TokenRequestAls{New: "S", Old: "T"}, TokenText(`\*[Q] \*[S] \*[T]`)}, env)
	if err != nil || len(out) != 1 || out[0] != TokenText("first second second") {
		t.Fatalf("alias replacement destroyed earlier object: %v %v", out, err)
	}
}
