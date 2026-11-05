package man

import (
	"fmt"
	"sort"

	groff "github.com/arran4/golang-gtroff"
)

// Syntax returns the original immutable source AST supplied to Build. The
// semantic hierarchy is an optional interpretation of this snapshot, not a
// replacement for it. A Document decoded from JSON has no source AST and returns
// nil. Mutating semantic fields does not change the syntax tree.
func (d *Document) Syntax() *groff.AST { return d.syntax }

// SourceNodes returns the original physical AST lines intersecting a semantic
// node's source span, in byte order. Unlike Origin.ASTIDs (which identifies
// semantic contributors), this includes comments, blank lines, delimiters and
// opaque material within the span. Each pointer and NodeID belongs to Syntax().
// Inline nodes remain accessible through each physical line's Children.
//
// The model is validated before its caller-owned spans are used. Invalid IDs,
// invalid models and JSON documents without an original syntax snapshot return
// errors. An empty span yields no lines. This method neither reparses source nor
// resolves includes; it cannot prove the original ParseAST call succeeded.
func (d *Document) SourceNodes(id ID) ([]*groff.ASTNode, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	n := d.Node(id)
	if n == nil {
		return nil, fmt.Errorf("semantic node %d not found", id)
	}
	if d.syntax == nil {
		return nil, fmt.Errorf("semantic document has no original syntax snapshot")
	}
	start, end := n.Origin.Span.Start.Offset, n.Origin.Span.End.Offset
	if start == end {
		return nil, nil
	}
	first := sort.Search(len(d.syntaxLines), func(i int) bool {
		return d.syntaxLines[i].Span().End.Offset > start
	})
	last := sort.Search(len(d.syntaxLines), func(i int) bool {
		return d.syntaxLines[i].Span().Start.Offset >= end
	})
	return append([]*groff.ASTNode(nil), d.syntaxLines[first:last]...), nil
}
