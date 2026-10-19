package groff

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

// CustomHandler allows registering handwritten handlers for irregular or complex roff constructs.
type CustomHandler interface {
	CanHandle(cmd string, pkg string) bool
	Parse(cmd string, args string, pkg string) (Token, error)
}

// Registry manages registered custom handlers.
type Registry struct {
	handlers []CustomHandler
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) RegisterCustom(h CustomHandler) {
	r.handlers = append(r.handlers, h)
}

func (r *Registry) DispatchCustom(cmd, args, pkg string) (Token, bool, error) {
	for _, h := range r.handlers {
		if h.CanHandle(cmd, pkg) {
			tok, err := h.Parse(cmd, args, pkg)
			return tok, true, err
		}
	}
	return nil, false, nil
}

var defaultRegistry = NewRegistry()

// RegisterCustomHandler registers a custom handler with the default global registry.
func RegisterCustomHandler(h CustomHandler) {
	defaultRegistry.RegisterCustom(h)
}
