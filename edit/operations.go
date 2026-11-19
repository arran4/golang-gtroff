package edit

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
)

func (w *Working) commandPatches(ctx context.Context, c Command, matches []query.Match, m *budget) ([]Patch, []Diagnostic, error) {
	if len(c.Text) > w.o.MaxBytes || len(c.Find) > w.o.MaxBytes {
		return nil, nil, ErrLimit
	}
	payloadBytes := 0
	var patches []Patch
	var ds []Diagnostic
	var destination *query.Match
	if c.Operation == MoveBefore || c.Operation == MoveAfter {
		if m.left == 0 {
			return nil, nil, ErrLimit
		}
		r, err := query.Execute(ctx, w.set, c.Destination, query.Options{View: c.View, MaxWork: m.left, MaxResultBytes: 16 << 20, MaxMatches: w.o.MaxPatches})
		if err != nil {
			return nil, nil, err
		}
		if err = m.spend(r.WorkUnits); err != nil {
			return nil, nil, err
		}
		if len(r.Matches) != 1 {
			return nil, nil, fmt.Errorf("%w: move destination must be unique", ErrSelection)
		}
		destination = &r.Matches[0]
		if destination.Source.Name != w.root && !w.o.IncludeEdits {
			return nil, nil, fmt.Errorf("%w: move destination is included source %q", ErrTarget, destination.Source.Name)
		}
	}
	models := map[string]*man.Document{}
	blocked := map[string][]insertionRange{}
	for _, match := range matches {
		if err := m.spend(1 + len(c.Text)); err != nil {
			return nil, nil, err
		}
		name := match.Source.Name
		s := w.files[name]
		a, b := match.Span.Start.Offset, match.Span.End.Offset
		if name != w.root && !w.o.IncludeEdits {
			return nil, nil, fmt.Errorf("%w: match %d belongs to included source %q", ErrTarget, match.Index, name)
		}
		if a < 0 || b < a || b > len(s) {
			return nil, nil, ErrTarget
		}
		p := Patch{Source: name, Digest: match.Source.Digest, Start: a, End: b, Match: match.Index, owner: fmt.Sprintf("%s:%d:%d", name, a, b)}
		switch c.Operation {
		case ReplaceSource:
			p.Data = []byte(c.Text)
		case FindReplaceSource:
			if err := m.spend(b - a + len(c.Find) + len(c.Text)); err != nil {
				return nil, nil, err
			}
			if !strings.Contains(s[a:b], c.Find) {
				if w.o.Strict {
					return nil, nil, fmt.Errorf("%w: find text absent", ErrSelection)
				}
				ds = append(ds, Diagnostic{"find-absent", "source find text absent", name, match.Index})
				continue
			}
			// Patch each occurrence rather than a whole container: unchanged
			// malformed/opaque bytes and their diagnostics retain their ranges.
			at := a
			for at <= b-len(c.Find) {
				rel := strings.Index(s[at:b], c.Find)
				if rel < 0 {
					break
				}
				start := at + rel
				piece := p
				piece.Start = start
				piece.End = start + len(c.Find)
				if err := m.spend(len(c.Text)); err != nil {
					return nil, nil, err
				}
				piece.Data = []byte(c.Text)
				payloadBytes += len(piece.Data)
				if payloadBytes > w.o.MaxBytes {
					return nil, nil, ErrLimit
				}
				patches = append(patches, piece)
				if len(patches) > w.o.MaxPatches {
					return nil, nil, ErrLimit
				}
				at = piece.End
			}
			continue
		case Delete:
		case InsertBefore:
			p.End = p.Start
			p.Data = []byte(c.Text)
		case InsertAfter:
			p.Start = p.End
			p.Data = []byte(c.Text)
		case MoveBefore, MoveAfter:
			position := destination.Span.Start.Offset
			if c.Operation == MoveAfter {
				position = destination.Span.End.Offset
			}
			if name == destination.Source.Name && position > a && position < b {
				return nil, nil, ErrOverlap
			}
			if err := m.spend(b - a); err != nil {
				return nil, nil, err
			}
			insert := Patch{Source: destination.Source.Name, Digest: destination.Source.Digest, Start: position, End: position, Data: []byte(s[a:b]), Match: match.Index, owner: p.owner}
			payloadBytes += len(insert.Data)
			if payloadBytes > w.o.MaxBytes {
				return nil, nil, ErrLimit
			}
			patches = append(patches, insert)
		case AppendParagraph, AppendSection, AppendDefinition:
			if match.Kind != query.Source && match.Kind != query.Document && match.Kind != query.Section && match.Kind != query.Subsection {
				return nil, nil, fmt.Errorf("%w: append requires source/document/section/subsection", ErrTarget)
			}
			ranges, ok := blocked[name]
			if !ok {
				var err error
				ranges, err = w.insertionRanges(match, m)
				if err != nil {
					return nil, nil, err
				}
				blocked[name] = ranges
			}
			if err := safeInsertion(ranges, b); err != nil {
				return nil, nil, err
			}
			fragment, err := canonical(c, w.o.MaxBytes)
			if err != nil {
				return nil, nil, err
			}
			p.Start = b
			p.End = b
			p.Data = []byte(withSeparator(s, b, lineEnding(s, b), fragment))
		case ReplaceDescription:
			if match.Kind != query.Definition || match.View != query.Semantic {
				return nil, nil, fmt.Errorf("%w: description requires semantic definition", ErrTarget)
			}
			model := models[name]
			if model == nil {
				if err := m.spend(len(s)); err != nil {
					return nil, nil, err
				}
				var err error
				model, err = man.Build(w.set.File(match.Source.ID).AST(), man.Options{MaxBytes: w.o.MaxBytes})
				if err != nil {
					return nil, nil, err
				}
				models[name] = model
			}
			n := model.Node(match.SemanticID)
			if n == nil || n.Kind != man.Item {
				return nil, nil, ErrTarget
			}
			if len(n.Tag) == 0 {
				header := strings.Fields(s[n.Origin.Span.Start.Offset:n.Origin.Span.End.Offset])
				if len(header) > 0 && (header[0] == ".TP" || header[0] == "'TP") {
					return nil, nil, fmt.Errorf("%w: definition lacks a captured tag line", ErrTarget)
				}
			}
			start, end, plain := descriptionRange(s, n, model)
			if !plain && !c.Rewrite {
				return nil, nil, ErrNeedsRewrite
			}
			if !plain {
				ds = append(ds, Diagnostic{"canonical-rewrite", "complex description replaced with escaped literal text; review removed source bytes", name, match.Index})
			}
			if err := literal(c.Text, false); err != nil {
				return nil, nil, err
			}
			p.Start = start
			p.End = end
			var body strings.Builder
			for _, line := range strings.Split(strings.ReplaceAll(c.Text, "\r\n", "\n"), "\n") {
				body.WriteString(convert.ProtectLine(line))
				body.WriteByte('\n')
			}
			p.Data = []byte(withSeparator(s, start, lineEnding(s, start), body.String()))
		default:
			return nil, nil, ErrOptions
		}
		if len(p.Data) > w.o.MaxBytes {
			return nil, nil, ErrLimit
		}
		payloadBytes += len(p.Data)
		if payloadBytes > w.o.MaxBytes {
			return nil, nil, ErrLimit
		}
		patches = append(patches, p)
		if len(patches) > w.o.MaxPatches {
			return nil, nil, ErrLimit
		}
	}
	return patches, ds, nil
}

func literal(s string, single bool) error {
	if !utf8.ValidString(s) || single && strings.ContainsAny(s, "\r\n") {
		return fmt.Errorf("%w: expected literal UTF-8 data", ErrOptions)
	}
	for _, r := range s {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' || r == 127 {
			return fmt.Errorf("%w: control character in literal data", ErrOptions)
		}
	}
	if single && len(s) > 4096 {
		return ErrLimit
	}
	return nil
}
func canonical(c Command, maxBytes int) (string, error) {
	if len(c.Text) > maxBytes || len(c.Term) > maxBytes || len(c.Heading) > maxBytes {
		return "", ErrLimit
	}
	if err := literal(c.Text, false); err != nil {
		return "", err
	}
	root := man.Node{ID: 1, Kind: man.DocumentNode, Children: []man.ID{2}}
	d := &man.Document{Version: 1, Nodes: []man.Node{root}}
	runs := []man.Run{{Text: strings.ReplaceAll(c.Text, "\r\n", "\n")}}
	switch c.Operation {
	case AppendParagraph:
		d.Nodes = append(d.Nodes, man.Node{ID: 2, Parent: 1, Kind: man.Paragraph, Runs: runs})
	case AppendSection:
		if c.Heading == "" {
			return "", fmt.Errorf("%w: heading is required", ErrOptions)
		}
		if err := literal(c.Heading, true); err != nil {
			return "", err
		}
		d.Nodes = append(d.Nodes, man.Node{ID: 2, Parent: 1, Kind: man.Section, Heading: c.Heading})
		if c.Text != "" {
			d.Nodes[1].Children = []man.ID{3}
			d.Nodes = append(d.Nodes, man.Node{ID: 3, Parent: 2, Kind: man.Paragraph, Runs: runs})
		}
	case AppendDefinition:
		if c.Term == "" {
			return "", fmt.Errorf("%w: term is required", ErrOptions)
		}
		if err := literal(c.Term, true); err != nil {
			return "", err
		}
		d.Nodes = append(d.Nodes, man.Node{ID: 2, Parent: 1, Kind: man.DefinitionList, Children: []man.ID{3}}, man.Node{ID: 3, Parent: 2, Kind: man.Item, Tag: []man.Run{{Text: c.Term}}, Children: []man.ID{4}}, man.Node{ID: 4, Parent: 3, Kind: man.Paragraph, Runs: runs})
	default:
		return "", ErrOptions
	}
	var out bytes.Buffer
	_, err := convert.WriteMan(&out, d, convert.Options{Strict: true, MaxOutputBytes: maxBytes})
	return out.String(), err
}

func lineEnding(s string, at int) string {
	if at > len(s) {
		at = len(s)
	}
	if i := strings.LastIndexByte(s[:at], '\n'); i >= 0 {
		if i > 0 && s[i-1] == '\r' {
			return "\r\n"
		}
		return "\n"
	}
	if i := strings.IndexByte(s[at:], '\n'); i >= 0 {
		if at+i > 0 && s[at+i-1] == '\r' {
			return "\r\n"
		}
	}
	return "\n"
}
func withSeparator(s string, at int, nl, fragment string) string {
	fragment = strings.ReplaceAll(fragment, "\n", nl)
	if at > 0 && s[at-1] != '\n' {
		return nl + fragment
	}
	return fragment
}

type insertionRange struct{ lo, hi int }

func (w *Working) insertionRanges(match query.Match, m *budget) ([]insertionRange, error) {
	var ranges []insertionRange
	var err error
	w.set.File(match.Source.ID).AST().Root().Walk(func(n *groff.ASTNode) groff.WalkAction {
		if err = m.spend(1); err != nil {
			return groff.WalkStop
		}
		if n.Kind() != groff.ASTBlock {
			return groff.WalkContinue
		}
		span := n.Span()
		hi := span.End.Offset
		if n.Attributes().Closed {
			hi--
		}
		lo := span.Start.Offset + 1
		if hi >= lo {
			if len(ranges) > 0 && ranges[len(ranges)-1].hi >= lo {
				ranges[len(ranges)-1].hi = max(ranges[len(ranges)-1].hi, hi)
			} else {
				ranges = append(ranges, insertionRange{lo, hi})
			}
		}
		return groff.WalkContinue
	})
	return ranges, err
}
func safeInsertion(ranges []insertionRange, at int) error {
	i := sort.Search(len(ranges), func(i int) bool { return ranges[i].lo > at }) - 1
	if i >= 0 && at <= ranges[i].hi {
		return fmt.Errorf("%w: insertion inside captured/unclosed block", ErrTarget)
	}
	return nil
}
func descriptionRange(s string, n *man.Node, d *man.Document) (int, int, bool) {
	start := n.Origin.Span.Start.Offset
	if start >= 0 && start < len(s) {
		if i := strings.IndexByte(s[start:], '\n'); i >= 0 {
			start += i + 1
		} else {
			start = len(s)
		}
	}
	// Runs retain complete physical tag-line spans, including styled tags.
	for _, r := range n.Tag {
		if r.Origin.Span.End.Offset > start {
			start = r.Origin.Span.End.Offset
		}
	}
	end := start
	if len(n.Children) > 0 {
		start = len(s)
	}
	plain := true
	for _, id := range n.Children {
		child := d.Node(id)
		if child == nil {
			continue
		}
		if child.Origin.Span.Start.Offset < start {
			start = child.Origin.Span.Start.Offset
		}
		if child.Origin.Span.End.Offset > end {
			end = child.Origin.Span.End.Offset
		}
		if child.Kind != man.Paragraph || len(child.Children) > 0 {
			plain = false
		}
	}
	if start < 0 || end < start || end > len(s) {
		return 0, 0, false
	}
	for _, line := range strings.Split(s[start:end], "\n") {
		if strings.Contains(line, "\\") || strings.HasPrefix(line, ".") || strings.HasPrefix(line, "'") {
			plain = false
		}
	}
	return start, end, plain
}
