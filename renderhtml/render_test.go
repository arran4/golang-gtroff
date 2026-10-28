package renderhtml

import (
	"bytes"
	"encoding/base64"
	"errors"
	"math"
	"regexp"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	xhtml "golang.org/x/net/html"
)

func document(t *testing.T, source string) *man.Document {
	t.Helper()
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := man.Build(tree, man.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func renderTest(t *testing.T, d *man.Document, o Options) (string, Result) {
	t.Helper()
	var b bytes.Buffer
	res, err := Render(&b, d, o)
	if err != nil {
		t.Fatal(err)
	}
	return b.String(), res
}
func parse(t *testing.T, text string) *xhtml.Node {
	t.Helper()
	root, err := xhtml.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func elements(root *xhtml.Node, tag string) []*xhtml.Node {
	var out []*xhtml.Node
	stack := []*xhtml.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n.Type == xhtml.ElementNode && n.Data == tag {
			out = append(out, n)
		}
		for c := n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}
	return out
}
func attr(n *xhtml.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}
func TestSemanticStructureAndFonts(t *testing.T) {
	d := document(t, ".TH DEMO 1\n.SH NAME\ndemo\n.SH OPTIONS\n.TP 6\n.B \\-o\nChoose a\n.I file .\n.SH EXAMPLES\n.EX\n  code  雪\n.EE\n")
	output, res := renderTest(t, d, Options{})
	if len(res.Diagnostics) != 0 || res.Bytes != len(output) {
		t.Fatal("unexpected diagnostics/byte count")
	}
	root := parse(t, output)
	for _, tc := range []struct {
		tag   string
		count int
	}{{"h1", 1}, {"h2", 3}, {"dl", 1}, {"dt", 1}, {"dd", 1}, {"pre", 1}, {"em", 2}} {
		// Heading emphasis contributes strong; exact italic count depends on .I args.
		if got := len(elements(root, tc.tag)); got != tc.count {
			t.Fatalf("%s count %d want %d", tc.tag, got, tc.count)
		}
	}
	if elements(root, "dt")[0].Parent.Data != "dl" || elements(root, "dd")[0].Parent.Data != "dl" {
		t.Fatal("HTML parser repaired definition-list structure")
	}
	pre := elements(root, "pre")[0]
	var preText strings.Builder
	for _, n := range elements(pre, "code") {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == xhtml.TextNode {
				preText.WriteString(c.Data)
			}
		}
	}
	if preText.String() != "  code  雪\n" {
		t.Fatalf("preformatted text %q", preText.String())
	}
	re := regexp.MustCompile(`data:font/(?:ttf|otf);base64,([A-Za-z0-9+/=]+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) != 4 {
		t.Fatalf("bundled faces %d", len(matches))
	}
	fonts := fontset.Default()
	for i, data := range [][]byte{fonts.Regular, fonts.Bold, fonts.Italic, fonts.Mono} {
		decoded, err := base64.StdEncoding.DecodeString(matches[i][1])
		if err != nil || !bytes.Equal(decoded, data) {
			t.Fatal("embedded font differs from bundled face")
		}
	}
	if !strings.Contains(output, "Bigelow &amp;") && !strings.Contains(output, "Bigelow & Holmes") {
		t.Fatal("font license missing")
	}
}
func TestLinksEscapingAndStrict(t *testing.T) {
	source := ".TH \\\"X\\\" 1\n.SH DESCRIPTION\n<script>alert(1)</script> & <img src=x onerror=bad>\n.UR javascript:alert(1)\nunsafe\n.UE\n.UR https://example.invalid/path?a=1&b=2\nsafe\n.UE\n.so /etc/passwd\n"
	d := document(t, source)
	output, res := renderTest(t, d, Options{Assets: OmitFonts, Title: "</title><script>bad</script>"})
	root := parse(t, output)
	if len(elements(root, "script")) != 0 || len(elements(root, "img")) != 0 {
		t.Fatal("source HTML became executable markup")
	}
	found := false
	for _, a := range elements(root, "a") {
		href := attr(a, "href")
		if strings.HasPrefix(href, "javascript:") {
			t.Fatal("unsafe URL active")
		}
		if href == "https://example.invalid/path?a=1&b=2" {
			found = true
		}
	}
	if !found || len(res.Diagnostics) < 2 || !strings.Contains(output, "&lt;script&gt;") {
		t.Fatal("escaping/link diagnostics missing")
	}
	var b bytes.Buffer
	strict, err := Render(&b, d, Options{Strict: true, Assets: OmitFonts})
	if !errors.Is(err, ErrStrict) || b.Len() != 0 || len(strict.Diagnostics) == 0 {
		t.Fatal("strict output mutated destination")
	}
	if d.Source() != source {
		t.Fatal("renderer changed source snapshot")
	}
}
func TestURLPolicy(t *testing.T) {
	for _, s := range []string{"https://example.invalid/x?q=1&n=2", "http://example.invalid/", "mailto:person@example.invalid", "#man-node-2"} {
		if !SafeURL(s) {
			t.Fatalf("safe URL rejected %q", s)
		}
	}
	for _, s := range []string{"javascript:alert(1)", "JaVaScRiPt:alert(1)", "data:text/html,x", "file:///etc/passwd", "//example.invalid", "/local", "https://u:p@example.invalid", "https://example.invalid/%0aheader", "https:\n//example.invalid", "mailto:", "https:", "java%73cript:x", "\x00https://example.invalid"} {
		if SafeURL(s) {
			t.Fatalf("unsafe URL accepted %q", s)
		}
	}
}
func TestSelectionsFragmentsAndContext(t *testing.T) {
	d := document(t, ".TH DEMO 1\n.SH OPTIONS\nfirst\n.SS OPTIONS\nnested\n.SH OTHER\noutside\n.SH OPTIONS\n.RS 4\nchosen\n.RE\n")
	output, _ := renderTest(t, d, Options{Section: "OPTIONS", Fragment: true, Assets: OmitFonts, NoAnchors: true, Theme: "dark"})
	if strings.Contains(output, "<!doctype") || strings.Contains(output, "outside") || strings.Contains(output, "permalink\" href=") {
		t.Fatal("selected fragment leaked document/sibling/anchors")
	}
	root := parse(t, output)
	if len(elements(root, "section")) != 3 || len(elements(root, "h2")) != 2 || len(elements(root, "h3")) != 1 {
		t.Fatal("repeated/nested selection lost content or repeated it")
	}
	var paragraph man.ID
	for _, n := range d.Nodes {
		if n.Kind == man.Paragraph && len(n.Runs) > 0 && n.Runs[0].Text == "chosen" {
			paragraph = n.ID
		}
	}
	selected, _ := renderTest(t, d, Options{Node: paragraph, Fragment: true, Assets: OmitFonts})
	if !strings.Contains(selected, "selection-context\" style=\"margin-inline-start:2em") || strings.Contains(selected, "nested") {
		t.Fatal("selected node lost inherited indent or leaked siblings")
	}
	selection, err := d.SelectRange(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if _, err := RenderSelection(&b, d, selection, Options{Assets: OmitFonts}); err != nil {
		t.Fatal(err)
	}
	if len(elements(parse(t, b.String()), "section")) != 2 {
		t.Fatal("overlapping selected roots repeated/missed content")
	}
}
func TestCustomOptionsBudgetsAndInvalidModels(t *testing.T) {
	d := document(t, ".SH NAME\ntext\n")
	for _, o := range []Options{{Theme: "oops"}, {Assets: "remote"}, {FontSize: math.NaN()}, {LineHeight: math.Inf(1)}, {MaxWidth: 2}, {Palette: Palette{Accent: "red;url(x)"}}, {Fonts: fontset.Set{Regular: []byte("bad")}}, {Section: "NAME", Node: 1}} {
		var b bytes.Buffer
		if _, err := Render(&b, d, o); err == nil || b.Len() != 0 {
			t.Fatal("invalid options wrote output")
		}
	}
	var b bytes.Buffer
	if _, err := Render(&b, d, Options{MaxOutputBytes: 20}); !errors.Is(err, ErrLimit) || b.Len() != 0 {
		t.Fatal("output budget not enforced atomically")
	}
	d.Nodes[0].Children = append(d.Nodes[0].Children, 1)
	if _, err := Render(&b, d, Options{}); err == nil {
		t.Fatal("invalid cyclic model accepted")
	}
}

type badWriter struct{ short bool }

func (w badWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) / 2, nil
	}
	return 0, errors.New("write failed")
}
func TestWriterFailuresAndCustomFonts(t *testing.T) {
	d := document(t, ".SH NAME\ntext\n")
	for _, w := range []badWriter{{}, {short: true}} {
		if _, err := Render(w, d, Options{Assets: OmitFonts}); err == nil {
			t.Fatal("writer failure ignored")
		}
	}
	fonts := fontset.Default()
	fonts.Regular = fonts.Mono
	output, _ := renderTest(t, d, Options{Fonts: fonts})
	encoded := base64.StdEncoding.EncodeToString(fonts.Mono)
	if !strings.Contains(output, "font-weight:400;font-display:swap;src:url(data:font/ttf;base64,"+encoded) {
		t.Fatal("custom font ignored")
	}
}
func FuzzHTMLSource(f *testing.F) {
	for _, s := range []string{"", ".SH NAME\n雪\n", ".UR javascript:x\n<script>text</script>\n.UE", ".TP\n.B opt\nbody", ".EX\ncode\n.EE", ".B \"broken\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 8192 {
			t.Skip()
		}
		tree, _ := groff.ParseAST(strings.NewReader(source), groff.WithASTNodes(4096), groff.WithASTDepth(16))
		d, _ := man.Build(tree, man.Options{MaxNodes: 2048, MaxDepth: 16})
		var b bytes.Buffer
		if _, err := Render(&b, d, Options{Assets: OmitFonts, MaxOutputBytes: 1 << 20}); err != nil {
			if errors.Is(err, ErrLimit) {
				return
			}
			t.Fatal(err)
		}
		root := parse(t, b.String())
		if len(elements(root, "script")) != 0 || len(elements(root, "iframe")) != 0 {
			t.Fatal("active source markup escaped containment")
		}
		for _, a := range elements(root, "a") {
			if !SafeURL(attr(a, "href")) {
				t.Fatal("unsafe emitted link")
			}
		}
	})
}

func TestSelectedBodyUsesAncestorItemWidth(t *testing.T) {
	d := document(t, ".SH OPTIONS\n.TP 8\n.B OUTER\nouter\n.RS 4\n.TP 6\n.B INNER\ninner\n.RE\nafter inner\n")
	var id man.ID
	for _, n := range d.Nodes {
		if n.Kind == man.Paragraph && len(n.Runs) > 0 && n.Runs[0].Text == "after inner" {
			id = n.ID
		}
	}
	output, _ := renderTest(t, d, Options{Node: id, Fragment: true, Assets: OmitFonts})
	if !strings.Contains(output, "selection-context\" style=\"margin-inline-start:4em") {
		t.Fatal("later tag-width state replaced the enclosing item's width")
	}
}

func TestStandaloneSubsectionHasAccessibleHeadingRank(t *testing.T) {
	d := document(t, ".TH DEMO 1\n.SH DESCRIPTION\n.SS DETAILS\ncontent\n")
	output, _ := renderTest(t, d, Options{Section: "DETAILS", Assets: OmitFonts})
	root := parse(t, output)
	if len(elements(root, "h1")) != 1 || len(elements(root, "h2")) != 1 || len(elements(root, "h3")) != 0 {
		t.Fatal("selected subsection skipped a heading level")
	}
}
