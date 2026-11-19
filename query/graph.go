package query

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

// View explicitly chooses physical syntax or the optional man projection.
type View string

const (
	Syntax   View = "syntax"
	Semantic View = "semantic"
)

// Options bounds one execution, including graph construction and output. MaxNodes
// bounds AST preflight per source and the complete projected graph. Zero
// or negative budgets use defaults: 200,000 projected nodes, 64 million work
// units, 10,000 matches and 16 MiB of materialized match data. Work counts node,
// predicate and source/text-byte processing; it is not a wall-clock promise.
type Options struct {
	View                                          View
	MaxNodes, MaxWork, MaxMatches, MaxResultBytes int
}

var ErrLimit = errors.New("query resource limit exceeded")

type meter struct {
	ctx       context.Context
	remaining int
}

func (m *meter) spend(n int) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > m.remaining {
		return ErrLimit
	}
	m.remaining -= n
	return nil
}

type graphNode struct {
	order    int // source-relative child position in the parent projection
	kind     Kind
	parent   int
	children []int
	file     *source.File
	syntax   *groff.ASTNode
	model    *man.Document
	semantic man.ID
	term     bool
	span     groff.Span
	astIDs   []groff.NodeID
	identity string
}
type graph struct {
	set          *source.Set
	nodes        []graphNode
	roots        []int
	files        map[source.ID]int
	syntaxOwners map[source.ID]map[groff.NodeID]int
	sourceInfo   map[source.ID]SourceInfo
	diagnostics  []source.Diagnostic
	bySource     map[source.ID][]source.Diagnostic
}

// SourceInfo is the original source occurrence, not a relocated synthetic file.
// Digest is SHA-256 of exact captured bytes. Complete records observed EOF only.
type SourceInfo struct {
	ID         source.ID     `json:"id"`
	Name       string        `json:"name"`
	ParentEdge source.EdgeID `json:"parentEdge,omitempty"`
	Depth      int           `json:"depth"`
	Digest     string        `json:"digest"`
	Complete   bool          `json:"complete"`
}

func buildGraph(set *source.Set, o Options, m *meter) (*graph, error) {
	g := &graph{set: set, files: map[source.ID]int{}, syntaxOwners: map[source.ID]map[groff.NodeID]int{}, sourceInfo: map[source.ID]SourceInfo{}, bySource: map[source.ID][]source.Diagnostic{}}
	if set == nil {
		return g, fmt.Errorf("nil source set")
	}
	g.diagnostics = set.Diagnostics()
	initialBySource := map[source.ID][]source.Diagnostic{}
	for _, diagnostic := range g.diagnostics {
		initialBySource[diagnostic.Source] = append(initialBySource[diagnostic.Source], diagnostic)
	}
	add := func(node graphNode) (int, error) {
		if err := m.spend(1); err != nil {
			return 0, err
		}
		if len(g.nodes) >= o.MaxNodes {
			return 0, ErrLimit
		}
		index := len(g.nodes)
		node.order = node.span.Start.Offset
		g.nodes = append(g.nodes, node)
		if node.parent >= 0 {
			g.nodes[node.parent].children = append(g.nodes[node.parent].children, index)
		}
		return index, nil
	}
	for _, file := range set.Files() {
		tree := file.AST()
		if err := m.spend(len(file.Name())); err != nil {
			return g, err
		}
		if err := m.spend(len(tree.Source())); err != nil {
			return g, err
		}
		digest := sha256.Sum256([]byte(tree.Source()))
		g.sourceInfo[file.ID()] = SourceInfo{file.ID(), file.Name(), file.ParentEdge(), file.Depth(), fmt.Sprintf("%x", digest), tree.SourceComplete()}
		preflight := 0
		var boundsErr error
		tree.Root().Walk(func(*groff.ASTNode) groff.WalkAction {
			preflight++
			if preflight > o.MaxNodes {
				boundsErr = ErrLimit
				return groff.WalkStop
			}
			if err := m.spend(1); err != nil {
				boundsErr = err
				return groff.WalkStop
			}
			return groff.WalkContinue
		})
		if boundsErr != nil {
			return g, boundsErr
		}
		wrapper, err := add(graphNode{kind: Source, parent: -1, file: file, span: tree.Root().Span(), syntax: tree.Root(), identity: fmt.Sprintf("source:%d", file.ID())})
		if err != nil {
			return g, err
		}
		g.files[file.ID()] = wrapper
		owners := map[groff.NodeID]int{}
		g.syntaxOwners[file.ID()] = owners
		if o.View == Syntax {
			var walkErr error
			tree.Root().Walk(func(n *groff.ASTNode) groff.WalkAction {
				parent := wrapper
				if n.Parent() != nil {
					parent = owners[n.Parent().ID()]
				}
				index, err := add(graphNode{kind: Kind(n.Kind()), parent: parent, file: file, syntax: n, span: n.Span(), astIDs: []groff.NodeID{n.ID()}, identity: fmt.Sprintf("source:%d/syntax:%d", file.ID(), n.ID())})
				if err != nil {
					walkErr = err
					return groff.WalkStop
				}
				owners[n.ID()] = index
				return groff.WalkContinue
			})
			if walkErr != nil {
				return g, walkErr
			}
			continue
		}
		model, err := man.Build(tree, man.Options{MaxNodes: o.MaxNodes, MaxBytes: len(tree.Source()) + 1})
		if model != nil {
			// Parser diagnostics already have source classification. Add only new
			// semantic findings, retaining include diagnostics as separate facts.
			seen := map[string]bool{}
			key := func(d groff.Diagnostic) string {
				return strconv.Itoa(d.Position.Offset) + ":" + d.Severity + ":" + d.Message
			}
			for _, d := range initialBySource[file.ID()] {
				seen[key(d.Diagnostic)] = true
			}
			for _, d := range model.Diagnostics {
				if !seen[key(d)] {
					g.diagnostics = append(g.diagnostics, source.Diagnostic{Source: file.ID(), Code: "semantic", Diagnostic: d})
					seen[key(d)] = true
				}
			}
		}
		if err != nil {
			if errors.Is(err, man.ErrLimit) {
				err = errors.Join(ErrLimit, err)
			}
			return g, fmt.Errorf("project %s: %w", file.Name(), err)
		}
		indices := map[man.ID]int{}
		for _, n := range model.Nodes {
			parent := wrapper
			if n.Parent != 0 {
				parent = indices[n.Parent]
			}
			kind := Kind(n.Kind)
			if n.Kind == man.Item {
				kind = Definition
			}
			var syntax *groff.ASTNode
			if len(n.Origin.ASTIDs) > 0 {
				syntax = tree.Node(n.Origin.ASTIDs[0])
			}
			index, err := add(graphNode{kind: kind, parent: parent, file: file, syntax: syntax, model: model, semantic: n.ID, span: n.Origin.Span, astIDs: append([]groff.NodeID(nil), n.Origin.ASTIDs...), identity: fmt.Sprintf("source:%d/semantic:%d", file.ID(), n.ID)})
			if err != nil {
				return g, err
			}
			indices[n.ID] = index
			// The last explicit contributor is the smallest owner, matching SelectAST.
			for _, id := range n.Origin.ASTIDs {
				owners[id] = index
			}
			if kind == Definition && len(n.Tag) > 0 {
				span := n.Tag[0].Origin.Span
				for _, run := range n.Tag {
					if run.Origin.Span.Start.Offset < span.Start.Offset {
						span.Start = run.Origin.Span.Start
					}
					if run.Origin.Span.End.Offset > span.End.Offset {
						span.End = run.Origin.Span.End
					}
				}
				tagIDs := []groff.NodeID{}
				seenIDs := map[groff.NodeID]bool{}
				for _, run := range n.Tag {
					for _, id := range run.Origin.ASTIDs {
						if !seenIDs[id] {
							tagIDs = append(tagIDs, id)
							seenIDs[id] = true
						}
					}
				}
				tagSyntax := syntax
				if len(tagIDs) > 0 {
					tagSyntax = tree.Node(tagIDs[0])
				}
				_, err = add(graphNode{kind: Term, parent: index, file: file, syntax: tagSyntax, model: model, semantic: n.ID, term: true, span: span, astIDs: tagIDs, identity: fmt.Sprintf("source:%d/semantic:%d/term", file.ID(), n.ID)})
				if err != nil {
					return g, err
				}
			}
		}
	}
	edges := set.Edges()
	for _, file := range set.Files() {
		wrapper := g.files[file.ID()]
		if file.ParentEdge() == 0 {
			g.roots = append(g.roots, wrapper)
			continue
		}
		var edge *source.Edge
		if file.ParentEdge() > 0 && file.ParentEdge() <= source.EdgeID(len(edges)) {
			edge = &edges[file.ParentEdge()-1]
		}
		if edge == nil {
			return g, fmt.Errorf("source occurrence has no parent edge")
		}
		owner, ok := g.syntaxOwners[edge.From][edge.Node]
		if !ok {
			return g, fmt.Errorf("include directive has no query owner")
		}
		g.nodes[wrapper].parent = owner
		g.nodes[wrapper].order = edge.Span.Start.Offset
		g.nodes[owner].children = append(g.nodes[owner].children, wrapper)
	}
	if o.View == Semantic {
		// A directive can belong to an item/tag or preformatted context that also
		// owns later content. Place its source occurrence at the directive position,
		// after equal-position existing children, rather than at the owner's end.
		for i := range g.nodes {
			children := g.nodes[i].children
			if len(children) < 2 {
				continue
			}
			if err := m.spend(len(children) * bits.Len(uint(len(children)))); err != nil {
				return g, err
			}
			sort.SliceStable(children, func(a, b int) bool { return g.nodes[children[a]].order < g.nodes[children[b]].order })
		}
	}
	sort.SliceStable(g.diagnostics, func(i, j int) bool {
		a, b := g.diagnostics[i], g.diagnostics[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Position.Offset < b.Position.Offset
	})
	for _, d := range g.diagnostics {
		g.bySource[d.Source] = append(g.bySource[d.Source], d)
	}
	return g, nil
}

func (n *graphNode) attributes() groff.ASTAttributes {
	if n.syntax == nil {
		return groff.ASTAttributes{}
	}
	syntax := n.syntax
	if syntax.Kind() != groff.ASTLine && syntax.Kind() != groff.ASTInline {
		next := n.file.AST().Node(syntax.ID() + 1)
		if next != nil && next.Kind() == groff.ASTLine && next.Span().Start.Offset == syntax.Span().Start.Offset {
			syntax = next
		}
	}
	return syntax.Attributes()
}
func kindMatches(n *graphNode, kind Kind) bool {
	if kind == Any || kind == n.kind {
		return true
	}
	if n.kind != Line {
		return false
	}
	token := n.attributes().TokenKind
	return kind == Macro && token == groff.KindMacro || kind == Request && token == groff.KindRequest || kind == Comment && token == groff.KindComment || kind == Unknown && token == groff.KindUnknown
}
func runsText(runs []man.Run) string {
	var out strings.Builder
	for _, run := range runs {
		out.WriteString(run.Text)
	}
	return out.String()
}
func normalize(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
func (n *graphNode) termText() string {
	if n.model == nil || n.semantic == 0 {
		return ""
	}
	return runsText(n.model.Node(n.semantic).Tag)
}
func (n *graphNode) text(m *meter) (string, error) {
	if n.term {
		text := n.termText()
		return text, m.spend(len(text))
	}
	if n.model == nil {
		if n.syntax == nil {
			return "", nil
		}
		var out strings.Builder
		var err error
		n.syntax.Walk(func(p *groff.ASTNode) groff.WalkAction {
			if err = m.spend(1); err != nil {
				return groff.WalkStop
			}
			if p.Kind() == groff.ASTLine || p.Kind() == groff.ASTInline {
				text := p.Text()
				if err = m.spend(len(text)); err != nil {
					return groff.WalkStop
				}
				out.WriteString(text)
				return groff.WalkPrune
			}
			return groff.WalkContinue
		})
		return out.String(), err
	}
	var selected []man.Node
	stack := []man.ID{n.semantic}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := m.spend(1); err != nil {
			return "", err
		}
		node := n.model.Node(id)
		size := len(node.Heading)
		for _, run := range node.Runs {
			size += len(run.Text)
		}
		for _, run := range node.Tag {
			size += len(run.Text)
		}
		if err := m.spend(size); err != nil {
			return "", err
		}
		selected = append(selected, *node)
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	selection := man.Selection{Nodes: selected}
	return selection.Text(man.TextOptions{}), nil
}
