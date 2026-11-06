package lint

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

type scope struct {
	key    string
	span   groff.Span
	parent int
}

func makeContext(ctx context.Context, document Document, index int, policy Policy, meter *budget, maxNodes int) (*Context, error) {
	c := &Context{document: document, index: index, models: map[source.ID]*man.Document{}, scopes: map[source.ID][]scope{}, edges: map[source.EdgeID]source.Edge{}, includeStarts: map[source.ID][]int{}, policy: policy}
	if document.Sources == nil {
		return c, fmt.Errorf("document %q has no source snapshot", document.Name)
	}
	if err := meter.spend(1); err != nil {
		return c, err
	}
	if meter.remaining < 1 {
		return c, ErrLimit
	}
	analysis, err := analyze.Run(ctx, document.Sources, analyze.Options{MaxNodes: maxNodes, MaxWork: meter.remaining, MaxFindings: maxNodes * 3})
	c.analysis = analysis
	if chargeErr := meter.spend(analysis.WorkUnits); chargeErr != nil {
		return c, chargeErr
	}
	if err != nil {
		if errors.Is(err, analyze.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return c, err
	}
	c.manScope = analysis.Profile.Name != "ms" && analysis.Profile.Name != "mdoc" && analysis.Profile.Name != "mixed"
	c.holes = analysis.Flags.HasPartialSource || analysis.Flags.RequiresEvaluation || analysis.Flags.HasUnknown
	if document.InputError != nil {
		c.holes = true
	}
	for _, edge := range document.Sources.Edges() {
		c.edges[edge.ID] = edge
		c.includeStarts[edge.From] = append(c.includeStarts[edge.From], edge.Span.Start.Offset)
		if edge.State != "resolved" {
			c.holes = true
		}
	}
	for id := range c.includeStarts {
		sort.Ints(c.includeStarts[id])
	}
	for _, diagnostic := range analysis.Diagnostics {
		if diagnostic.Severity == "error" {
			c.holes = true
		}
	}
	if !c.manScope {
		return c, nil
	}
	for _, file := range document.Sources.Files() {
		bytes, nodes := len(file.AST().Source()), 0
		var countErr error
		file.AST().Root().Walk(func(*groff.ASTNode) groff.WalkAction {
			nodes++
			if err := meter.spend(1); err != nil {
				countErr = err
				return groff.WalkStop
			}
			return groff.WalkContinue
		})
		if countErr != nil {
			return c, countErr
		}
		if err := meter.spend(bytes + nodes); err != nil {
			return c, err
		}
		model, err := man.Build(file.AST(), man.Options{MaxNodes: maxNodes, MaxBytes: bytes + 1})
		if err != nil {
			if errors.Is(err, man.ErrLimit) {
				err = errors.Join(ErrLimit, err)
			}
			return c, err
		}
		c.models[file.ID()] = model
		if c.title == "" && model.Metadata.Title != "" {
			c.title = model.Metadata.Title
			c.manualSection = model.Metadata.Section
		}
		scopeByID := map[man.ID]int{}
		for _, node := range model.Nodes {
			if node.Kind != man.Section && node.Kind != man.Subsection && node.Kind != man.Indent {
				continue
			}
			parent := -1
			for id := node.Parent; id != 0; id = model.Node(id).Parent {
				if value, ok := scopeByID[id]; ok {
					parent = value
					break
				}
			}
			key := fmt.Sprintf("source:%d/%s:%d", file.ID(), node.Kind, node.Origin.Span.Start.Offset)
			scopeByID[node.ID] = len(c.scopes[file.ID()])
			c.scopes[file.ID()] = append(c.scopes[file.ID()], scope{key, node.Origin.Span, parent})
		}
		for _, node := range model.Nodes {
			if node.Kind != man.Section && node.Kind != man.Subsection && node.Kind != man.Item {
				continue
			}
			text, unsupported, err := subtree(model, node.ID, meter)
			if err != nil {
				return c, err
			}
			if node.Kind == man.Section || node.Kind == man.Subsection {
				c.sections = append(c.sections, Section{Kind: node.Kind, Source: file.ID(), ID: node.ID, Span: node.Origin.Span, Heading: node.Heading, Text: text, Unsupported: unsupported, ASTIDs: append([]groff.NodeID(nil), node.Origin.ASTIDs...)})
			} else {
				term := runsText(node.Tag)
				simple := true
				tagLine := 0
				for _, run := range node.Tag {
					if tagLine == 0 {
						tagLine = run.Origin.Span.Start.Line
					} else if tagLine != run.Origin.Span.Start.Line {
						simple = false
					}
				}
				expected := false
				for _, id := range node.Origin.ASTIDs {
					ast := file.AST().Node(id)
					if ast != nil && ast.Kind() == groff.ASTLine && ast.Span().Start.Offset == node.Origin.Span.Start.Offset {
						expected = ast.Attributes().Name == ".TP" || ast.Attributes().Name == ".TQ"
						break
					}
				}
				c.definitions = append(c.definitions, Definition{Source: file.ID(), ID: node.ID, Span: node.Origin.Span, Term: term, Text: text, ExpectedTerm: expected, SimpleTerm: simple, Unsupported: unsupported, ASTIDs: append([]groff.NodeID(nil), node.Origin.ASTIDs...)})
			}
		}
	}
	// Resolve fragment scopes after every source's local scopes exist, including
	// descendants loaded before later parent-file nodes in the logical sequence.
	for i := range c.definitions {
		definition := &c.definitions[i]
		definition.Scope = c.scopeAt(definition.Source, definition.Span.Start.Offset)
	}
	return c, nil
}
func subtree(model *man.Document, id man.ID, meter *budget) (string, bool, error) {
	var nodes []man.Node
	unsupported := false
	stack := []man.ID{id}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := meter.spend(1); err != nil {
			return "", false, err
		}
		node := model.Node(current)
		nodes = append(nodes, *node)
		if node.Kind == man.Unsupported {
			unsupported = true
		}
		for _, runs := range [][]man.Run{node.Runs, node.Tag} {
			for _, run := range runs {
				if err := meter.spend(len(run.Text)); err != nil {
					return "", false, err
				}
				unsupported = unsupported || run.Unsupported
			}
		}
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	// Omit the root heading/tag from section/definition body checks, while keeping
	// nested content and unsupported source visible as uncertainty.
	content := nodes
	if len(content) > 0 {
		content[0].Heading = ""
		content[0].Runs = nil
		content[0].Tag = nil
	}
	selection := man.Selection{Nodes: content}
	text := selection.Text(man.TextOptions{})
	return text, unsupported, nil
}
func (c *Context) scopeAt(id source.ID, offset int) string {
	for hops := 0; hops < 129; hops++ {
		scopes := c.scopes[id]
		at := sort.Search(len(scopes), func(i int) bool { return scopes[i].span.Start.Offset > offset }) - 1
		for at >= 0 {
			if offset < scopes[at].span.End.Offset {
				return scopes[at].key
			}
			at = scopes[at].parent
		}
		file := c.document.Sources.File(id)
		if file == nil || file.ParentEdge() == 0 {
			break
		}
		edge, ok := c.edges[file.ParentEdge()]
		if !ok {
			break
		}
		id = edge.From
		offset = edge.Span.Start.Offset
	}
	return "document:" + strconv.Itoa(c.index)
}
func runsText(runs []man.Run) string {
	var text strings.Builder
	for _, run := range runs {
		text.WriteString(run.Text)
	}
	return text.String()
}
func normalizeHeading(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}
func normalizeTerm(value string) string { return strings.Join(strings.Fields(value), " ") }
func rootFinding(c *Context, message string) Finding {
	sourceID := source.ID(1)
	file := c.document.Sources.File(sourceID)
	finding := Finding{Source: sourceID, Message: message}
	if file != nil {
		position := file.AST().Root().Span().Start
		finding.Node = file.AST().Root().ID()
		finding.Span = groff.Span{Start: position, End: position}
	}
	return finding
}
