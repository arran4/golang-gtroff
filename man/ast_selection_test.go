package man_test

import (
	"encoding/json"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func TestASTStructuralBlockSelection(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		kind man.Kind
	}{
		{".RS 4\nchosen\n.RE\n", man.Indent},
		{".nf\nchosen\n.fi\n", man.Preformatted},
		{".EX\nchosen\n.EE\n", man.Preformatted},
	} {
		t.Run(strings.Split(tc.raw, "\n")[0], func(t *testing.T) {
			source := ".TH OUTER 1\n.SH REAL\nbefore\n.RS 2\n.ft I\n" + tc.raw + "outside\n.RE\n"
			tree, d := buildCaptured(t, source)
			var block *groff.ASTNode
			for _, n := range tree.Root().FindKind(groff.ASTBlock) {
				if n.Raw() == tc.raw {
					block = n
				}
			}
			if block == nil {
				t.Fatal("AST block absent")
			}
			for _, id := range []groff.NodeID{block.ID(), block.Children()[0].ID()} {
				selected, err := d.SelectAST(id)
				if err != nil {
					t.Fatal(err)
				}
				if len(selected.Roots) != 1 || d.Node(selected.Roots[0]).Kind != tc.kind || selected.Text(man.TextOptions{Raw: true}) != tc.raw || selected.Text(man.TextOptions{Normalize: true}) != "chosen" {
					t.Fatalf("block selection does not retain its complete semantic owner: %+v", selected)
				}
				context := selected.Contexts[0]
				if !context.Style.Italic || len(context.Indents) != 1 || context.Indents[0] != "2" {
					t.Fatalf("upstream state lost: %+v", context)
				}
			}
			encoded, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			var decoded man.Document
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			selected, err := decoded.SelectAST(block.ID())
			if err != nil || selected.Text(man.TextOptions{Normalize: true}) != "chosen" || selected.Text(man.TextOptions{Raw: true}) != "" {
				t.Fatal("JSON model lost AST provenance or invented raw source", err)
			}
		})
	}
}

func TestASTCapturedSelectionUsesOneOwner(t *testing.T) {
	raw := ".de1 OUTER OUTEREND\n.de1 INNER INNEREND\nstored\n.INNEREND\n.OUTEREND\n"
	for _, tc := range []struct {
		name, before, after string
		kind                man.Kind
	}{
		{"literal", "", "", man.Unsupported},
		{"preformatted", ".nf\n", ".fi\n", man.Preformatted},
		{"tag", ".TP\n", "description\n.PP\n", man.Item},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tree, d := buildCaptured(t, ".TH OUTER 1\n.SH REAL\nbefore\n"+tc.before+raw+tc.after+"outside\n")
			var block *groff.ASTNode
			for _, n := range tree.Root().FindKind(groff.ASTBlock) {
				if n.Raw() == raw {
					block = n
				}
			}
			if block == nil {
				t.Fatal("capture absent")
			}
			owner := man.ID(0)
			block.Walk(func(n *groff.ASTNode) groff.WalkAction {
				if n.Kind() != groff.ASTBlock && n.Kind() != groff.ASTLine {
					return groff.WalkContinue
				}
				selected, err := d.SelectAST(n.ID())
				if err != nil {
					t.Fatalf("stored source %d cannot select its semantic owner: %v", n.ID(), err)
				}
				if len(selected.Roots) != 1 || d.Node(selected.Roots[0]).Kind != tc.kind {
					t.Fatal("wrong semantic owner")
				}
				if owner != 0 && owner != selected.Roots[0] {
					t.Fatal("capture fragmented across semantic owners")
				}
				owner = selected.Roots[0]
				if got := selected.Text(man.TextOptions{Raw: true}); !strings.Contains(got, raw) || strings.Contains(got, "outside") || strings.Contains(got, "before") {
					t.Fatalf("raw selection includes unrelated source: %q", got)
				}
				return groff.WalkContinue
			})
			for _, n := range d.Nodes {
				seen := map[groff.NodeID]bool{}
				for _, id := range n.Origin.ASTIDs {
					if tree.Node(id) == nil || seen[id] {
						t.Fatal("invalid or duplicate source identity")
					}
					seen[id] = true
				}
			}
		})
	}
}

func TestASTSelectionDoesNotInventContent(t *testing.T) {
	tree, d := buildCaptured(t, ".TH OUTER 1\n.SH REAL\n.\\\" comment\nchosen\n")
	for _, id := range []groff.NodeID{0, ^groff.NodeID(0), tree.Root().FindKind(groff.ASTInline)[0].ID(), tree.Root().FindMacro(".\\\"")[0].ID()} {
		if selected, err := d.SelectAST(id); err == nil || selected != nil {
			t.Fatalf("non-content AST ID %d selected", id)
		}
	}
	// Malformed structural headers do not acquire an invented whole-block owner.
	tree, d = buildCaptured(t, ".RS \"broken\nchosen\n.RE\n")
	block := tree.Root().FindKind(groff.ASTBlock)[0]
	if selected, err := d.SelectAST(block.ID()); err == nil || selected != nil {
		t.Fatal("malformed structure acquired a semantic group")
	}
}

func TestASTCapturedSelectionCRLFAndJSON(t *testing.T) {
	raw := ".de1 SAVED\r\n雪\r\n..\r\n"
	tree, d := buildCaptured(t, ".SH REAL\r\n"+raw+"outside")
	block := tree.Root().FindKind(groff.ASTBlock)[0]
	line := block.FindKind(groff.ASTLine)[1]
	selected, err := d.SelectAST(line.ID())
	if err != nil || selected.Text(man.TextOptions{Raw: true}) != raw {
		t.Fatal("CRLF/Unicode source selection changed", err)
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var decoded man.Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	again, err := decoded.SelectAST(line.ID())
	if err != nil || again.Text(man.TextOptions{}) != selected.Text(man.TextOptions{}) || again.Text(man.TextOptions{Raw: true}) != "" {
		t.Fatal("captured provenance lost during JSON round trip", err)
	}
}
