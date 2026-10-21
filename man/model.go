// Package man derives a bounded semantic model of typical Linux man pages from
// a source-preserving roff AST. It never evaluates requests or loads resources.
package man

import (
	"errors"
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

type ID uint64
type Kind string

const (
	DocumentNode   Kind = "document"
	Section        Kind = "section"
	Subsection     Kind = "subsection"
	Paragraph      Kind = "paragraph"
	DefinitionList Kind = "definition-list"
	Item           Kind = "item"
	Indent         Kind = "indent"
	Preformatted   Kind = "preformatted"
	Space          Kind = "space"
	Unsupported    Kind = "unsupported"
)

var ErrLimit = errors.New("man model resource limit exceeded")

// Origin identifies physical syntax, not canonical serialization. Runs use
// physical-line spans; container spans extend over their contained content.
type Origin struct {
	ASTIDs []groff.NodeID `json:"astIds,omitempty"`
	Span   groff.Span     `json:"span"`
}
type Style struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Mono   bool `json:"mono,omitempty"`
	Small  bool `json:"small,omitempty"`
}

// State is captured before a semantic node is interpreted. A standalone
// selection has the upstream font, fill, indentation, tag-width and link state.
type State struct {
	Style             Style    `json:"style"`
	Previous          Style    `json:"previous"`
	Fill              bool     `json:"fill"`
	Indents           []string `json:"indents,omitempty"`
	TagIndent         string   `json:"tagIndent,omitempty"`
	Link              string   `json:"link,omitempty"`
	ParagraphDistance string   `json:"paragraphDistance,omitempty"`
}
type Run struct {
	Text        string `json:"text"`
	Style       Style  `json:"style"`
	Link        string `json:"link,omitempty"`
	Reference   string `json:"reference,omitempty"`
	Break       bool   `json:"break,omitempty"`
	Unsupported bool   `json:"unsupported,omitempty"`
	Origin      Origin `json:"origin"`
}
type Metadata struct {
	Title   string `json:"title,omitempty"`
	Section string `json:"section,omitempty"`
	Date    string `json:"date,omitempty"`
	Source  string `json:"source,omitempty"`
	Manual  string `json:"manual,omitempty"`
}

// Node IDs are one-based and contiguous in creation order. Nodes form an
// explicitly linked tree; use Validate after accepting a JSON model from outside.
// Exported fields are caller-owned data and mutations must retain that contract.
type Node struct {
	ID       ID     `json:"id"`
	Parent   ID     `json:"parent,omitempty"`
	Children []ID   `json:"children,omitempty"`
	Kind     Kind   `json:"kind"`
	Heading  string `json:"heading,omitempty"`
	Value    string `json:"value,omitempty"`
	Runs     []Run  `json:"runs,omitempty"`
	Tag      []Run  `json:"tag,omitempty"`
	State    State  `json:"state"`
	Origin   Origin `json:"origin"`
}
type Document struct {
	Version        int                `json:"version"`
	Metadata       Metadata           `json:"metadata"`
	MetadataOrigin *Origin            `json:"metadataOrigin,omitempty"`
	Nodes          []Node             `json:"nodes"`
	Diagnostics    []groff.Diagnostic `json:"diagnostics,omitempty"`
	source         string
}

func (d *Document) Source() string { return d.source }
func (d *Document) Node(id ID) *Node {
	if id == 0 || id > ID(len(d.Nodes)) {
		return nil
	}
	return &d.Nodes[id-1]
}
func (d *Document) Raw(id ID) string {
	n := d.Node(id)
	if n == nil {
		return ""
	}
	a, b := n.Origin.Span.Start.Offset, n.Origin.Span.End.Offset
	if a < 0 || b < a || b > len(d.source) {
		return ""
	}
	return d.source[a:b]
}
func cloneState(s State) State       { s.Indents = append([]string(nil), s.Indents...); return s }
func origin(n *groff.ASTNode) Origin { return Origin{ASTIDs: []groff.NodeID{n.ID()}, Span: n.Span()} }

// Validate checks links, identity, source ranges, depth and reachability without
// recursive traversal. A model unmarshaled from JSON has no original Source.
func (d *Document) Validate() error {
	if d.Version != 1 || len(d.Nodes) == 0 || len(d.Nodes) > 100001 || d.Nodes[0].Kind != DocumentNode || d.Nodes[0].Parent != 0 {
		return fmt.Errorf("invalid semantic document root/version/budget")
	}
	if d.MetadataOrigin != nil {
		s := d.MetadataOrigin.Span
		if s.Start.Offset < 0 || s.End.Offset < s.Start.Offset || s.End.Offset > d.Nodes[0].Origin.Span.End.Offset {
			return fmt.Errorf("invalid metadata source span")
		}
	}
	seen := make([]bool, len(d.Nodes))
	stack := []struct {
		id    ID
		depth int
	}{{1, 0}}
	for len(stack) > 0 {
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if entry.id == 0 || entry.id > ID(len(d.Nodes)) || entry.depth > 512 {
			return fmt.Errorf("invalid semantic link/depth")
		}
		index := int(entry.id) - 1
		n := &d.Nodes[index]
		if seen[index] || n.ID != entry.id {
			return fmt.Errorf("cycle, repeated child or invalid semantic identity")
		}
		seen[index] = true
		switch n.Kind {
		case DocumentNode, Section, Subsection, Paragraph, DefinitionList, Item, Indent, Preformatted, Space, Unsupported:
		default:
			return fmt.Errorf("unknown semantic kind %q", n.Kind)
		}
		a, b := n.Origin.Span.Start.Offset, n.Origin.Span.End.Offset
		if a < 0 || b < a || (d.source != "" && b > len(d.source)) {
			return fmt.Errorf("invalid semantic source span")
		}
		for _, runs := range [][]Run{n.Runs, n.Tag} {
			for _, run := range runs {
				rs, re := run.Origin.Span.Start.Offset, run.Origin.Span.End.Offset
				if rs < a || re < rs || re > b {
					return fmt.Errorf("invalid semantic run source span")
				}
			}
		}
		for _, child := range n.Children {
			c := d.Node(child)
			if c == nil || c.Parent != n.ID || c.Origin.Span.Start.Offset < a || c.Origin.Span.End.Offset > b {
				return fmt.Errorf("inconsistent semantic parent/child")
			}
			stack = append(stack, struct {
				id    ID
				depth int
			}{child, entry.depth + 1})
		}
	}
	for _, v := range seen {
		if !v {
			return fmt.Errorf("unreachable semantic node")
		}
	}
	return nil
}

// TextOptions chooses exact source bytes or derived semantic text. Normalize
// folds whitespace only in semantic mode; Raw takes precedence over Normalize.
type TextOptions struct{ Raw, Normalize bool }

func (d *Document) Text(id ID, o TextOptions) string {
	if o.Raw {
		return d.Raw(id)
	}
	selection, err := d.SelectNode(id)
	if err != nil {
		return ""
	}
	return selection.Text(o)
}
func textRuns(rs []Run) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(r.Text)
	}
	return b.String()
}
func normalize(s string) string { return strings.Join(strings.Fields(s), " ") }
