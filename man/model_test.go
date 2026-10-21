package man

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
)

func buildTest(t *testing.T, source string, o Options) (*Document, *groff.AST) {
	t.Helper()
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"), groff.WithSourceName("input.1"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := Build(tree, o)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	return d, tree
}
func byKind(d *Document, kind Kind) []Node {
	var out []Node
	for _, n := range d.Nodes {
		if n.Kind == kind {
			out = append(out, n)
		}
	}
	return out
}
func TestFlowStylesAndHardBreaks(t *testing.T) {
	source := ".TH TOOL 1\n.SH DESCRIPTION\nThe\n.I tool\nhandles\n.B files\nwell.\n.br\nNext line.\n.PP\n.BR printf (3)\n.BI bold italic\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	paragraphs := byKind(d, Paragraph)
	if len(paragraphs) != 2 {
		t.Fatalf("each source line became a paragraph: %d", len(paragraphs))
	}
	if got := textRuns(paragraphs[0].Runs); got != "The tool handles files well.\nNext line." {
		t.Fatalf("flow/hard break: %q", got)
	}
	if got := textRuns(paragraphs[1].Runs); got != "printf(3) bolditalic" {
		t.Fatalf("alternating styles %q", got)
	}
	var foundBold, foundItalic bool
	for _, r := range paragraphs[0].Runs {
		if r.Text == "files" {
			foundBold = r.Style.Bold
		}
		if r.Text == "tool" {
			foundItalic = r.Style.Italic
		}
		if r.Origin.Span.Start.Line < 3 {
			t.Fatal("run lost line provenance")
		}
	}
	if !foundBold || !foundItalic {
		t.Fatal("macro style lost")
	}
	if d.Source() != source || d.Raw(1) != source {
		t.Fatal("semantic extraction overwrote source")
	}
}
func TestListsIndentAndPreformatted(t *testing.T) {
	source := ".SH OPTIONS\n.TP 6\n.B \\-o\nChoose a file\non disk.\n.IP \\(bu 3\nA bullet.\n.PP\nAfter list.\n.RS 4\nIndented text.\n.RS 2\nNested text.\n.RE\n.RE\n.SH EXAMPLES\n.EX\n  code  雪\n.B bold\n.EE\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	lists := byKind(d, DefinitionList)
	items := byKind(d, Item)
	indents := byKind(d, Indent)
	pre := byKind(d, Preformatted)
	if len(lists) != 1 || len(items) != 2 || len(indents) != 2 || len(pre) != 1 {
		t.Fatalf("structure lists/items/indents/pre=%d/%d/%d/%d", len(lists), len(items), len(indents), len(pre))
	}
	if textRuns(items[0].Tag) != "-o" || textRuns(items[1].Tag) != "•" {
		t.Fatal("tag extraction")
	}
	if textRuns(pre[0].Runs) != "  code  雪\nbold\n" {
		t.Fatalf("preformatted spacing/macros: %q", textRuns(pre[0].Runs))
	}
	paragraphs := byKind(d, Paragraph)
	for _, p := range paragraphs {
		if textRuns(p.Runs) == "After list." && d.Node(p.Parent).Kind == Item {
			t.Fatal("paragraph remained in definition item")
		}
	}
	if len(indents[1].State.Indents) != 1 || indents[1].State.Indents[0] != "4" {
		t.Fatal("nested indent context")
	}
	// IDs in a freshly built model must also be the structural preorder.
	var order []ID
	stack := []ID{1}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		order = append(order, id)
		children := d.Node(id).Children
		for i := len(children) - 1; i >= 0; i-- {
			stack = append(stack, children[i])
		}
	}
	for i, id := range order {
		if id != ID(i+1) {
			t.Fatal("noncontiguous definition list reordered source")
		}
	}
}
func TestSelectionCarriesStateAndRepeatedSections(t *testing.T) {
	source := ".TH TOOL 1 DATE OS MANUAL\n.SH OPTIONS\n.PP\n.ft B\n.RS 4\nchosen\ntext\n.RE\n.SS OPTIONS\nnested\n.SH OTHER\noutside\n.SH OPTIONS\nsecond\n"
	d, tree := buildTest(t, source, Options{Strict: true})
	selection, err := d.SelectSection("options", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Roots) != 3 || selection.Metadata.Title != "TOOL" || strings.Contains(selection.Text(TextOptions{}), "outside") {
		t.Fatal("repeated section selection")
	}
	var paragraph Node
	for _, n := range d.Nodes {
		if n.Kind == Paragraph && textRuns(n.Runs) == "chosen text" {
			paragraph = n
		}
	}
	single, err := d.SelectNode(paragraph.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !single.Contexts[0].Style.Bold || len(single.Contexts[0].Indents) != 1 || single.Contexts[0].Indents[0] != "4" {
		t.Fatalf("upstream state %+v", single.Contexts[0])
	}
	if got := single.Text(TextOptions{Raw: true}); got != "chosen\ntext\n" {
		t.Fatalf("raw selected bytes %q", got)
	}
	if got := single.Text(TextOptions{Normalize: true}); got != "chosen text" {
		t.Fatalf("normalized selected text %q", got)
	}
	headingAST := tree.Root().FindSection("OPTIONS", false)[0]
	fromAST, err := d.SelectAST(headingAST.ID())
	if err != nil || fromAST.Roots[0] != selection.Roots[0] {
		t.Fatal("stable AST section selection")
	}
	rangeSelection, err := d.SelectRange(paragraph.ID, paragraph.ID)
	if err != nil || rangeSelection.Text(TextOptions{}) != single.Text(TextOptions{}) {
		t.Fatal("semantic range")
	}
	if ids := d.FindTerm("CHOSEN   TEXT", true); len(ids) != 1 || ids[0] != paragraph.ID {
		t.Fatal("term extraction")
	}
	term, err := d.SelectTerm("CHOSEN   TEXT", true)
	if err != nil || term.Text(TextOptions{Normalize: true}) != "chosen text" {
		t.Fatal("term selection extraction")
	}

	if _, err := d.SelectSection("absent", false); err == nil {
		t.Fatal("missing section matched")
	}
	for _, pair := range [][2]ID{{0, 1}, {2, 1}, {1, ID(len(d.Nodes) + 1)}} {
		if _, err := d.SelectRange(pair[0], pair[1]); err == nil {
			t.Fatal("invalid range matched")
		}
	}
}
func TestInlineStateReferencesAndSafeFallback(t *testing.T) {
	source := ".SH DESCRIPTION\n\\fBbold\nstill bold \\fPnormal \\(bu \\*[secret]\n.PP\n.UR https://example.invalid\nlinked text\n.UE .\n.MR printf 3 ,\n.so /etc/passwd\n.sy touch forbidden\n.de CUSTOM\nnot evaluated\n..\n"
	d, _ := buildTest(t, source, Options{})
	if len(d.Diagnostics) < 3 {
		t.Fatal("unsupported semantics were silent")
	}
	paragraphs := byKind(d, Paragraph)
	if len(paragraphs) < 2 {
		t.Fatal("paragraphs missing")
	}
	for _, r := range paragraphs[0].Runs {
		if strings.Contains(r.Text, "still bold") && !r.Style.Bold {
			t.Fatal("font state lost across physical line")
		}
	}
	var foundLink, foundReference bool
	for _, p := range paragraphs {
		for _, r := range p.Runs {
			if r.Text == "linked text" && r.Link == "https://example.invalid" {
				foundLink = true
			}
			if r.Reference == "printf(3)" {
				foundReference = true
			}
		}
	}
	if !foundLink || !foundReference {
		t.Fatal("references lost")
	}
	all := d.Text(1, TextOptions{})
	for _, want := range []string{"\\*[secret]", ".so /etc/passwd", ".sy touch forbidden", "not evaluated"} {
		if !strings.Contains(all, want) {
			t.Fatalf("unsupported input disappeared: %s", want)
		}
	}
	tree, _ := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	strict, err := Build(tree, Options{Strict: true})
	if err == nil || strict.Source() != source {
		t.Fatal("strict fallback source contract")
	}
}
func TestNextLineHeadingAndNextLineStyle(t *testing.T) {
	source := ".TH TOOL 1\n.SH\nNAME\n.B\ncommand\n.PP\nordinary\n.SS\nDetails\nbody\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	if len(byKind(d, Section)) != 1 || byKind(d, Section)[0].Heading != "NAME" || byKind(d, Subsection)[0].Heading != "Details" {
		t.Fatal("next-line headings")
	}
	paragraphs := byKind(d, Paragraph)
	if len(paragraphs) != 3 || !paragraphs[0].Runs[0].Style.Bold || paragraphs[1].Runs[0].Style.Bold {
		t.Fatal("next-line style scope")
	}
}
func TestModelJSONCanonicalRoundTrip(t *testing.T) {
	d, _ := buildTest(t, ".TH TOOL 1\n.SH NAME\nhello\n.B world\n", Options{Strict: true})
	want, err := d.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var again Document
	if err := json.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	got, err := again.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("semantic JSON round-trip\n%s\n%s", want, got)
	}
	if again.Source() != "" {
		t.Fatal("JSON silently invented original source")
	}
	// Equivalent flowing source line splits have the same canonical semantics.
	a, _ := buildTest(t, ".SH NAME\nhello\nworld\n", Options{Strict: true})
	b, _ := buildTest(t, ".SH NAME\nhello world\n", Options{Strict: true})
	left, _ := a.Canonical()
	right, _ := b.Canonical()
	if !bytes.Equal(left, right) {
		t.Fatal("canonical semantics depend on physical line wrapping")
	}
}
func TestBoundsAndInvalidModels(t *testing.T) {
	for _, tc := range []struct {
		input string
		opts  Options
	}{
		{strings.Repeat(".PP\ntext\n", 20), Options{MaxNodes: 8}},
		{strings.Repeat(".RS\n", 20) + "text\n" + strings.Repeat(".RE\n", 20), Options{MaxDepth: 3}},
		{"source remains available", Options{MaxBytes: 4}},
	} {
		tree, _ := groff.ParseAST(strings.NewReader(tc.input))
		d, err := Build(tree, tc.opts)
		if !errors.Is(err, ErrLimit) || d.Source() != tc.input || len(d.Diagnostics) == 0 {
			t.Fatal("semantic budget failure lost source")
		}
		if err := d.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Build(nil, Options{}); err == nil {
		t.Fatal("nil AST accepted")
	}
	d, _ := buildTest(t, ".SH NAME\nbody\n", Options{})
	d.Nodes[0].Children = append(d.Nodes[0].Children, 1)
	if err := d.Validate(); err == nil {
		t.Fatal("cyclic model accepted")
	}
	if _, err := d.SelectNode(1); err == nil {
		t.Fatal("selection accepted mutated invalid tree")
	}
}
func FuzzManModel(f *testing.F) {
	for _, s := range []string{"", ".SH NAME\n雪\n", ".TP\n.B opt\nbody", ".EX\n.B bold\n.EE", ".B \"broken\n", ".de X\nbody", "\xff\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 16384 {
			t.Skip()
		}
		tree, _ := groff.ParseAST(strings.NewReader(source), groff.WithASTNodes(4096), groff.WithASTDepth(16))
		d, _ := Build(tree, Options{MaxNodes: 2048, MaxDepth: 16})
		if d == nil || d.Source() != source || d.Raw(1) != source {
			t.Fatal("semantic model lost source")
		}
		if err := d.Validate(); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Canonical(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRawSectionsAndContinuation(t *testing.T) {
	source := ".TH TOOL 1\n.SH NAME\nfirst\\c\nsecond\n.\\\" trailing comment\n\n.SH OTHER\noutside\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	selection, err := d.SelectSection("NAME", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := selection.Text(TextOptions{}); got != "NAME\nfirstsecond\n" {
		t.Fatalf("line continuation: %q", got)
	}
	want := ".SH NAME\nfirst\\c\nsecond\n.\\\" trailing comment\n\n"
	if got := selection.Text(TextOptions{Raw: true, Normalize: true}); got != want {
		t.Fatalf("raw section ending: %q", got)
	}
	if d.MetadataOrigin == nil || d.MetadataOrigin.Span.Start.Line != 1 || len(d.MetadataOrigin.ASTIDs) != 1 {
		t.Fatal("title provenance missing")
	}
	// A supported maximum indent depth accepts content and close delimiters.
	buildTest(t, ".RS\n.RS\n.RS\nbody\n.RE\n.RE\n.RE\n", Options{Strict: true, MaxDepth: 3})
}

func TestMalformedTagAndPreformattedFallback(t *testing.T) {
	for _, source := range []string{
		".SH OPTIONS\n.TP\n.UNKNOWN tag\nbody\n",
		".SH EXAMPLES\n.nf\n.de X\nbody\n..\ntext\n.fi\n",
		".SH OPTIONS\n.TP\n.de X\ntag body\n..\ndescription\n",
	} {
		d, _ := buildTest(t, source, Options{})
		if len(d.Diagnostics) == 0 {
			t.Fatal("unsupported semantic form was silent")
		}
		if d.Raw(1) != source {
			t.Fatal("fallback lost source")
		}
	}
}

func TestExistingManFixtures(t *testing.T) {
	for _, path := range []string{"../testdata/simple.1", "../testdata/complex.7", "../testdata/environ_snippet.7"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		d, _ := buildTest(t, string(source), Options{})
		if d.Metadata.Title == "" || len(byKind(d, Section)) == 0 || d.Text(1, TextOptions{}) == "" {
			t.Fatal("realistic man fixture has no semantic output")
		}
	}
}

func TestUnexpectedStructureRemainsVisible(t *testing.T) {
	for _, source := range []string{
		".EX\n.TP\ntag\nbody\n.EE\n",
		".TP\n.PP\nbody\n",
		".TP\n.TP\n.B tag\nbody\n",
	} {
		d, _ := buildTest(t, source, Options{})
		if len(d.Diagnostics) == 0 {
			t.Fatal("unexpected structure was silently accepted")
		}
	}
}

func TestCommentsInPreformattedAndExtraOptionArguments(t *testing.T) {
	d, _ := buildTest(t, ".EX\n.\\\" comment with \" unmatched\ntext\n.EE\n", Options{Strict: true})
	if got := textRuns(byKind(d, Preformatted)[0].Runs); got != "text\n" {
		t.Fatalf("comment rendered as text: %q", got)
	}
	extra, _ := buildTest(t, ".SH SYNOPSIS\n.OP -f file extra\n", Options{})
	if !strings.Contains(extra.Text(1, TextOptions{}), "extra") || len(extra.Diagnostics) == 0 {
		t.Fatal("extra option content silently disappeared")
	}
}

func TestBundledManualFontContracts(t *testing.T) {
	source := ".SH DESCRIPTION\n.ft B\nbold\n.PP\nnormal\n.EX\ncode\n.EE\nrestored\n.BR\nplain\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	paragraphs := byKind(d, Paragraph)
	pre := byKind(d, Preformatted)
	if len(pre) != 1 || !pre[0].Runs[0].Style.Mono {
		t.Fatal("EX did not select mono")
	}
	if !paragraphs[0].Runs[0].Style.Bold || paragraphs[1].Runs[0].Style.Bold || paragraphs[2].Runs[0].Style.Mono || paragraphs[2].Runs[len(paragraphs[2].Runs)-1].Style.Bold {
		t.Fatal("paragraph/example/empty alternation font scope")
	}
	heading, _ := buildTest(t, ".SH\n.IR Title .\nbody\n", Options{Strict: true})
	h := byKind(heading, Section)[0]
	if h.Heading != "Title." || !h.Runs[0].Style.Italic || !h.Runs[0].Style.Bold {
		t.Fatal("styled next-line heading")
	}
	nested, _ := buildTest(t, ".B\n.I styled\nbody\n", Options{})
	if len(nested.Diagnostics) == 0 || byKind(nested, Paragraph)[0].Runs[len(byKind(nested, Paragraph)[0].Runs)-1].Style.Bold {
		t.Fatal("nested next-line style leaked to body")
	}
}

func TestNameAndSynopsisMeaning(t *testing.T) {
	source := ".TH TOOL 1 DATE Linux Manual\n.SH NAME\ntool \\- inspect files\n.SH SYNOPSIS\n.SY tool\n.OP \\-v\n.OP \\-o file\n.I input\n.YS\n"
	d, _ := buildTest(t, source, Options{Strict: true})
	if d.Metadata.Title != "TOOL" || d.Metadata.Section != "1" || d.Metadata.Date != "DATE" || d.Metadata.Source != "Linux" || d.Metadata.Manual != "Manual" {
		t.Fatal("title metadata")
	}
	synopsis, err := d.SelectSection("SYNOPSIS", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := synopsis.Text(TextOptions{Normalize: true}); got != "SYNOPSIS tool [-v] [-o file] input" {
		t.Fatalf("synopsis meanings: %q", got)
	}
}
