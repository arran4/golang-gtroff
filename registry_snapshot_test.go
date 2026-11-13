package groff

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// These callbacks deliberately register handlers to exercise the lock boundary.
// Pure CanHandle predicates remain recommended; dispatch must not hold a lock
// while invoking a caller's code or admit new registrations halfway through.
type registryRegisteringLine struct {
	*registryTestHandler
	registry    *Registry
	once        sync.Once
	inPredicate bool
	addBlock    bool
}

func (h *registryRegisteringLine) register() {
	h.once.Do(func() {
		if h.addBlock {
			h.registry.RegisterBlock(&blockTestHandler{command: ".LOCAL"})
		} else {
			h.registry.RegisterCustom(&registryTestHandler{})
		}
	})
}
func (h *registryRegisteringLine) CanHandle(cmd, pkg string) bool {
	if h.inPredicate {
		h.register()
	}
	return h.registryTestHandler.CanHandle(cmd, pkg)
}
func (h *registryRegisteringLine) Parse(cmd, args, pkg string) (Token, error) {
	if !h.inPredicate {
		h.register()
	}
	return h.registryTestHandler.Parse(cmd, args, pkg)
}
func TestRegistryReentrantRegistrationUsesOneDispatchSnapshot(t *testing.T) {
	for _, predicate := range []bool{false, true} {
		registry := NewRegistry()
		first := &registryRegisteringLine{registryTestHandler: &registryTestHandler{}, registry: registry, inPredicate: predicate}
		registry.RegisterCustom(first)
		var tok Token
		var err error
		registryCallbackCompletes(t, func() { tok, err = registry.ParseLine(".LOCAL original", "man") })
		if err != nil || tok == nil || tok.String() != ".CUSTOM man original" {
			t.Fatal(predicate, tok, err)
		}
		if first.calls.Load() != 1 {
			t.Fatal("first parse skipped")
		}
		// The next dispatch sees both registrations and rejects ambiguity without
		// executing either Parse callback; the current dispatch saw only its snapshot.
		if _, err = registry.ParseLine(".LOCAL next", "man"); !errors.Is(err, ErrAmbiguousHandler) {
			t.Fatal(predicate, err)
		}
		if first.calls.Load() != 1 {
			t.Fatal("ambiguous dispatch executed Parse")
		}
	}
}

type registryRegisteringBlock struct {
	*blockTestHandler
	registry *Registry
	once     sync.Once
}

func (h *registryRegisteringBlock) CanHandle(cmd, pkg string) bool {
	h.once.Do(func() { h.registry.RegisterBlock(&blockTestHandler{}) })
	return h.blockTestHandler.CanHandle(cmd, pkg)
}
func TestBlockLookupReentrantRegistrationUsesOneSnapshot(t *testing.T) {
	registry := NewRegistry()
	first := &registryRegisteringBlock{blockTestHandler: &blockTestHandler{}, registry: registry}
	registry.RegisterBlock(first)
	var line CustomHandler
	var block BlockHandler
	var err error
	registryCallbackCompletes(t, func() { line, block, err = registry.documentHandler(".BOX", "man") })
	if err != nil || line != nil || block != first {
		t.Fatal(line, block, err)
	}
	if _, _, err = registry.documentHandler(".BOX", "man"); !errors.Is(err, ErrAmbiguousHandler) {
		t.Fatal(err)
	}
	if first.begins.Load() != 0 || first.parses.Load() != 0 {
		t.Fatal("selection executed block callbacks")
	}
}
func TestNilRegistryDispatchAndGeneratedFallback(t *testing.T) {
	var registry *Registry
	if _, handled, err := registry.DispatchCustom(".UNREGISTERED", "", "man"); err != nil || handled {
		t.Fatal(handled, err)
	}
	tok, err := registry.ParseLine(".TH TITLE 1", "man")
	if _, ok := tok.(*TokenMacroTH); err != nil || !ok {
		t.Fatal(tok, err)
	}
	line, block, err := registry.documentHandler(".TS", "roff")
	if err != nil || line != nil || block == nil {
		t.Fatal(line, block, err)
	}
}

func TestRegistryLookupSnapshotsLineAndBlockHandlersTogether(t *testing.T) {
	registry := NewRegistry()
	first := &registryRegisteringLine{registryTestHandler: &registryTestHandler{}, registry: registry, inPredicate: true, addBlock: true}
	registry.RegisterCustom(first)
	var line CustomHandler
	var block BlockHandler
	var err error
	registryCallbackCompletes(t, func() { line, block, err = registry.documentHandler(".LOCAL", "man") })
	if err != nil || line != first || block != nil {
		t.Fatal(line, block, err)
	}
	if _, _, err = registry.documentHandler(".LOCAL", "man"); !errors.Is(err, ErrAmbiguousHandler) {
		t.Fatal(err)
	}
	if first.calls.Load() != 0 {
		t.Fatal("selection ran a Parse callback")
	}
}

// A broken lock boundary should fail promptly rather than consume the complete
// suite timeout. This bounds a possible deadlock; it is not a performance gate.
func registryCallbackCompletes(t *testing.T, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { call(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("registry callback blocked during reentrant registration")
	}
}
