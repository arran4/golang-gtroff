package convert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/man"
)

// Semantics is the cross-format comparison view. It omits source identities and
// interpreter state while retaining visible structure, styled runs, links,
// references, unsupported content and explicit spacing. A zero-width indent is
// transparent. Flow ASCII whitespace folds; preformatted text remains exact.
type Semantics struct {
	Metadata man.Metadata   `json:"metadata"`
	Children []SemanticNode `json:"children,omitempty"`
}
type SemanticNode struct {
	Kind              man.Kind       `json:"kind"`
	Value             string         `json:"value,omitempty"`
	ParagraphDistance string         `json:"paragraphDistance,omitempty"`
	Runs              []SemanticRun  `json:"runs,omitempty"`
	Tag               []SemanticRun  `json:"tag,omitempty"`
	Children          []SemanticNode `json:"children,omitempty"`
}
type SemanticRun struct {
	Text        string    `json:"text"`
	Style       man.Style `json:"style,omitempty"`
	Link        string    `json:"link,omitempty"`
	Reference   string    `json:"reference,omitempty"`
	Break       bool      `json:"break,omitempty"`
	Unsupported bool      `json:"unsupported,omitempty"`
}

func Normalize(d *man.Document) (Semantics, error) {
	if d == nil {
		return Semantics{}, fmt.Errorf("nil semantic document")
	}
	if err := d.Validate(); err != nil {
		return Semantics{}, err
	}
	var visit func(man.ID) []SemanticNode
	visit = func(id man.ID) []SemanticNode {
		n := d.Node(id)
		out := SemanticNode{Kind: n.Kind}
		for _, c := range n.Children {
			out.Children = append(out.Children, visit(c)...)
		}
		if n.Kind == man.Indent && dimension(n.Value, "4") == "0em" {
			return out.Children
		}
		switch n.Kind {
		case man.Indent:
			out.Value = dimension(n.Value, "4")
		case man.Item:
			out.Value = dimension(n.Value, "4")
		case man.Space:
			out.Value = n.Value
			if out.Value == "" {
				out.Value = "1v"
			}
		case man.Unsupported:
			out.Value = n.Value
		}
		if n.Kind == man.Paragraph {
			out.ParagraphDistance = n.State.ParagraphDistance
		}
		heading := n.Kind == man.Section || n.Kind == man.Subsection
		rs := n.Runs
		if heading && len(rs) == 0 {
			rs = []man.Run{{Text: n.Heading}}
		}
		for _, r := range normalizeRuns(rs, n.Kind == man.Preformatted || n.Kind == man.Unsupported, heading) {
			out.Runs = append(out.Runs, semanticRun(r))
		}
		for _, r := range normalizeRuns(n.Tag, false, false) {
			out.Tag = append(out.Tag, semanticRun(r))
		}
		return []SemanticNode{out}
	}
	out := Semantics{Metadata: d.Metadata}
	for _, id := range d.Nodes[0].Children {
		out.Children = append(out.Children, visit(id)...)
	}
	return out, nil
}
func semanticRun(r man.Run) SemanticRun {
	return SemanticRun{r.Text, r.Style, r.Link, r.Reference, r.Break, r.Unsupported}
}
func sameRun(a, b man.Run) bool {
	return !a.Break && !b.Break && a.Style == b.Style && a.Link == b.Link && a.Reference == b.Reference && a.Break == b.Break && a.Unsupported == b.Unsupported
}
func normalizeRuns(rs []man.Run, pre, heading bool) []man.Run {
	if !pre {
		var expanded []man.Run
		for _, r := range rs {
			if !r.Break {
				expanded = append(expanded, r)
				continue
			}
			parts := strings.Split(r.Text, "\n")
			for i, p := range parts {
				if i > 0 {
					expanded = append(expanded, man.Run{Text: "\n", Break: true, Unsupported: r.Unsupported, Origin: r.Origin})
				}
				if p != "" {
					text := r
					text.Text = p
					text.Break = false
					expanded = append(expanded, text)
				}
			}
		}
		rs = expanded
	}
	var out []man.Run
	var text strings.Builder
	var current man.Run
	flush := func() {
		if text.Len() > 0 {
			current.Text = text.String()
			out = append(out, current)
			text = strings.Builder{}
		}
	}
	space := false
	for _, r := range rs {
		if heading {
			r.Style.Bold = true
		}
		if pre {
			r.Style.Mono = true
			r.Break = false
		} else if r.Break {
			r.Text = "\n"
			space = false
		} else {
			var s strings.Builder
			for _, c := range flow(r.Text) {
				if c == ' ' {
					if space {
						continue
					}
					space = true
				} else {
					space = false
				}
				s.WriteRune(c)
			}
			r.Text = s.String()
		}
		if r.Text == "" {
			continue
		}
		if text.Len() > 0 && !sameRun(current, r) {
			flush()
		}
		if text.Len() == 0 {
			current = r
		}
		text.WriteString(r.Text)
	}
	flush()
	if !pre {
		for len(out) > 0 {
			out[0].Text = strings.TrimLeft(out[0].Text, " ")
			if out[0].Text != "" {
				break
			}
			out = out[1:]
		}
		for len(out) > 0 {
			last := len(out) - 1
			out[last].Text = strings.TrimRight(out[last].Text, " ")
			if out[last].Text != "" {
				break
			}
			out = out[:last]
		}
	}
	return out
}

var dimensionPattern = regexp.MustCompile(`^([+]?[0-9]+(?:\.[0-9]+)?)(n|m|em)?$`)

func dimension(s, fallback string) string {
	if s == "" {
		s = fallback
	}
	m := dimensionPattern.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return s
	}
	if m[2] == "" || m[2] == "n" {
		v /= 2
	}
	return strconv.FormatFloat(v, 'f', -1, 64) + "em"
}
