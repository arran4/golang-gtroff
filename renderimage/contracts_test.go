package renderimage_test

import (
	"bytes"
	"errors"
	"image"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
)

func TestGeometryAndBudgets(t *testing.T) {
	d := document(t, ".SH TEXT\n"+strings.Repeat(".PP\nvisible words\n", 80))
	for _, o := range []renderimage.Options{{Width: -1}, {Width: math.MaxInt}, {Margin: math.MaxInt}, {PageHeight: math.MaxInt}, {FontSize: math.NaN()}, {DPI: math.Inf(1)}, {LineHeight: -1}, {Quality: 101}, {MaxPixels: math.MaxInt64}, {MaxPages: -1}, {MaxGlyphs: -1}, {MaxOutputBytes: -1}, {Width: 128, Margin: 60}, {PageHeight: 128, Margin: 60}, {Theme: "blue"}} {
		if e := renderimage.ValidateOptions(o); e == nil {
			t.Fatalf("bad options accepted %+v", o)
		}
	}
	for _, o := range []renderimage.Options{{Width: 400, PageHeight: 300, MaxPages: 1}, {Width: 400, PageHeight: 300, MaxPixels: 120000}, {Single: true, MaxHeight: 150}, {MaxGlyphs: 2}} {
		var out bytes.Buffer
		_, e := renderimage.Render(&out, d, o)
		if !errors.Is(e, renderimage.ErrLimit) || out.Len() != 0 {
			t.Fatalf("budget error/output %+v: %v %d", o, e, out.Len())
		}
	}
	var out bytes.Buffer
	_, e := renderimage.Render(&out, d, renderimage.Options{Width: 400, PageHeight: 300})
	if !errors.Is(e, renderimage.ErrMultiplePages) || out.Len() != 0 {
		t.Fatal("multi-page stdout not rejected", e)
	}
	unknown := document(t, ".UNKNOWN literal\n")
	_, e = renderimage.Render(&out, unknown, renderimage.Options{Strict: true})
	if !errors.Is(e, renderimage.ErrStrict) || out.Len() != 0 {
		t.Fatal("strict wrote", e)
	}
	p, e := renderimage.Layout(document(t, ".SH SHORT\ntext\n"), renderimage.Options{MaxOutputBytes: 2})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.WritePage(&out, 0); !errors.Is(e, renderimage.ErrLimit) || out.Len() != 0 {
		t.Fatal("encoded budget wrote output", e)
	}
}
func TestPageDescriptionsAndFontsAreOwned(t *testing.T) {
	d := document(t, ".SH TEXT\n\\f[BI]Both\\f[R] \\f[CBI]mono\\f[R].\n")
	custom := fontset.Default()
	p, e := renderimage.Layout(d, renderimage.Options{Fonts: custom, Single: true})
	if e != nil {
		t.Fatal(e)
	}
	pages := p.Pages()
	before := p.Pages()
	pages[0].Lines[0].Fragments[0].Text = "mutated"
	for i := range custom.Regular {
		custom.Regular[i] = 0
	}
	if !reflect.DeepEqual(before, p.Pages()) {
		t.Fatal("page snapshot mutated plan")
	}
	if _, e = p.Image(0); e != nil {
		t.Fatal("caller font mutation changed plan", e)
	}
	result := p.Result()
	if len(result.Diagnostics) > 0 {
		result.Diagnostics[0].Message = "mutated"
	}
	if reflect.DeepEqual(result, p.Result()) && len(result.Diagnostics) > 0 {
		t.Fatal("diagnostics shared")
	}
	if _, e = renderimage.Layout(d, renderimage.Options{Fonts: fontset.Set{Regular: []byte("bad")}}); e == nil {
		t.Fatal("invalid font accepted")
	}
	if _, e = p.Image(-1); e == nil {
		t.Fatal("bad page index accepted")
	}
}
func TestSelectionsRetainActualIndent(t *testing.T) {
	d := document(t, ".TH TOOL 1\n.SH OPTIONS\n.TP 8\n.B --flag\nselected body\n.TP 2\n.B --other\nother body\n")
	var id man.ID
	for _, n := range d.Nodes {
		if n.Kind == man.Paragraph && strings.Contains(d.Text(n.ID, man.TextOptions{}), "selected body") {
			id = n.ID
		}
	}
	p, e := renderimage.Layout(d, renderimage.Options{Node: id, Single: true, Width: 500, Margin: 24})
	if e != nil {
		t.Fatal(e)
	}
	for _, pg := range p.Pages() {
		for _, l := range pg.Lines {
			if l.Node == id && l.Fragments[0].X < 104 {
				t.Fatal("selected body lost 8n/4em context", l)
			}
		}
	}
}

var errSink = errors.New("sink stopped")

type shortImageWriter struct{}

func (shortImageWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }
func TestWriterAndSinkErrors(t *testing.T) {
	d := document(t, ".SH TEXT\ntext\n")
	p, e := renderimage.Layout(d, renderimage.Options{Single: true})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.WritePage(shortImageWriter{}, 0); !errors.Is(e, io.ErrShortWrite) {
		t.Fatal(e)
	}
	if _, e = p.Encode(func(renderimage.Page, []byte) error { return errSink }); !errors.Is(e, errSink) {
		t.Fatal(e)
	}
	if _, e = p.Stream(func(renderimage.Page, image.Image) error { return errSink }); !errors.Is(e, errSink) {
		t.Fatal(e)
	}
	if _, e = p.WritePage(nil, 0); e == nil {
		t.Fatal("nil writer accepted")
	}
	if _, e = p.Encode(nil); e == nil {
		t.Fatal("nil sink accepted")
	}
}
func TestOutputNames(t *testing.T) {
	for _, tc := range []struct {
		path, format string
		n            int
		want         []string
	}{{"foo.png", "png", 1, []string{"foo.png"}}, {"foo.png", "png", 3, []string{"foo-001.png", "foo-002.png", "foo-003.png"}}, {"foo.JPEG", "jpg", 2, []string{"foo-001.JPEG", "foo-002.JPEG"}}} {
		got, e := renderimage.OutputPaths(tc.path, tc.format, tc.n)
		if e != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatal(got, e)
		}
	}
	if _, e := renderimage.OutputPaths("-", "png", 2); !errors.Is(e, renderimage.ErrMultiplePages) {
		t.Fatal(e)
	}
	if _, e := renderimage.OutputPaths("foo.jpg", "png", 1); e == nil {
		t.Fatal("wrong extension accepted")
	}
}
func FuzzImageLayout(f *testing.F) {
	f.Add(".SH TEXT\nVisible words\n", 400, 300, 24, 15.0, 96.0)
	f.Add(".SH TEXT\n漢字\n", math.MaxInt, math.MaxInt, math.MaxInt, math.Inf(1), 96.0)
	f.Fuzz(func(t *testing.T, source string, width, height, margin int, size, dpi float64) {
		if len(source) > 2048 {
			t.Skip()
		}
		o := renderimage.Options{Width: width, PageHeight: height, Margin: margin, FontSize: size, DPI: dpi, MaxPixels: 1000000, MaxPages: 4, MaxGlyphs: 3000}
		if e := renderimage.ValidateOptions(o); e != nil {
			return
		}
		tree, e := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"), groff.WithASTBytes(2048))
		if e != nil {
			return
		}
		d, e := man.Build(tree, man.Options{})
		if e != nil {
			return
		}
		p, e := renderimage.Layout(d, o)
		if e != nil {
			return
		}
		if p.Result().Pixels > 1000000 {
			t.Fatal("pixel budget escaped")
		}
		for _, pg := range p.Pages() {
			for _, l := range pg.Lines {
				if l.Top < 0 || l.Bottom > pg.Height {
					t.Fatal("bad layout bounds")
				}
			}
		}
		if _, e = p.Image(0); e != nil {
			t.Fatal(e)
		}
	})
}

func TestParagraphDistanceAndHardBreaks(t *testing.T) {
	d := document(t, ".PD 0\n.PP\nfirst\n.PP\nsecond\n.br\n.br\nthird\n")
	p, e := renderimage.Layout(d, renderimage.Options{Single: true})
	if e != nil {
		t.Fatal(e)
	}
	lines := p.Pages()[0].Lines
	if len(lines) != 4 {
		t.Fatalf("line count %d %+v", len(lines), lines)
	}
	if lines[1].Top != lines[0].Bottom {
		t.Fatal("PD 0 did not suppress paragraph gap", lines[0], lines[1])
	}
	if !lines[1].HardBreak || !lines[2].HardBreak {
		t.Fatal("repeated breaks lost")
	}
}

func TestDPIAndJPEGQuality(t *testing.T) {
	d := document(t, ".PP\nMeasured text with several words.\n")
	low, e := renderimage.Layout(d, renderimage.Options{Single: true, DPI: 96, Format: "jpg", Quality: 1})
	if e != nil {
		t.Fatal(e)
	}
	high, e := renderimage.Layout(d, renderimage.Options{Single: true, DPI: 192})
	if e != nil {
		t.Fatal(e)
	}
	a := low.Pages()[0].Lines[0]
	b := high.Pages()[0].Lines[0]
	if b.Bottom-b.Top < 2*(a.Bottom-a.Top)-3 {
		t.Fatal("point size did not scale with DPI")
	}
	maxQuality, e := renderimage.Layout(d, renderimage.Options{Single: true, Format: "jpg", Quality: 100})
	if e != nil {
		t.Fatal(e)
	}
	var cheap, full bytes.Buffer
	if _, e = low.WritePage(&cheap, 0); e != nil {
		t.Fatal(e)
	}
	if _, e = maxQuality.WritePage(&full, 0); e != nil {
		t.Fatal(e)
	}
	if full.Len() <= cheap.Len() {
		t.Fatal("JPEG quality ignored")
	}
}
