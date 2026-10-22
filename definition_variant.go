package groff

import (
	"errors"
	"fmt"
	"strings"
)

// TokenMacroDefinitionVariant retains the distinct syntax of de1/dei/dei1 and
// am1/ami/ami1. Name/End are parsed header values, not evaluated string references.
// Delimiter is the actual dot-normalized closing command selected for capture.
// Raw bytes and closing arguments remain in DocumentToken.Block/AST BlockSource.
type TokenMacroDefinitionVariant struct {
	Request   string
	Name      string
	End       string
	Delimiter string
	Body      []string
}

func (t *TokenMacroDefinitionVariant) String() string {
	var out strings.Builder
	out.WriteString(t.Request + " " + Quote(t.Name))
	if t.End != "" {
		out.WriteString(" " + Quote(t.End))
	}
	out.WriteByte('\n')
	for _, line := range t.Body {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	end := t.Delimiter
	if end == "" {
		end = ".."
	}
	out.WriteString(end)
	return out.String()
}
func (t *TokenMacroDefinitionVariant) TokenName() string    { return t.Request }
func (t *TokenMacroDefinitionVariant) TokenKind() TokenKind { return KindMacro }
func (t *TokenMacroDefinitionVariant) PackageName() string  { return "roff" }

var _ Node = (*TokenMacroDefinitionVariant)(nil)

// BlockNameResolver resolves an indirect definition's end-name reference using
// explicitly supplied state. Structural parsing never reads evaluator state or
// executes preceding .ds/.as requests. The callback is trusted Go code and must
// protect shared state and bound its own work. Results are final names (not roff
// expressions), limited to 64 KiB; empty means the default '..' delimiter.
type BlockNameResolver interface {
	ResolveBlockName(BlockHeader, string) (string, error)
}
type BlockNameResolverFunc func(BlockHeader, string) (string, error)

func (f BlockNameResolverFunc) ResolveBlockName(header BlockHeader, name string) (string, error) {
	return f(header, name)
}

var ErrUnresolvedBlockDelimiter = errors.New("unresolved indirect block delimiter")

// BlockDelimiterError retains the reference and optional resolver error. A body
// with an unknown boundary must stay opaque instead of being parsed top-level.
type BlockDelimiterError struct {
	Reference string
	Err       error
}

func (e *BlockDelimiterError) Error() string {
	message := fmt.Sprintf("%v %q; remaining source preserved as opaque block", ErrUnresolvedBlockDelimiter, e.Reference)
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}
func (e *BlockDelimiterError) Unwrap() []error {
	if e.Err == nil {
		return []error{ErrUnresolvedBlockDelimiter}
	}
	return []error{ErrUnresolvedBlockDelimiter, e.Err}
}
func withBlockNameResolver(handler BlockHandler, resolver BlockNameResolver) BlockHandler {
	if h, ok := handler.(builtinBlockHandler); ok {
		h.resolver = resolver
		return h
	}
	return handler
}
func isMacroDefinitionBlock(cmd string) bool {
	h, ok := generatedBlockHandler(cmd).(builtinBlockHandler)
	return ok && (h.kind == "definition" || h.kind == "append" || h.kind == "definition_variant" || h.kind == "append_variant")
}
func (h builtinBlockHandler) variant() bool {
	return h.kind == "definition_variant" || h.kind == "append_variant"
}
func (h builtinBlockHandler) indirect() bool { return h.variant() && strings.Contains(h.command, "i") }
func (h builtinBlockHandler) resolveDelimiter(header BlockHeader, reference string) (string, error) {
	if nilInterface(h.resolver) {
		return "", &BlockDelimiterError{Reference: reference}
	}
	name, err := h.resolver.ResolveBlockName(header, reference)
	if err == nil && (len(name) > 64<<10 || strings.ContainsAny(name, " \t\r\n\x00") || name != "" && !validBlockCommand("."+name)) {
		err = fmt.Errorf("resolved block name must be a valid name of at most 65536 bytes")
	}
	if err != nil {
		return "", &BlockDelimiterError{Reference: reference, Err: err}
	}
	if name == "" {
		return "..", nil
	}
	return "." + name, nil
}
