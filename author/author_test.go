package author_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/author"
	"github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/man"
)

func page(t *testing.T) *author.Page {
	t.Helper()
	p, err := author.NewManPage("tool", "1", author.WithDate("2026-11-04"), author.WithSource("Project"), author.WithManual("User Commands"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestLiteralGenerationAndSemanticLoops(t *testing.T) {
	p := page(t)
	if err := p.SetTitle(`Tool "quoted" \ path`); err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddSection("NAME", author.Paragraph("tool - example")); err != nil {
		t.Fatal(err)
	}
	section, err := p.AddSection("OPTIONS", author.Definition(`--quote " \`, "A literal \\fBword\\fP and 漢字 café."), author.Definition("--other", "Second."), author.Subsection("DETAIL", author.Paragraph("Multiple\nsoft\tlines.")), author.LiteralRoff(".so never-read\n.sy never-run\n'apostrophe\n\\literal\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := section.AddParagraph("After the subsection."); err != nil {
		t.Fatal(err)
	}
	var out, again bytes.Buffer
	if err := p.WriteRoff(&out); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteRoff(&again); err != nil {
		t.Fatal(err)
	}
	if out.String() != again.String() {
		t.Fatal("generation was not deterministic")
	}
	tree, err := p.AST()
	if err != nil || tree.Source() != out.String() || !tree.SourceComplete() || len(tree.Diagnostics()) != 0 {
		t.Fatal("generated syntax contract", err)
	}
	tree.Root().Walk(func(node *groff.ASTNode) groff.WalkAction {
		if node.Kind() == groff.ASTLine && (node.Attributes().Name == ".so" || node.Attributes().Name == ".sy") {
			t.Fatal("literal controls became active")
		}
		return groff.WalkContinue
	})
	first, err := convert.ImportMan(bytes.NewReader(out.Bytes()), convert.Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if first.Document.Metadata.Title != p.Metadata().Title {
		t.Fatal("quoted metadata changed", first.Document.Metadata)
	}
	if !strings.Contains(first.Document.Text(1, man.TextOptions{}), `.so never-read`) {
		t.Fatal("literal source disappeared")
	}
	var canonical bytes.Buffer
	if _, err := convert.WriteMan(&canonical, first.Document, convert.Options{Strict: true}); err != nil {
		t.Fatal(err)
	}
	second, err := convert.ImportMan(bytes.NewReader(canonical.Bytes()), convert.Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	a, err := convert.Normalize(first.Document)
	if err != nil {
		t.Fatal(err)
	}
	b, err := convert.Normalize(second.Document)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("supported semantic loop changed\n%#v\n%#v\n%s", a, b, canonical.String())
	}
	var stable bytes.Buffer
	if _, err := convert.WriteMan(&stable, second.Document, convert.Options{Strict: true}); err != nil || stable.String() != canonical.String() {
		t.Fatal("canonical output was not stable", err)
	}
}
func TestConstructionErrorsDoNotMutatePage(t *testing.T) {
	for _, fields := range [][2]string{{"", "1"}, {"tool", ""}, {"bad\n.TH", "1"}, {"bad\xff", "1"}, {"bad\x00", "1"}} {
		if _, err := author.NewManPage(fields[0], fields[1]); !errors.Is(err, author.ErrInvalid) {
			t.Fatal("invalid header accepted", err)
		}
	}
	if _, err := author.NewManPage("tool", "1", nil); !errors.Is(err, author.ErrInvalid) {
		t.Fatal("nil option accepted")
	}
	p := page(t)
	section, err := p.AddSection("GOOD", author.Paragraph("Before."))
	if err != nil {
		t.Fatal(err)
	}
	var before bytes.Buffer
	if err := p.WriteRoff(&before); err != nil {
		t.Fatal(err)
	}
	for _, block := range []author.Block{{}, author.Paragraph("invalid\xff"), author.Definition("bad\nterm", "Description."), author.Definition("", "Description."), author.Preformatted("bare\rcarriage"), author.Paragraph("NUL\x00"), author.Subsection("A", author.Subsection("B", author.Paragraph("nested")))} {
		if err := section.Append(block); !errors.Is(err, author.ErrInvalid) {
			t.Fatal("invalid block appended", err)
		}
	}
	if err := p.SetDate("bad\nvalue"); !errors.Is(err, author.ErrInvalid) {
		t.Fatal("invalid setter accepted")
	}
	if _, err := p.AddSection(""); !errors.Is(err, author.ErrInvalid) {
		t.Fatal("empty heading accepted")
	}
	var after bytes.Buffer
	if err := p.WriteRoff(&after); err != nil || before.String() != after.String() {
		t.Fatal("failed mutation changed page", err)
	}
}
func TestLimitsAndWriterFailures(t *testing.T) {
	p, err := author.NewManPage("T", "1", author.WithLimits(author.Limits{MaxBlocks: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddSection("X", author.Paragraph("Text")); !errors.Is(err, author.ErrLimit) {
		t.Fatal("block cap ignored")
	}
	section, err := p.AddSection("X")
	if err != nil {
		t.Fatal(err)
	}
	if err := section.AddParagraph("Text"); !errors.Is(err, author.ErrLimit) {
		t.Fatal("append cap ignored")
	}
	p, err = author.NewManPage("T", "1", author.WithLimits(author.Limits{MaxBytes: 20}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.AddSection("X", author.Paragraph("T")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := p.WriteRoff(&out); !errors.Is(err, author.ErrLimit) || out.Len() != 0 {
		t.Fatal("failed generation touched output", err)
	}
	p = page(t)
	if err := p.WriteRoff(shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal("short write lost", err)
	}
	sentinel := errors.New("writer failed")
	if err := p.WriteRoff(failWriter{sentinel}); !errors.Is(err, sentinel) {
		t.Fatal("writer error lost")
	}
	var nilBuffer *bytes.Buffer
	if err := p.WriteRoff(nilBuffer); !errors.Is(err, author.ErrInvalid) {
		t.Fatal("typed nil accepted")
	}
}
func TestSubsectionSnapshotAndCRLF(t *testing.T) {
	p := page(t)
	children := []author.Block{author.Paragraph("Original.")}
	sub := author.Subsection("DETAIL", children...)
	children[0] = author.Paragraph("CHANGED")
	s, err := p.AddSection("MAIN", sub, author.Preformatted("line one\r\n.line two\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Heading() != "MAIN" {
		t.Fatal("heading")
	}
	metadata := p.Metadata()
	metadata.Title = "CHANGED"
	tree, err := p.AST()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tree.Source(), "CHANGED") || strings.Contains(tree.Source(), "\r") {
		t.Fatal("construction data aliased or CRLF was not canonicalized")
	}
	model, err := man.Build(tree, man.Options{Strict: true})
	if err != nil || !strings.Contains(model.Text(1, man.TextOptions{}), "line one\n.line two") {
		t.Fatal("preformatted literal layout lost", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return 0, nil }

type failWriter struct{ err error }

func (w failWriter) Write(data []byte) (int, error) { return 0, w.err }
