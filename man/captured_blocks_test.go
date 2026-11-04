package man_test

import (
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

const storedControls = ".TH HIDDEN 9\n.SH HIDDEN\n.ft B\n.UR https://stored.invalid\n.RS 8\n.nf\nstored text\n"

func buildCaptured(t *testing.T, source string, opts ...groff.Option) (*groff.AST, *man.Document) {
	t.Helper()
	opts = append([]groff.Option{groff.WithPackage("man"), groff.WithSourceName("capture.1")}, opts...)
	tree, err := groff.ParseAST(strings.NewReader(source), opts...)
	if err != nil {
		t.Fatal(err)
	}
	d, err := man.Build(tree, man.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	if d.Source() != source || d.Raw(1) != source {
		t.Fatal("source changed")
	}
	return tree, d
}

func assertCapturedState(t *testing.T, d *man.Document, after bool) {
	t.Helper()
	if d.Metadata.Title != "OUTER" || d.Metadata.Section != "1" {
		t.Fatalf("stored metadata interpreted: %+v", d.Metadata)
	}
	foundAfter := false
	for _, n := range d.Nodes {
		if (n.Kind == man.Section || n.Kind == man.Subsection) && n.Heading != "REAL" {
			t.Fatalf("stored heading interpreted: %+v", n)
		}
		if n.Kind != man.Paragraph {
			continue
		}
		for _, r := range n.Runs {
			if strings.Contains(r.Text, "after") {
				foundAfter = true
				if r.Style != (man.Style{}) || r.Link != "" || len(n.State.Indents) != 0 || !n.State.Fill {
					t.Fatalf("stored state leaked: %+v %+v", n.State, r)
				}
			}
		}
	}
	if foundAfter != after {
		t.Fatalf("following paragraph presence=%v want %v", foundAfter, after)
	}
	if len(d.Diagnostics) == 0 {
		t.Fatal("unsupported capture was silent")
	}
}

func TestCapturedDefinitionsDoNotInterpretBodies(t *testing.T) {
	for _, cmd := range []string{"de", "am", "de1", "dei", "dei1", "am1", "ami", "ami1", "ig"} {
		t.Run(cmd, func(t *testing.T) {
			header := "." + cmd + " X\n"
			if cmd == "ig" {
				header = ".ig\n"
			}
			raw := header + storedControls + "..\n"
			source := ".TH OUTER 1\n.SH REAL\nbefore\n" + raw + "after\n"
			tree, d := buildCaptured(t, source)
			assertCapturedState(t, d, true)
			var fallback *man.Node
			for i := range d.Nodes {
				n := &d.Nodes[i]
				if n.Kind == man.Unsupported {
					if fallback != nil {
						t.Fatal("capture fragmented")
					}
					fallback = n
				}
			}
			if fallback == nil || fallback.Value != "."+cmd || len(fallback.Runs) != 1 || fallback.Runs[0].Text != raw || !fallback.Runs[0].Unsupported || d.Raw(fallback.ID) != raw {
				t.Fatalf("capture lost raw representation: %+v", fallback)
			}
			block := tree.Root().FindKind(groff.ASTBlock)[0]
			if fallback.Origin.Span != block.Span() {
				t.Fatal("block span lost")
			}
			selected, err := d.SelectAST(block.ID())
			if err != nil || selected.Text(man.TextOptions{Raw: true}) != raw {
				t.Fatalf("block selection: %v", err)
			}
			strict, err := man.Build(tree, man.Options{Strict: true})
			if err == nil || strict.Source() != source {
				t.Fatal("strict mode lost partial source")
			}
			assertCapturedState(t, strict, true)
		})
	}
}

type semanticCaptureHandler struct {
	command string
	calls   *int
}

func (h semanticCaptureHandler) CanHandle(cmd, pkg string) bool {
	return cmd == h.command && pkg == "man"
}
func (h semanticCaptureHandler) Begin(groff.BlockHeader) (groff.BlockRule, error) {
	return groff.BlockRule{Ends: []string{".ENDCAPTURE"}}, nil
}
func (h semanticCaptureHandler) ParseBlock(groff.BlockSource) (groff.Token, error) {
	(*h.calls)++
	return groff.TokenText("handler canonical text"), nil
}

func TestCustomCapturesOverrideSemanticCommands(t *testing.T) {
	for _, cmd := range []string{".LITERAL", ".TH", ".SH", ".EX", ".RS", ".TP", ".\\\""} {
		t.Run(cmd, func(t *testing.T) {
			calls := 0
			opener := "'" + cmd[1:]
			raw := opener + "\n" + storedControls + ".ENDCAPTURE\n"
			// Match only the no-break spelling, leaving live TH/SH headers intact.
			source := ".TH OUTER 1\n.SH REAL\n" + raw + "after\n"
			h := semanticCaptureHandler{command: opener, calls: &calls}
			tree, d := buildCaptured(t, source, groff.WithBlockHandler(h))
			if calls != 1 {
				t.Fatalf("handler calls=%d", calls)
			}
			assertCapturedState(t, d, true)
			captured := 0
			for _, n := range d.Nodes {
				if n.Kind == man.Unsupported {
					captured++
					if len(n.Runs) != 1 || n.Runs[0].Text != raw || !n.Runs[0].Unsupported {
						t.Fatal("handler formatting replaced raw block")
					}
				}
			}
			if captured != 1 {
				t.Fatalf("unsupported blocks=%d", captured)
			}
			_, err := man.Build(tree, man.Options{})
			if err != nil || calls != 1 {
				t.Fatal("semantic construction reran handler")
			}
		})
	}
}

func TestIncompleteNestedAndIndirectCaptures(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		opts      []groff.Option
	}{
		{name: "unterminated", raw: ".de1 X\n" + storedControls},
		{name: "unresolved", raw: ".dei X ENDREF\n.ENDREF\n" + storedControls + "..\n"},
		{name: "nested", raw: ".de1 OUTER OUTEREND\n.de1 INNER INNEREND\n" + storedControls + ".INNEREND\n.OUTEREND\n"},
		{name: "resolved", raw: ".ami1 X ENDREF\n" + storedControls + ".DONE\n", opts: []groff.Option{groff.WithBlockNameResolver(groff.BlockNameResolverFunc(func(groff.BlockHeader, string) (string, error) { return "DONE", nil }))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := ".TH OUTER 1\n.SH REAL\n" + tc.raw
			tree, d := buildCaptured(t, source, tc.opts...)
			assertCapturedState(t, d, false)
			count := 0
			for _, n := range d.Nodes {
				if n.Kind == man.Unsupported {
					count++
					if len(n.Runs) != 1 || n.Runs[0].Text != tc.raw {
						t.Fatal("recovery fragmented or lost source")
					}
				}
			}
			if count != 1 {
				t.Fatalf("unsupported blocks=%d", count)
			}
			if tc.name == "unterminated" || tc.name == "unresolved" {
				if len(tree.Diagnostics()) == 0 {
					t.Fatal("syntax recovery not diagnosed")
				}
			}
		})
	}
}

func TestCapturedBlocksInsidePreformattedAndTags(t *testing.T) {
	raw := ".de1 X\n" + storedControls + "..\n"
	for _, tc := range []struct {
		name, before, after string
		kind                man.Kind
	}{
		{"preformatted", ".nf\n", ".fi\n", man.Preformatted},
		{"tag", ".TP\n", "description\n.PP\n", man.Item},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, d := buildCaptured(t, ".TH OUTER 1\n.SH REAL\n"+tc.before+raw+tc.after+"after\n")
			assertCapturedState(t, d, true)
			found := false
			for _, n := range d.Nodes {
				if n.Kind != tc.kind {
					continue
				}
				runs := n.Runs
				if tc.kind == man.Item {
					runs = n.Tag
				}
				for _, r := range runs {
					if r.Unsupported && r.Text == raw {
						found = true
						if d.Source()[r.Origin.Span.Start.Offset:r.Origin.Span.End.Offset] != raw {
							t.Fatal("capture origin lost")
						}
					}
				}
			}
			if !found {
				t.Fatal("capture not retained in its semantic context")
			}
		})
	}
}

func FuzzCapturedSemanticIsolation(f *testing.F) {
	for _, body := range []string{storedControls, ".de1 INNER END\n.SH hidden\n.END\n", "\\fBtext\n\\*[stored]", "\xff\x00\n", ""} {
		f.Add(body)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if len(body) > 2048 || strings.Contains("\n"+body, "\n.ENDCAPTURE") || strings.Contains("\n"+body, "\n'ENDCAPTURE") {
			t.Skip()
		}
		calls := 0
		raw := ".LITERAL\n" + body + "\n.ENDCAPTURE\n"
		source := ".TH OUTER 1\n.SH REAL\n" + raw + "after\n"
		tree, d := buildCaptured(t, source, groff.WithBlockHandler(semanticCaptureHandler{command: ".LITERAL", calls: &calls}))
		assertCapturedState(t, d, true)
		if calls != 1 {
			t.Fatal("handler lifecycle changed")
		}
		count := 0
		for _, n := range d.Nodes {
			if n.Kind == man.Unsupported {
				count++
				if len(n.Runs) != 1 || n.Runs[0].Text != raw || d.Raw(n.ID) != raw {
					t.Fatal("captured body changed")
				}
			}
		}
		if count != 1 {
			t.Fatal("captured body fragmented")
		}
		block := tree.Root().FindKind(groff.ASTBlock)[0]
		lines := block.FindKind(groff.ASTLine)
		for _, n := range []*groff.ASTNode{block, lines[0], lines[len(lines)/2], lines[len(lines)-1]} {
			selected, err := d.SelectAST(n.ID())
			if err != nil || selected.Text(man.TextOptions{Raw: true}) != raw {
				t.Fatalf("captured source identity cannot select its complete owner: %v", err)
			}
		}
	})
}
