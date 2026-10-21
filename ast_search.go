package groff

import "strings"

// WalkAction controls an iterative, deterministic preorder walk.
type WalkAction uint8

const (
	WalkContinue WalkAction = iota
	WalkPrune
	WalkStop
)

// Walk visits n and its descendants. Prune skips the current subtree; Stop ends
// the walk. Immutable links make cycles impossible through the public API.
func (n *ASTNode) Walk(visit func(*ASTNode) WalkAction) {
	for i := int(n.id) - 1; i < n.limit; {
		p := n.tree.nodes[i]
		switch visit(p) {
		case WalkStop:
			return
		case WalkPrune:
			i = p.limit
		default:
			i++
		}
	}
}

// Find searches n and descendants in document order. A nil predicate matches all.
func (n *ASTNode) Find(match func(*ASTNode) bool) []*ASTNode {
	var out []*ASTNode
	n.Walk(func(p *ASTNode) WalkAction {
		if match == nil || match(p) {
			out = append(out, p)
		}
		return WalkContinue
	})
	return out
}
func (n *ASTNode) FindKind(kind ASTKind) []*ASTNode {
	return n.Find(func(p *ASTNode) bool { return p.kind == kind })
}

// FindMacro matches physical control lines, including requests/unknown controls.
func (n *ASTNode) FindMacro(name string) []*ASTNode {
	if !strings.HasPrefix(name, ".") {
		name = "." + name
	}
	return n.Find(func(p *ASTNode) bool { return p.kind == ASTLine && p.attrs.Name == name })
}

// FindSection returns all repeated headings (sections and subsections), matching
// exactly or after whitespace folding and Unicode-aware case conversion.
func (n *ASTNode) FindSection(heading string, normalized bool) []*ASTNode {
	if normalized {
		heading = normalizeASTText(heading)
	}
	return n.Find(func(p *ASTNode) bool {
		if p.kind != ASTSection && p.kind != ASTSubsection {
			return false
		}
		s := p.attrs.Heading
		if normalized {
			s = normalizeASTText(s)
		}
		return s == heading
	})
}

// FindText searches line/inline contributions (not repeated container text).
// Selecting ASTLine yields each matching physical line once.
func (n *ASTNode) FindText(text string, normalized bool) []*ASTNode {
	if normalized {
		text = normalizeASTText(text)
	}
	return n.Find(func(p *ASTNode) bool {
		if p.kind != ASTLine && p.kind != ASTInline {
			return false
		}
		s := p.text
		if normalized {
			s = normalizeASTText(s)
		}
		return strings.Contains(s, text)
	})
}

// FindSpan returns intersecting nodes for a nonempty half-open byte range.
// For a point (start==end), it returns nodes containing that byte; EOF is absent.
// Ranges outside this source, reversed ranges, and negative offsets are absent.
func (n *ASTNode) FindSpan(start, end int) []*ASTNode {
	if start < 0 || end < start || end > len(n.tree.source) {
		return nil
	}
	return n.Find(func(p *ASTNode) bool {
		a, b := p.span.Start.Offset, p.span.End.Offset
		if start == end {
			return a <= start && start < b
		}
		return a < end && start < b
	})
}
func normalizeASTText(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
