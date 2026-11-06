package query

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

// Match has a one-based ordinal and stable identity within the captured source
// set/view. ASTIDs are contributors, Span is original-file byte provenance, and
// Raw is the exact original slice. Text is interpreted only in semantic view;
// NormalizedText folds case/whitespace. JSON replaces invalid UTF-8 in strings.
// Term nodes identify complete physical tag lines, not character-exact glyphs.
type Match struct {
	SchemaVersion  int                 `json:"schemaVersion"`
	Index          int                 `json:"index"`
	ID             string              `json:"id"`
	View           View                `json:"view"`
	Kind           Kind                `json:"kind"`
	Source         SourceInfo          `json:"source"`
	Span           groff.Span          `json:"span"`
	ASTIDs         []groff.NodeID      `json:"astIds,omitempty"`
	SemanticID     man.ID              `json:"semanticId,omitempty"`
	Name           string              `json:"name,omitempty"`
	Term           string              `json:"term,omitempty"`
	Text           string              `json:"text"`
	NormalizedText string              `json:"normalizedText"`
	Raw            string              `json:"raw"`
	Diagnostics    []source.Diagnostic `json:"diagnostics,omitempty"`
}

// Result is a versioned report. Complete means query enumeration completed over
// the supplied snapshot; it does not prove EOF, successful include resolution,
// valid syntax or semantic support. Source.Complete and diagnostics retain those
// separate facts. On execution errors any returned matches are a valid prefix.
// Error describes execution failure. InputError is optional host loading context;
// Execute never performs input I/O and leaves that field empty.
type Result struct {
	// WorkUnits counts charged query work for composed consumers; not serialized.
	WorkUnits     int                 `json:"-"`
	Command       string              `json:"command"`
	Error         string              `json:"error,omitempty"`
	InputError    string              `json:"inputError,omitempty"`
	SchemaVersion int                 `json:"schemaVersion"`
	View          View                `json:"view"`
	Complete      bool                `json:"complete"`
	Matches       []Match             `json:"matches"`
	Diagnostics   []source.Diagnostic `json:"diagnostics,omitempty"`
}

// Execute evaluates a selector without modifying the source set or retaining
// state between calls. Includes appear as child source occurrences attached to
// the original directive's owner. Semantic grouping is local to each physical
// file: child files do not change the surrounding file's heading/font state.
// The syntax view preserves original AST links within each source. No file opens
// or callbacks into a script engine occur during execution.
func Execute(ctx context.Context, set *source.Set, selector Selector, o Options) (result *Result, err error) {
	var accounting *meter
	defer func() {
		if result != nil && accounting != nil {
			result.WorkUnits = o.MaxWork - accounting.remaining
		}
		if result != nil && err != nil {
			result.Error = err.Error()
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	if o.View == "" {
		o.View = Semantic
	}
	result = &Result{Command: "query", SchemaVersion: 1, View: o.View, Matches: []Match{}}
	if err := selector.Validate(); err != nil {
		return result, err
	}
	if o.View != Syntax && o.View != Semantic {
		return result, fmt.Errorf("query view must be syntax or semantic")
	}
	if o.MaxNodes <= 0 {
		o.MaxNodes = 200000
	}
	if o.MaxWork <= 0 {
		o.MaxWork = 64 << 20
	}
	if o.MaxMatches <= 0 {
		o.MaxMatches = 10000
	}
	if o.MaxResultBytes <= 0 {
		o.MaxResultBytes = 16 << 20
	}
	m := &meter{ctx: ctx, remaining: o.MaxWork}
	accounting = m
	g, err := buildGraph(set, o, m)
	result.Diagnostics = g.diagnostics
	if err != nil {
		return result, err
	}
	// Each bit is one selector step. Propagated ancestor masks implement arbitrary
	// descendant chains in O(projected nodes * selector steps), without recursive
	// backtracking through combinations of matching ancestors.
	matched, ancestors := make([]uint64, len(g.nodes)), make([]uint64, len(g.nodes))
	stack := make([]int, 0, len(g.roots))
	for i := len(g.roots) - 1; i >= 0; i-- {
		stack = append(stack, g.roots[i])
	}
	outputBytes := 0
	for len(stack) > 0 {
		index := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		n := &g.nodes[index]
		if err := m.spend(1); err != nil {
			return result, err
		}
		if n.parent >= 0 {
			ancestors[index] = ancestors[n.parent] | matched[n.parent]
		}
		cache := map[Field]string{}
		for j, step := range selector.steps {
			if err := m.spend(1); err != nil {
				return result, err
			}
			if j > 0 {
				previous := uint64(1) << uint(j-1)
				if n.parent < 0 || step.relation == child && matched[n.parent]&previous == 0 || step.relation == descendant && ancestors[index]&previous == 0 {
					continue
				}
			}
			if !kindMatches(n, step.kind) {
				continue
			}
			ok := true
			for _, predicate := range step.predicates {
				value, err := fieldValue(n, predicate.field, cache, m)
				if err != nil {
					return result, err
				}
				if err := m.spend(len(value) + len(predicate.value) + 1); err != nil {
					return result, err
				}
				if !testPredicate(predicate, value) {
					ok = false
					break
				}
			}
			if ok {
				matched[index] |= uint64(1) << uint(j)
			}
		}
		if matched[index]&(uint64(1)<<uint(len(selector.steps)-1)) != 0 {
			if len(result.Matches) >= o.MaxMatches {
				return result, ErrLimit
			}
			text, err := fieldValue(n, Text, cache, m)
			if err != nil {
				return result, err
			}
			raw, err := fieldValue(n, Raw, cache, m)
			if err != nil {
				return result, err
			}
			name, err := fieldValue(n, Name, cache, m)
			if err != nil {
				return result, err
			}
			term, err := fieldValue(n, TermText, cache, m)
			if err != nil {
				return result, err
			}
			if err := m.spend(len(text)); err != nil {
				return result, err
			}
			normalized := normalize(text)
			findings := g.bySource[n.file.ID()]
			first := sort.Search(len(findings), func(i int) bool { return findings[i].Position.Offset >= n.span.Start.Offset })
			last := sort.Search(len(findings), func(i int) bool { return findings[i].Position.Offset >= n.span.End.Offset })
			size := 256 + len(n.identity) + len(n.file.Name()) + len(text) + len(raw) + len(name) + len(term) + len(normalized) + len(n.astIDs)*8
			for _, finding := range findings[first:last] {
				size += len(finding.Message) + 128
			}
			if size > o.MaxResultBytes-outputBytes {
				return result, ErrLimit
			}
			outputBytes += size
			result.Matches = append(result.Matches, Match{SchemaVersion: 1, Index: len(result.Matches) + 1, ID: n.identity, View: o.View, Kind: n.kind, Source: g.sourceInfo[n.file.ID()], Span: n.span, ASTIDs: append([]groff.NodeID(nil), n.astIDs...), SemanticID: n.semantic, Name: name, Term: term, Text: text, NormalizedText: normalized, Raw: raw, Diagnostics: append([]source.Diagnostic(nil), findings[first:last]...)})
		}
		for i := len(n.children) - 1; i >= 0; i-- {
			stack = append(stack, n.children[i])
		}
	}
	result.Complete = true
	return result, nil
}

func fieldValue(n *graphNode, field Field, cache map[Field]string, m *meter) (string, error) {
	if value, ok := cache[field]; ok {
		return value, nil
	}
	var attrs groff.ASTAttributes
	if field == Name || field == MacroName || field == Package || field == TokenKind || field == Opaque || strings.HasPrefix(string(field), "arg.") {
		attrs = n.attributes()
	}
	var value string
	switch field {
	case Name:
		switch n.kind {
		case Source:
			value = n.file.Name()
		case Section, Subsection:
			if n.model != nil {
				value = n.model.Node(n.semantic).Heading
			} else if n.syntax != nil {
				value = n.syntax.Attributes().Heading
			}
		case Definition, Term:
			value = n.termText()
		case Inline:
			if n.syntax != nil {
				if inline, ok := n.syntax.Inline(); ok {
					value = inline.Name
				}
			}
		default:
			value = attrs.Name
		}
	case MacroName:
		value = attrs.Name
	case Package:
		value = attrs.Package
		if value == "" && n.model != nil {
			value = "man"
		}
	case TokenKind:
		value = string(attrs.TokenKind)
	case InlineKind:
		if n.syntax != nil {
			if inline, ok := n.syntax.Inline(); ok {
				value = string(inline.Kind)
			}
		}
	case Text:
		var err error
		value, err = n.text(m)
		if err != nil {
			return "", err
		}
	case NormalizedText:
		text, err := fieldValue(n, Text, cache, m)
		if err != nil {
			return "", err
		}
		if err := m.spend(len(text)); err != nil {
			return "", err
		}
		value = normalize(text)
	case Raw:
		value = n.file.AST().Source()[n.span.Start.Offset:n.span.End.Offset]
	case TermText:
		value = n.termText()
	case SourceName:
		value = n.file.Name()
	case SourceID:
		value = strconv.FormatUint(uint64(n.file.ID()), 10)
	case NodeID:
		value = n.identity
	case StartLine:
		value = strconv.Itoa(n.span.Start.Line)
	case StartColumn:
		value = strconv.Itoa(n.span.Start.Column)
	case StartOffset:
		value = strconv.Itoa(n.span.Start.Offset)
	case EndOffset:
		value = strconv.Itoa(n.span.End.Offset)
	case IncludeDepth:
		value = strconv.Itoa(n.file.Depth())
	case Opaque:
		value = strconv.FormatBool(attrs.Opaque || n.syntax != nil && n.syntax.Attributes().Opaque)
	default:
		if strings.HasPrefix(string(field), "arg.") {
			index, _ := strconv.Atoi(strings.TrimPrefix(string(field), "arg."))
			if index >= 0 && index < len(attrs.Arguments) {
				value = attrs.Arguments[index].Value
			}
		}
	}
	cache[field] = value
	return value, nil
}
func testPredicate(p predicate, value string) bool {
	if p.op == Exists {
		if p.field == Opaque {
			return value == "true"
		}
		return value != ""
	}
	needle := p.value
	if p.field == NormalizedText {
		needle = normalize(needle)
	}
	if numericField(p.field) {
		a, errA := strconv.ParseUint(value, 10, 64)
		b, errB := strconv.ParseUint(needle, 10, 64)
		if errA != nil || errB != nil {
			return false
		}
		switch p.op {
		case Equal:
			return a == b
		case NotEqual:
			return a != b
		case Less:
			return a < b
		case LessEqual:
			return a <= b
		case Greater:
			return a > b
		case GreaterEqual:
			return a >= b
		}
		return false
	}
	switch p.op {
	case Equal:
		return value == needle
	case NotEqual:
		return value != needle
	case Contains:
		return strings.Contains(value, needle)
	case Prefix:
		return strings.HasPrefix(value, needle)
	case Suffix:
		return strings.HasSuffix(value, needle)
	}
	return false
}
