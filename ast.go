package groff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ASTKind describes syntax structure, independently of a token's macro package.
type ASTKind string

const (
	// ASTDocument owns the entire captured source, including any recovery tail.
	ASTDocument ASTKind = "document"
	// ASTSection groups a man .SH heading and subsequent syntax up to the next .SH.
	ASTSection ASTKind = "section"
	// ASTSubsection groups a man .SS heading within a section, or at the root.
	ASTSubsection ASTKind = "subsection"
	// ASTBlock groups an explicit opener, its body and any captured terminator.
	// Opaque blocks retain syntax without interpreting stored controls as live.
	ASTBlock ASTKind = "block"
	// ASTLine is a physical source line, or an opaque remainder on budget failure.
	// Blank lines, comments, unknown controls and delimiters remain visible here.
	ASTLine ASTKind = "line"
	// ASTInline describes inline syntax within a physical line. Its source span
	// overlaps its owning line; it is not an additional physical contribution.
	ASTInline ASTKind = "inline"
)

// NodeID is a stable, one-based preorder index within one parsed source snapshot.
// IDs are reproducible for the same source/options, not persistent across edits.
type NodeID uint64

// AST is an immutable source snapshot. ParseAST never evaluates requests or
// resolves includes. The older Document and Node token APIs remain independent.
type AST struct {
	source         string
	sourceComplete bool
	root           *ASTNode
	nodes          []*ASTNode
	diagnostics    []Diagnostic
}

// ASTNode has immutable structure and metadata. All slice accessors return copies.
// A zero ASTNode is not a parsed node; obtain nodes through ParseAST.
type ASTNode struct {
	block       *BlockSource
	tree        *AST
	id          NodeID
	kind        ASTKind
	parent      *ASTNode
	children    []*ASTNode
	sibling     int
	limit       int // exclusive subtree end in tree.nodes
	span        Span
	attrs       ASTAttributes
	tokenData   json.RawMessage
	tokenType   reflect.Type
	inline      *InlineToken
	diagnostics []Diagnostic
	text        string // line/inline contribution; containers concatenate physical lines
}

// ASTAttributes is the syntax-level inspection view. Names use dot-normalized
// control spelling (including the leading dot); raw source retains apostrophes.
// Argument Start/End positions are rune indexes relative to the argument string,
// as in LexArguments. Source spans always use byte offsets.
type ASTAttributes struct {
	Name          string           `json:"name,omitempty"`
	Package       string           `json:"package,omitempty"`
	TokenKind     TokenKind        `json:"tokenKind,omitempty"`
	Arguments     []ArgumentLexeme `json:"arguments,omitempty"`
	Heading       string           `json:"heading,omitempty"`
	EndMarker     string           `json:"endMarker,omitempty"`
	UnresolvedEnd string           `json:"unresolvedEnd,omitempty"`
	Closed        bool             `json:"closed,omitempty"`
	Opaque        bool             `json:"opaque,omitempty"`
}

func (a *AST) Root() *ASTNode { return a.root }
func (a *AST) Source() string { return a.source }

// SourceComplete reports whether this source's reader actually returned EOF.
// It does not imply valid syntax or success; check ParseAST's error/diagnostics.
func (a *AST) SourceComplete() bool { return a.sourceComplete }

func (a *AST) Diagnostics() []Diagnostic { return append([]Diagnostic(nil), a.diagnostics...) }

// Node looks up a document-local identity in O(1); zero and out-of-range IDs are absent.
func (a *AST) Node(id NodeID) *ASTNode {
	if id == 0 || id > NodeID(len(a.nodes)) {
		return nil
	}
	return a.nodes[id-1]
}

// BlockSource returns a caller-owned capture snapshot for a block node. Raw
// strings and spans remain independent of handler tokens and String formatting.
func (n *ASTNode) BlockSource() (*BlockSource, bool) {
	if n.block == nil {
		return nil, false
	}
	copy := n.block.clone()
	// Derive physical body metadata lazily from the immutable source. Retaining
	// one body-line slice per enclosing definition would multiply memory by depth.
	end := n.span.End.Offset
	if copy.Closed {
		end = copy.Terminator.Span.Start.Offset
	}
	if copy.Tail.RawSource != "" {
		end = copy.Tail.Span.Start.Offset
	}
	start := copy.Header.Line.Span.End
	copy.Body = physicalBlockLines(n.tree.source[start.Offset:end], start)
	return &copy, true
}

func (n *ASTNode) ID() NodeID           { return n.id }
func (n *ASTNode) Kind() ASTKind        { return n.kind }
func (n *ASTNode) Parent() *ASTNode     { return n.parent }
func (n *ASTNode) Children() []*ASTNode { return append([]*ASTNode(nil), n.children...) }
func (n *ASTNode) Span() Span           { return n.span }
func (n *ASTNode) Raw() string          { return n.tree.source[n.span.Start.Offset:n.span.End.Offset] }
func (n *ASTNode) Attributes() ASTAttributes {
	a := n.attrs
	a.Arguments = append([]ArgumentLexeme(nil), a.Arguments...)
	return a
}
func (n *ASTNode) Diagnostics() []Diagnostic { return append([]Diagnostic(nil), n.diagnostics...) }

// Path is the slash-separated, zero-based child path; the document path is "/".
func (n *ASTNode) Path() string {
	var parts []string
	for p := n; p.parent != nil; p = p.parent {
		parts = append(parts, strconv.Itoa(p.sibling))
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return "/" + strings.Join(parts, "/")
}
func (n *ASTNode) PreviousSibling() *ASTNode {
	if n.parent == nil || n.sibling == 0 {
		return nil
	}
	return n.parent.children[n.sibling-1]
}
func (n *ASTNode) NextSibling() *ASTNode {
	if n.parent == nil || n.sibling+1 == len(n.parent.children) {
		return nil
	}
	return n.parent.children[n.sibling+1]
}

// Ancestors returns nearest parent first, ending with the document root.
func (n *ASTNode) Ancestors() []*ASTNode {
	var out []*ASTNode
	for p := n.parent; p != nil; p = p.parent {
		out = append(out, p)
	}
	return out
}

// Descendants returns preorder, excluding n, in O(number of descendants).
func (n *ASTNode) Descendants() []*ASTNode {
	return append([]*ASTNode(nil), n.tree.nodes[int(n.id):n.limit]...)
}

// Token returns a caller-owned JSON-round-tripped copy of the concrete line
// or custom block token. Custom tokens must support JSON round trips for
// inspectable state. Ordinary containers and inline nodes have no Token.
// Mutating a copy cannot edit the AST.
func (n *ASTNode) Token() (Token, error) {
	if n.tokenType == nil {
		return nil, nil
	}
	typ := n.tokenType
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	v := reflect.New(typ)
	if err := json.Unmarshal(n.tokenData, v.Interface()); err != nil {
		return nil, err
	}
	if n.tokenType.Kind() != reflect.Ptr {
		v = v.Elem()
	}
	tok, ok := v.Interface().(Token)
	if !ok {
		return nil, fmt.Errorf("token snapshot has no Token implementation")
	}
	return tok, nil
}
func (n *ASTNode) Inline() (InlineToken, bool) {
	if n.inline == nil {
		return InlineToken{}, false
	}
	return *n.inline, true
}

// Text extracts syntax text in source order, once per physical line. Known
// macro/request lines contribute unquoted argument values, comments/delimiters
// contribute nothing, opaque/unknown/malformed lines retain literal source.
// Escapes and references remain roff syntax; no semantic expansion is implied.
func (n *ASTNode) Text() string {
	if n.kind == ASTLine || n.kind == ASTInline {
		return n.text
	}
	var b strings.Builder
	n.Walk(func(p *ASTNode) WalkAction {
		if p.kind == ASTLine {
			b.WriteString(p.text)
			return WalkPrune
		}
		return WalkContinue
	})
	return b.String()
}

// MarshalJSON encodes a deterministic flat preorder table with explicit links.
// Raw byte preservation is provided by Source/Raw, including invalid UTF-8;
// JSON itself follows encoding/json's UTF-8 replacement rules.
func (a *AST) MarshalJSON() ([]byte, error) {
	type row struct {
		ID          NodeID          `json:"id"`
		Parent      NodeID          `json:"parent,omitempty"`
		Children    []NodeID        `json:"children,omitempty"`
		Kind        ASTKind         `json:"kind"`
		Span        Span            `json:"span"`
		Attributes  ASTAttributes   `json:"attributes"`
		Token       json.RawMessage `json:"token,omitempty"`
		Inline      *InlineToken    `json:"inline,omitempty"`
		Diagnostics []Diagnostic    `json:"diagnostics,omitempty"`
	}
	rows := make([]row, 0, len(a.nodes))
	for _, n := range a.nodes {
		r := row{ID: n.id, Kind: n.kind, Span: n.span, Attributes: n.Attributes(), Token: n.tokenData, Inline: n.inline, Diagnostics: n.diagnostics}
		if n.parent != nil {
			r.Parent = n.parent.id
		}
		for _, c := range n.children {
			r.Children = append(r.Children, c.id)
		}
		rows = append(rows, r)
	}
	return json.Marshal(struct {
		Nodes       []row        `json:"nodes"`
		Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	}{rows, a.diagnostics})
}
