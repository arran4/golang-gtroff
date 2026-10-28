package man

import (
	"encoding/json"
	"fmt"
	"strings"

	groff "github.com/arran4/golang-gtroff"
)

// Selection references semantic nodes in document order and carries the upstream
// state for each requested root. The original document remains the source owner.
type Selection struct {
	Metadata Metadata `json:"metadata"`
	Roots    []ID     `json:"roots"`
	Contexts []State  `json:"contexts"`
	Nodes    []Node   `json:"nodes"`
	doc      *Document
}

func (d *Document) selectIDs(roots []ID, descend bool) (*Selection, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("semantic selection matched no nodes")
	}
	s := &Selection{Metadata: d.Metadata, Roots: append([]ID(nil), roots...), doc: d}
	seen := map[ID]bool{}
	for _, id := range roots {
		n := d.Node(id)
		if n == nil {
			return nil, fmt.Errorf("semantic node %d not found", id)
		}
		s.Contexts = append(s.Contexts, cloneState(n.State))
		stack := []ID{id}
		for len(stack) > 0 {
			next := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[next] {
				continue
			}
			seen[next] = true
			if descend {
				stack = append(stack, d.Node(next).Children...)
			}
		}
	}
	for _, n := range d.Nodes {
		if seen[n.ID] {
			s.Nodes = append(s.Nodes, n)
		}
	}
	return s, nil
}
func (d *Document) SelectNode(id ID) (*Selection, error) { return d.selectIDs([]ID{id}, true) }

// SelectRange selects inclusive semantic IDs and their descendants. It preserves
// complete semantic blocks (a flowing paragraph can cover several source lines).
func (d *Document) SelectRange(first, last ID) (*Selection, error) {
	if first == 0 || last < first || last > ID(len(d.Nodes)) {
		return nil, fmt.Errorf("invalid semantic node range")
	}
	roots := make([]ID, 0, int(last-first+1))
	for id := first; id <= last; id++ {
		roots = append(roots, id)
	}
	return d.selectIDs(roots, true)
}
func (d *Document) SelectSection(heading string, normalized bool) (*Selection, error) {
	var ids []ID
	if normalized {
		heading = strings.ToLower(normalize(heading))
	}
	for _, n := range d.Nodes {
		if n.Kind != Section && n.Kind != Subsection {
			continue
		}
		s := n.Heading
		if normalized {
			s = strings.ToLower(normalize(s))
		}
		if s == heading {
			ids = append(ids, n.ID)
		}
	}
	return d.selectIDs(ids, true)
}

// SelectAST selects the smallest semantic block explicitly originating at an AST
// ID. Section container IDs and heading-line IDs both resolve to the section.
func (d *Document) SelectAST(id groff.NodeID) (*Selection, error) {
	var chosen ID
	for _, n := range d.Nodes {
		for _, astID := range n.Origin.ASTIDs {
			if astID == id {
				chosen = n.ID
			}
		}
	}
	if chosen == 0 {
		return nil, fmt.Errorf("AST node %d has no semantic content", id)
	}
	return d.SelectNode(chosen)
}

// FindTerm returns content-bearing nodes in deterministic semantic order.
func (d *Document) FindTerm(term string, normalized bool) []ID {
	if normalized {
		term = strings.ToLower(normalize(term))
	}
	var ids []ID
	for _, n := range d.Nodes {
		if len(n.Runs) == 0 && len(n.Tag) == 0 {
			continue
		}
		s := textRuns(n.Tag) + " " + textRuns(n.Runs)
		if normalized {
			s = strings.ToLower(normalize(s))
		}
		if strings.Contains(s, term) {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

// SelectTerm extracts complete matching semantic blocks with their context.
func (d *Document) SelectTerm(term string, normalized bool) (*Selection, error) {
	return d.selectIDs(d.FindTerm(term, normalized), true)
}

func (s *Selection) Text(o TextOptions) string {
	if o.Raw {
		if s.doc == nil {
			return ""
		}
		// Merge overlapping source spans without including unselected gaps.
		type bounds struct{ start, end int }
		var ranges []bounds
		for _, n := range s.Nodes {
			a, b := n.Origin.Span.Start.Offset, n.Origin.Span.End.Offset
			if a < 0 || b < a || b > len(s.doc.source) {
				continue
			}
			if len(ranges) > 0 && a <= ranges[len(ranges)-1].end {
				if b > ranges[len(ranges)-1].end {
					ranges[len(ranges)-1].end = b
				}
			} else {
				ranges = append(ranges, bounds{a, b})
			}
		}
		var out strings.Builder
		for _, r := range ranges {
			out.WriteString(s.doc.source[r.start:r.end])
		}
		return out.String()
	}
	var out strings.Builder
	for _, n := range s.Nodes {
		switch n.Kind {
		case Section, Subsection:
			out.WriteString(n.Heading)
			out.WriteByte('\n')
		case Item:
			if tag := textRuns(n.Tag); tag != "" {
				out.WriteString(tag)
				out.WriteByte('\n')
			}
		case Paragraph, Preformatted, Unsupported:
			out.WriteString(textRuns(n.Runs))
			if len(n.Runs) > 0 && !strings.HasSuffix(textRuns(n.Runs), "\n") {
				out.WriteByte('\n')
			}
		case Space:
			out.WriteByte('\n')
		}
	}
	text := out.String()
	if o.Normalize {
		text = normalize(text)
	}
	return text
}

// Canonical is a deterministic semantic JSON projection. It omits source
// identity/diagnostics, merges adjacent equal runs without rewriting literal text.
// It is suitable for model JSON round-trips, not a byte-exact roff writer.
func (d *Document) Canonical() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	type projectedRun struct {
		Text        string `json:"text"`
		Style       Style  `json:"style"`
		Link        string `json:"link,omitempty"`
		Reference   string `json:"reference,omitempty"`
		Break       bool   `json:"break,omitempty"`
		Unsupported bool   `json:"unsupported,omitempty"`
	}
	type projectedNode struct {
		Kind    Kind           `json:"kind"`
		Parent  ID             `json:"parent,omitempty"`
		Heading string         `json:"heading,omitempty"`
		Value   string         `json:"value,omitempty"`
		Runs    []projectedRun `json:"runs,omitempty"`
		Tag     []projectedRun `json:"tag,omitempty"`
		State   State          `json:"state"`
	}
	project := func(runs []Run) []projectedRun {
		var out []projectedRun
		var current projectedRun
		var text strings.Builder
		active := false
		flush := func() {
			if active {
				current.Text = text.String()
				out = append(out, current)
				text.Reset()
			}
		}
		for _, r := range runs {
			v := projectedRun{Style: r.Style, Link: r.Link, Reference: r.Reference, Break: r.Break, Unsupported: r.Unsupported}
			if active && (current.Style != v.Style || current.Link != v.Link || current.Reference != v.Reference || current.Break != v.Break || current.Unsupported != v.Unsupported) {
				flush()
				active = false
			}
			if !active {
				current = v
				active = true
			}
			text.WriteString(r.Text)
		}
		flush()
		return out
	}
	nodes := make([]projectedNode, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = projectedNode{Kind: n.Kind, Parent: n.Parent, Heading: n.Heading, Value: n.Value, Runs: project(n.Runs), Tag: project(n.Tag), State: cloneState(n.State)}
	}
	return json.Marshal(struct {
		Metadata Metadata        `json:"metadata"`
		Nodes    []projectedNode `json:"nodes"`
	}{d.Metadata, nodes})
}
