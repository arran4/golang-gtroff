package analyze

import (
	"errors"
	"fmt"
	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
	"sort"
	"strings"
)

// indexSemantics uses actual source-backed man contributors rather than treating
// catalogue membership as interpretation support. Opaque/comment nodes never
// become active semantic evidence. Other macro-package semantics are unassessed.
func indexSemantics(file *source.File, profile Profile, maxNodes int) (map[groff.NodeID]Status, []groff.Diagnostic, error) {
	index := map[groff.NodeID]Status{}
	if profile.Name == "ms" || profile.Name == "mdoc" || profile.Name == "mixed" {
		return index, nil, nil
	}
	model, err := man.Build(file.AST(), man.Options{MaxNodes: maxNodes, MaxBytes: len(file.AST().Source()) + 1})
	if err != nil {
		if errors.Is(err, man.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return index, nil, fmt.Errorf("semantic capability inventory: %w", err)
	}
	var unsupported []groff.Span
	if model.MetadataOrigin != nil {
		for _, id := range model.MetadataOrigin.ASTIDs {
			index[id] = Modeled
		}
	}
	for _, node := range model.Nodes {
		for _, id := range node.Origin.ASTIDs {
			if index[id] == "" {
				index[id] = Modeled
			}
			if node.Kind == man.Unsupported {
				index[id] = Unsupported
			}
		}
		if node.Kind == man.Unsupported {
			unsupported = append(unsupported, node.Origin.Span)
		}
		for _, runs := range [][]man.Run{node.Runs, node.Tag} {
			for _, run := range runs {
				if run.Unsupported {
					for _, id := range run.Origin.ASTIDs {
						if index[id] != Unsupported {
							index[id] = PartiallyModeled
						}
					}
				}
			}
		}
	}
	// Captures can cover physical IDs beyond the model's contributor list.
	sort.Slice(unsupported, func(i, j int) bool { return unsupported[i].Start.Offset < unsupported[j].Start.Offset })
	intervals := make([]groff.Span, 0, len(unsupported))
	for _, span := range unsupported {
		if len(intervals) > 0 && span.Start.Offset <= intervals[len(intervals)-1].End.Offset {
			if span.End.Offset > intervals[len(intervals)-1].End.Offset {
				intervals[len(intervals)-1].End = span.End
			}
		} else {
			intervals = append(intervals, span)
		}
	}
	diagnosed := map[int]bool{}
	for _, diagnostic := range model.Diagnostics {
		diagnosed[diagnostic.Position.Offset] = true
	}
	for _, line := range file.AST().Root().FindKind(groff.ASTLine) {
		// Some supported state/closing controls produce no content node. The model
		// actually consumed them without a diagnostic; absence of a contributing
		// origin must not label these accepted operations as unknown support.
		attrs := line.Attributes()
		if index[line.ID()] == "" && !diagnosed[line.Span().Start.Offset] && attrs.Name != "" && attrs.TokenKind != groff.KindUnknown && attrs.TokenKind != groff.KindComment && activeNode(line) {
			index[line.ID()] = Modeled
		}

		if diagnosed[line.Span().Start.Offset] && index[line.ID()] == Modeled {
			index[line.ID()] = PartiallyModeled
		}
		at := sort.Search(len(intervals), func(i int) bool { return intervals[i].End.Offset > line.Span().Start.Offset })
		if at < len(intervals) && intervals[at].Start.Offset <= line.Span().Start.Offset {
			index[line.ID()] = Unsupported
		}
	}
	return index, model.Diagnostics, nil
}
func inlineSemantics(c nodeContext, index map[groff.NodeID]Status) Status {
	if !c.active || c.node.Parent() == nil || index[c.node.Parent().ID()] == "" {
		return NotAssessed
	}
	if index[c.node.Parent().ID()] == Unsupported {
		return Unsupported
	}
	switch c.inline.Kind {
	case groff.InlineText:
		return Modeled
	case groff.InlineStringRef, groff.InlineRegisterRef, groff.InlineSizeChange:
		return Unsupported
	case groff.InlineFontChange:
		if c.inline.Name == "P" {
			return Modeled
		}
		if contains([]string{"R", "I", "B", "BI", "1", "2", "3", "4", "C", "CW", "CR", "CB", "CBI", "CI"}, strings.ToUpper(c.inline.Name)) {
			return Modeled
		}
	case groff.InlineSpecialChar:
		if _, ok := groff.DecodeDefaultSpecialChar(c.inline.Name); ok {
			return Modeled
		}
	case groff.InlineEscape:
		// These are the actual cases in man.decode, not a generic groff support list.
		if contains([]string{"e", "\\", "-", " ", "~", "t", "&", "c", "\"", "#"}, c.inline.Name) {
			return Modeled
		}
	}
	return Unsupported
}
