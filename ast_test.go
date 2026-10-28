package groff

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func mustAST(t *testing.T, source string, opts ...Option) *AST {
	t.Helper()
	a, err := ParseAST(strings.NewReader(source), append([]Option{WithSourceName("input.1"), WithPackage("man")}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestASTSearchSelectionAndWalk(t *testing.T) {
	source := ".TH TOOL 1\n.SH NAME\n雪  Café\n.SS Details\nbody\n.SH NAME\nsecond\n"
	a := mustAST(t, source)
	verifyASTIntegrity(t, a, source)
	root := a.Root()
	headings := root.FindSection("name", true)
	if len(headings) != 2 || len(root.FindSection("name", false)) != 0 {
		t.Fatal("repeated/case-sensitive headings")
	}
	if got := headings[0].Text(); got != "NAME\n雪  Café\nDetails\nbody\n" {
		t.Fatalf("selected text %q", got)
	}
	if strings.Contains(headings[0].Raw(), "second") {
		t.Fatal("section selection leaked next section")
	}
	if got := root.FindMacro("SH"); len(got) != 2 || got[0].ID() >= got[1].ID() {
		t.Fatal("macro order")
	}
	if len(root.FindKind(ASTSubsection)) != 1 || len(root.FindText("雪 café", true)) != 2 || len(root.FindText("雪 café", false)) != 0 {
		t.Fatal("content normalization/kind search")
	}
	offset := strings.Index(source, "雪")
	hits := root.FindSpan(offset, offset+3)
	if len(hits) != 4 || hits[2].Kind() != ASTLine || hits[3].Kind() != ASTInline {
		t.Fatalf("span lookup %#v", hits)
	}
	if got := root.FindSpan(len(source), len(source)); len(got) != 0 {
		t.Fatal("EOF should not contain a byte")
	}
	for _, pair := range [][2]int{{-1, 0}, {3, 2}, {0, len(source) + 1}} {
		if len(root.FindSpan(pair[0], pair[1])) != 0 {
			t.Fatal("invalid span matched")
		}
	}
	var walked []NodeID
	root.Walk(func(n *ASTNode) WalkAction {
		walked = append(walked, n.ID())
		if n == headings[0] {
			return WalkPrune
		}
		return WalkContinue
	})
	for _, n := range headings[0].Descendants() {
		for _, id := range walked {
			if id == n.ID() {
				t.Fatal("prune visited descendant")
			}
		}
	}
	visits := 0
	root.Walk(func(n *ASTNode) WalkAction { visits++; return WalkStop })
	if visits != 1 {
		t.Fatal("stop ignored")
	}
	if headings[0].Path() != "/1" || headings[0].NextSibling() != headings[1] || headings[1].PreviousSibling() != headings[0] {
		t.Fatal("paths or siblings")
	}
	last := root.FindKind(ASTSubsection)[0]
	anc := last.Ancestors()
	if len(anc) != 2 || anc[0] != headings[0] || anc[1] != root {
		t.Fatal("ancestor order")
	}
}

func TestASTInspectionCannotMutateTree(t *testing.T) {
	a := mustAST(t, ".SH NAME\n.B \"quoted text\"\n雪 \\fBbold\\fR\n")
	before, _ := json.Marshal(a)
	n := a.Root().FindMacro("B")[0]
	tok, err := n.Token()
	if err != nil {
		t.Fatal(err)
	}
	bold, ok := tok.(*TokenManB)
	if !ok {
		t.Fatalf("typed token %T", tok)
	}
	if bold.Text != "quoted text" {
		t.Fatal("typed field")
	}
	bold.Text = "changed"
	tok2, err := n.Token()
	if err != nil || tok2.(*TokenManB).Text != "quoted text" {
		t.Fatal("token was not detached")
	}
	children := a.Root().Children()
	children[0] = nil
	attrs := n.Attributes()
	attrs.Arguments[0].Value = "changed"
	diag := a.Diagnostics()
	if len(diag) > 0 {
		diag[0].Message = "changed"
	}
	in := a.Root().FindKind(ASTInline)[0]
	v, ok := in.Inline()
	if !ok {
		t.Fatal("no inline inspection")
	}
	v.Raw = "changed"
	after, _ := json.Marshal(a)
	if !bytes.Equal(before, after) {
		t.Fatal("accessor mutated AST")
	}
	if tok, err := a.Root().Token(); err != nil || tok != nil {
		t.Fatal("container has line token")
	}
	if _, ok := a.Root().Inline(); ok {
		t.Fatal("container has inline token")
	}
}

type forbiddenASTResolver struct{ t *testing.T }

func (r forbiddenASTResolver) Resolve(string) (io.Reader, error) {
	r.t.Fatal("AST resolved an include")
	return nil, nil
}
func TestASTDoesNotEvaluateOrInclude(t *testing.T) {
	source := ".so /etc/passwd\n.mso remote\n.nr X 7\n\\n[X]\n.de LOOP\n.LOOP\n..\n.LOOP\n"
	a := mustAST(t, source, WithIncludeResolver(forbiddenASTResolver{t}))
	if a.Source() != source || !strings.Contains(a.Root().Text(), "\\n[X]") {
		t.Fatal("syntax was evaluated")
	}
}

type failingASTReader struct{ sent bool }

var errASTReader = errors.New("source reader failed")

func (r *failingASTReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, errASTReader
	}
	r.sent = true
	return copy(p, ".TS\n雪"), errASTReader
}
func TestASTReaderFailurePreservesReadBytes(t *testing.T) {
	a, err := ParseAST(&failingASTReader{}, WithSourceName("input.1"))
	if !errors.Is(err, errASTReader) {
		t.Fatalf("reader error %v", err)
	}
	verifyASTIntegrity(t, a, ".TS\n雪")
}

func TestASTBudgets(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		opt         Option
	}{
		{"bytes", "abcdefghijk", WithASTBytes(4)},
		{"lines", "a\nb\nc\nd\n", WithMaxLines(2)},
		{"line length", ".B abcdefghijk\n", WithMaxLineLength(4)},
		{"depth", strings.Repeat(".RS\n", 200) + "body\n" + strings.Repeat(".RE\n", 200), WithASTDepth(4)},
		{"nodes", strings.Repeat("a\\fBb\\fR\n", 100), WithASTNodes(20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, err := ParseAST(strings.NewReader(tc.input), tc.opt)
			if err == nil || a == nil || len(a.Diagnostics()) == 0 {
				t.Fatal("budget did not report failure")
			}
			if tc.name != "bytes" && a.Source() != tc.input {
				t.Fatal("budget discarded buffered source")
			}
			if a.Root().Raw() != a.Source() {
				t.Fatal("budget source not navigable")
			}
			if tc.name == "bytes" && a.Source() != tc.input[:5] {
				t.Fatal("read beyond bounded sentinel")
			}
			if tc.name == "nodes" && len(a.nodes) > 21 {
				t.Fatalf("node bound exceeded: %d", len(a.nodes))
			}
		})
	}
}

func TestASTRealManPagesAndDocumentTokens(t *testing.T) {
	for _, path := range []string{"testdata/simple.1", "testdata/environ_snippet.7", "testdata/complex.7"} {
		t.Run(path, func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			a := mustAST(t, string(source))
			verifyASTIntegrity(t, a, string(source))
			doc, err := ParseDocument(bytes.NewReader(source), WithPackage("man"))
			if err != nil {
				t.Fatal(err)
			}
			// Legacy multiline tokens cover several physical AST lines. Every token start
			// must coincide with a physical line, and every line has a token range owner.
			lines := a.Root().FindKind(ASTLine)
			starts := map[int]bool{}
			for _, n := range lines {
				starts[n.Span().Start.Offset] = true
			}
			next := 0
			accountedEnd := 0
			for _, tok := range doc.Tokens {
				if tok.Span.Start.Offset != accountedEnd {
					t.Fatal("gap/overlap between legacy token ranges")
				}
				accountedEnd = tok.Span.End.Offset
				if !starts[tok.Span.Start.Offset] {
					t.Fatalf("legacy token has no AST line: %+v", tok.Span)
				}
				for next < len(lines) && lines[next].Span().Start.Offset < tok.Span.End.Offset {
					next++
				}
			}
			if next != len(lines) {
				t.Fatal("legacy token traversal did not account for all AST lines")
			}
		})
	}
}

func FuzzParseAST(f *testing.F) {
	for _, s := range []string{"", ".TS\nx", ".de X\n.de Y END\n雪\n.END\n..", ".SH \"broken\n", "雪 \\*[missing", "\xff\xfe\n", "'CUSTOM Ω\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 32768 {
			t.Skip()
		}
		a, _ := ParseAST(strings.NewReader(source), WithASTBytes(32768), WithASTDepth(16), WithASTNodes(4096), WithSourceName("input.1"))
		if a.Source() != source || a.Root().Raw() != source {
			t.Fatal("lost raw input")
		}
		var raw strings.Builder
		a.Root().Walk(func(n *ASTNode) WalkAction {
			if n.Kind() == ASTLine {
				raw.WriteString(n.Raw())
			}
			return WalkContinue
		})
		if raw.String() != source {
			t.Fatal("lost physical input")
		}
		if _, err := json.Marshal(a); err != nil {
			t.Fatal(err)
		}
	})
}

func TestASTTextBoundaries(t *testing.T) {
	for _, source := range []string{"", "雪", "雪\n", "雪\r\n", "\r\n", "a\r\r\n", "trailing\\"} {
		a := mustAST(t, source)
		verifyASTIntegrity(t, a, source)
		if got := a.Root().Text(); got != source {
			t.Fatalf("text ending: got %q want %q", got, source)
		}
	}
}

func TestASTSourcePositionsHaveIndependentExpectedValues(t *testing.T) {
	a := mustAST(t, "雪\tΩ\nlast", WithStrict(true))
	lines := a.Root().FindKind(ASTLine)
	if s := lines[0].Span(); s.Start.Offset != 0 || s.End.Offset != 7 || s.Start.Line != 1 || s.End.Line != 2 || s.End.Column != 1 {
		t.Fatalf("first physical span: %+v", s)
	}
	if s := lines[1].Span(); s.Start.Offset != 7 || s.Start.Line != 2 || s.Start.Column != 1 || s.End.Offset != 11 || s.End.Line != 2 || s.End.Column != 5 {
		t.Fatalf("last physical span: %+v", s)
	}
	inlines := a.Root().FindKind(ASTInline)
	if s := inlines[0].Span(); s.End.Offset != 6 || s.End.Line != 1 || s.End.Column != 4 {
		t.Fatalf("inline Unicode/tab span: %+v", s)
	}
	// MaxInt is a valid large byte budget, not an overflowing negative read limit.
	maxInt := int(^uint(0) >> 1)
	b := mustAST(t, "still present", WithASTBytes(maxInt))
	if b.Source() != "still present" {
		t.Fatal("overflow dropped source")
	}
}

func TestASTMalformedBlockHeaderIsNotSuppressed(t *testing.T) {
	source := ".de\nbody\n..\n"
	a, err := ParseAST(strings.NewReader(source), WithStrict(true))
	if err == nil || len(a.Diagnostics()) == 0 || a.Source() != source {
		t.Fatal("missing macro name was silently accepted")
	}
}
