package groff

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// TokenKind identifies the semantic category of a token or node.
type TokenKind string

const (
	KindMacro    TokenKind = "Macro"
	KindRequest  TokenKind = "Request"
	KindString   TokenKind = "String"
	KindRegister TokenKind = "Register"
	KindText     TokenKind = "Text"
	KindComment  TokenKind = "Comment"
	KindBlock    TokenKind = "Block"
	KindControl  TokenKind = "Control"
	KindUnknown  TokenKind = "Unknown"
)

// Node extends Token with structured metadata.
type Node interface {
	Token
	TokenName() string
	TokenKind() TokenKind
	PackageName() string
}

// TokenUnknownControl represents an unrecognized control line preserved in permissive parse mode.
type TokenUnknownControl struct {
	ControlChar rune
	Name        string
	RawArgs     string
	RawLine     string
}

func (t *TokenUnknownControl) String() string {
	if t.RawLine != "" {
		return t.RawLine
	}
	if t.RawArgs == "" {
		return string(t.ControlChar) + t.Name
	}
	return string(t.ControlChar) + t.Name + " " + t.RawArgs
}

func (t *TokenUnknownControl) TokenName() string {
	return t.Name
}

func (t *TokenUnknownControl) TokenKind() TokenKind {
	return KindUnknown
}

func (t *TokenUnknownControl) PackageName() string {
	return ""
}

var _ Token = (*TokenUnknownControl)(nil)
var _ Node = (*TokenUnknownControl)(nil)

// TokenEmptyControl represents a control line with just . or ' and no command.
type TokenEmptyControl struct {
	ControlChar rune
}

func (t TokenEmptyControl) String() string {
	return string(t.ControlChar)
}

func (t TokenEmptyControl) TokenName() string {
	return string(t.ControlChar)
}

func (t TokenEmptyControl) TokenKind() TokenKind {
	return KindControl
}

func (t TokenEmptyControl) PackageName() string {
	return ""
}

var _ Token = TokenEmptyControl{}
var _ Node = TokenEmptyControl{}

// Node implementations for TokenText and TokenComment
func (t TokenText) TokenName() string {
	return ""
}

func (t TokenText) TokenKind() TokenKind {
	return KindText
}

func (t TokenText) PackageName() string {
	return ""
}

var _ Node = TokenText("")

func (t TokenComment) TokenName() string {
	return `.\"`
}

func (t TokenComment) TokenKind() TokenKind {
	return KindComment
}

func (t TokenComment) PackageName() string {
	return ""
}

var _ Node = TokenComment("")

// CustomHandler parses individual control lines before generated dispatch.
// cmd retains its original . or ' prefix; args excludes leading whitespace.
// CanHandle should be a pure predicate. Handlers used concurrently must protect
// their own state. Multiline document structure remains the parser's concern.
type CustomHandler interface {
	CanHandle(cmd string, pkg string) bool
	Parse(cmd string, args string, pkg string) (Token, error)
}

var (
	// ErrAmbiguousHandler reports overlapping predicates for an encountered line.
	ErrAmbiguousHandler = errors.New("multiple custom handlers match")
	// ErrInvalidHandler reports a nil registration or successful nil token result.
	ErrInvalidHandler = errors.New("invalid custom handler")
)

// Registry manages custom line handlers. Its zero value is usable. Registration
// and dispatch may run concurrently; do not copy a Registry after first use.
// A document/AST parse snapshots registrations once at entry, including includes.
type Registry struct {
	mu       sync.RWMutex
	handlers []CustomHandler
}

func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterCustom adds a handler. Invalid/overlapping registrations are reported
// by DispatchCustom when a control line is encountered, preserving this legacy
// method's signature. Prefer WithRegistry for caller-local configuration.
func (r *Registry) RegisterCustom(h CustomHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

// DispatchCustom selects exactly one matching custom handler. Overlapping
// predicates (including duplicate registrations) return ErrAmbiguousHandler
// before Parse is called. Nil handlers/tokens return ErrInvalidHandler.
func (r *Registry) DispatchCustom(cmd, args, pkg string) (Token, bool, error) {
	var selected CustomHandler
	for _, h := range r.snapshot().handlers {
		if nilInterface(h) {
			return nil, true, fmt.Errorf("%w: nil registration", ErrInvalidHandler)
		}
		if h.CanHandle(cmd, pkg) {
			if selected != nil {
				return nil, true, fmt.Errorf("%w for %s in package %q", ErrAmbiguousHandler, cmd, pkg)
			}
			selected = h
		}
	}
	if selected == nil {
		return nil, false, nil
	}
	tok, err := selected.Parse(cmd, args, pkg)
	if err == nil && nilInterface(tok) {
		err = fmt.Errorf("%w: %s returned a nil token", ErrInvalidHandler, cmd)
	}
	return tok, true, err
}

// ParseLine parses one line with this registry and explicit package context.
// Generated tokens remain the fallback when no custom handler matches.
func (r *Registry) ParseLine(line, pkg string) (Token, error) {
	return parseLineWithRegistry(line, pkg, false, r)
}

// ParseLinePermissive preserves unknown controls while still reporting invalid
// or ambiguous custom handler results.
func (r *Registry) ParseLinePermissive(line, pkg string) (Token, error) {
	return parseLineWithRegistry(line, pkg, true, r)
}

func (r *Registry) snapshot() *Registry {
	if r == nil {
		return NewRegistry()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return &Registry{handlers: append([]CustomHandler(nil), r.handlers...)}
}

func registryForOptions(o ParserOptions) *Registry {
	r := o.Registry
	if r == nil {
		r = defaultRegistry
	}
	snapshot := r.snapshot()
	snapshot.handlers = append(snapshot.handlers, o.CustomHandlers...)
	return snapshot
}

func nilInterface(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

var defaultRegistry = NewRegistry()

// RegisterCustomHandler registers a custom handler with the default global registry.
// Prefer WithRegistry or WithCustomHandler for parse-local configuration.
func RegisterCustomHandler(h CustomHandler) {
	defaultRegistry.RegisterCustom(h)
}
