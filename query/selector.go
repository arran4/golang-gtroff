// Package query compiles stateless selectors for source-aware roff inspection.
// Parsed selectors and immutable Go builders share one representation. Queries
// never acquire filesystem capabilities, evaluate roff or mutate documents.
package query

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Kind identifies syntax or semantic structure within a chosen execution view.
type Kind string

const (
	Any            Kind = "*"
	Source         Kind = "source"
	Document       Kind = "document"
	Section        Kind = "section"
	Subsection     Kind = "subsection"
	Definition     Kind = "definition"
	Term           Kind = "term"
	Paragraph      Kind = "paragraph"
	DefinitionList Kind = "definition-list"
	Indent         Kind = "indent"
	Preformatted   Kind = "preformatted"
	Space          Kind = "space"
	Unsupported    Kind = "unsupported"
	Block          Kind = "block"
	Line           Kind = "line"
	Inline         Kind = "inline"
	Macro          Kind = "macro"
	Request        Kind = "request"
	Comment        Kind = "comment"
	Unknown        Kind = "unknown"
)

// Field names a built-in source, syntax or semantic property. Arg returns a
// parsed control-argument field. Text is view-specific, Raw is original source,
// and NormalizedText folds whitespace and Unicode case for both operands.
type Field string

const (
	InlineKind     Field = "inline-kind"
	Name           Field = "name"
	MacroName      Field = "macro"
	Package        Field = "package"
	TokenKind      Field = "token-kind"
	Text           Field = "text"
	NormalizedText Field = "normalized-text"
	Raw            Field = "raw"
	TermText       Field = "term"
	SourceName     Field = "source"
	SourceID       Field = "source-id"
	NodeID         Field = "id"
	StartLine      Field = "line"
	StartColumn    Field = "column"
	StartOffset    Field = "offset"
	EndOffset      Field = "end-offset"
	IncludeDepth   Field = "include-depth"
	Opaque         Field = "opaque"
)

func Arg(index int) Field { return Field("arg." + strconv.Itoa(index)) }

// Operator is an explicit predicate operation. Ordering comparisons are numeric
// and only available for location/identity/depth fields. Exists uses no value.
type Operator string

const (
	Exists       Operator = ""
	Equal        Operator = "="
	NotEqual     Operator = "!="
	Contains     Operator = "*="
	Prefix       Operator = "^="
	Suffix       Operator = "$="
	Less         Operator = "<"
	LessEqual    Operator = "<="
	Greater      Operator = ">"
	GreaterEqual Operator = ">="
)

var ErrSelector = errors.New("invalid query selector")

// SyntaxError identifies a zero-based byte offset in a selector. It never refers
// to a roff source position. errors.Is(err, ErrSelector) classifies parse errors.
type SyntaxError struct {
	Offset  int
	Message string
}

func (e *SyntaxError) Error() string { return fmt.Sprintf("selector byte %d: %s", e.Offset, e.Message) }
func (e *SyntaxError) Unwrap() error { return ErrSelector }

type relation uint8

const (
	descendant relation = iota
	child
)

type predicate struct {
	field Field
	op    Operator
	value string
}
type step struct {
	kind       Kind
	relation   relation
	predicates []predicate
}

// Selector is an immutable query value. Select/Where/Child/Descendant build the
// same representation as ParseSelector. Zero selectors and invalid builders are
// rejected by Validate and execution; no procedural state is retained.
type Selector struct {
	steps []step
	err   error
}

func Select(kind Kind) Selector {
	if kind == "item" {
		kind = Definition
	}
	return Selector{steps: []step{{kind: kind}}}
}
func (s Selector) clone() Selector {
	s.steps = append([]step(nil), s.steps...)
	for i := range s.steps {
		s.steps[i].predicates = append([]predicate(nil), s.steps[i].predicates...)
	}
	return s
}
func (s Selector) Where(field Field, op Operator, value string) Selector {
	s = s.clone()
	if len(s.steps) == 0 {
		s.err = fmt.Errorf("%w: predicate has no node step", ErrSelector)
		return s
	}
	s.steps[len(s.steps)-1].predicates = append(s.steps[len(s.steps)-1].predicates, predicate{field, op, value})
	return s
}
func (s Selector) Child(next Selector) Selector      { return s.then(next, child) }
func (s Selector) Descendant(next Selector) Selector { return s.then(next, descendant) }
func (s Selector) then(next Selector, r relation) Selector {
	s = s.clone()
	next = next.clone()
	if len(s.steps) == 0 || len(next.steps) == 0 {
		s.err = fmt.Errorf("%w: empty structural step", ErrSelector)
		return s
	}
	next.steps[0].relation = r
	s.steps = append(s.steps, next.steps...)
	if s.err == nil {
		s.err = next.err
	}
	return s
}

// Validate checks the conservative grammar bounds for both builder and parser:
// 64 structural steps, 128 predicates and 16 KiB of predicate values in total.
func (s Selector) Validate() error {
	if s.err != nil {
		return s.err
	}
	if len(s.steps) == 0 || len(s.steps) > 64 {
		return fmt.Errorf("%w: expected 1..64 structural steps", ErrSelector)
	}
	predicates, values := 0, 0
	for _, step := range s.steps {
		if !validKind(step.kind) {
			return fmt.Errorf("%w: unknown node kind %q", ErrSelector, step.kind)
		}
		for _, p := range step.predicates {
			predicates++
			values += len(p.value)
			if !validField(p.field) {
				return fmt.Errorf("%w: unknown field %q", ErrSelector, p.field)
			}
			switch p.op {
			case Exists:
				if p.value != "" {
					return fmt.Errorf("%w: existence predicate has a value", ErrSelector)
				}
			case Equal, NotEqual, Contains, Prefix, Suffix, Less, LessEqual, Greater, GreaterEqual:
			default:
				return fmt.Errorf("%w: unknown operator %q", ErrSelector, p.op)
			}
			if numericField(p.field) && p.op != Exists {
				if _, err := strconv.ParseUint(p.value, 10, 64); err != nil {
					return fmt.Errorf("%w: %s requires an unsigned decimal value", ErrSelector, p.field)
				}
				if p.op == Contains || p.op == Prefix || p.op == Suffix {
					return fmt.Errorf("%w: numeric field cannot use a text operator", ErrSelector)
				}
			} else if p.op == Less || p.op == LessEqual || p.op == Greater || p.op == GreaterEqual {
				return fmt.Errorf("%w: ordering requires a numeric field", ErrSelector)
			}
			if p.field == Opaque && p.op != Exists && (p.op != Equal && p.op != NotEqual || p.value != "true" && p.value != "false") {
				return fmt.Errorf("%w: opaque requires a boolean equality", ErrSelector)
			}
		}
	}
	if predicates > 128 || values > 16<<10 {
		return fmt.Errorf("%w: predicate budget exceeded", ErrSelector)
	}
	return nil
}
func validKind(kind Kind) bool {
	switch kind {
	case Any, Source, Document, Section, Subsection, Definition, Term, Paragraph, DefinitionList, Indent, Preformatted, Space, Unsupported, Block, Line, Inline, Macro, Request, Comment, Unknown:
		return true
	}
	return false
}
func numericField(field Field) bool {
	switch field {
	case SourceID, StartLine, StartColumn, StartOffset, EndOffset, IncludeDepth:
		return true
	}
	return false
}
func validField(field Field) bool {
	switch field {
	case Name, MacroName, Package, TokenKind, InlineKind, Text, NormalizedText, Raw, TermText, SourceName, SourceID, NodeID, StartLine, StartColumn, StartOffset, EndOffset, IncludeDepth, Opaque:
		return true
	}
	if strings.HasPrefix(string(field), "arg.") {
		n, err := strconv.Atoi(strings.TrimPrefix(string(field), "arg."))
		return err == nil && n >= 0 && n < 1024
	}
	return false
}
