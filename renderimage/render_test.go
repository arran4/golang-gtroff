package renderimage_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
)

func document(t *testing.T, source string) *man.Document {
	t.Helper()
	tree, e := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if e != nil {
		t.Fatal(e)
	}
	d, e := man.Build(tree, man.Options{})
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestPaintAndPagination(t *testing.T) {
	d := document(t, ".TH TOOL 1\n.SH NAME\ntool \\- inspector\n.SH DESCRIPTION\n"+strings.Repeat(".PP\nA paragraph with \\fBbold\\fR and \\fIitalic\\fR words.\n", 30)+".TP 8\n.B --flag\nbody\n.EX\n  a\n\tb\n.EE\n")
	for _, format := range []string{"png", "jpg"} {
		t.Run(format, func(t *testing.T) {
			p, e := renderimage.Layout(d, renderimage.Options{Width: 600, PageHeight: 600, Margin: 32, Format: format})
			if e != nil {
				t.Fatal(e)
			}
			if len(p.Pages()) < 2 {
				t.Fatal("no pagination")
			}
			if len(p.Result().Diagnostics) > 0 {
				t.Fatal(p.Result().Diagnostics)
			}
			count := 0
			_, e = p.Encode(func(pg renderimage.Page, data []byte) error {
				count++
				var img image.Image
				var err error
				if format == "png" {
					img, err = png.Decode(bytes.NewReader(data))
				} else {
					img, err = jpeg.Decode(bytes.NewReader(data))
				}
				if err != nil {
					t.Fatal(err)
				}
				if img.Bounds().Dx() != 600 || img.Bounds().Dy() != 600 {
					t.Fatal("bounds")
				}
				for _, l := range pg.Lines {
					if l.Top < 32 || l.Bottom > 568 {
						t.Fatalf("line cropped %+v", l)
					}
					for _, f := range l.Fragments {
						if !f.Bounds.Empty() && !f.Bounds.In(image.Rect(32, 32, 568, 568)) {
							t.Fatalf("fragment cropped %+v", f)
						}
					}
				}
				return nil
			})
			if e != nil {
				t.Fatal(e)
			}
			if count != len(p.Pages()) {
				t.Fatal("page missing")
			}
		})
	}
}
func TestUnicodeFallbackAndLiteralSource(t *testing.T) {
	d := document(t, ".SH TEXT\ncafé Ελληνικά 漢字 🚀\n.unknown LITERAL\n")
	p, e := renderimage.Layout(d, renderimage.Options{Single: true})
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Result().Diagnostics) < 3 {
		t.Fatal("missing diagnostics")
	}
	var text strings.Builder
	for _, pg := range p.Pages() {
		for _, l := range pg.Lines {
			for _, f := range l.Fragments {
				text.WriteString(f.Text)
			}
		}
	}
	for _, s := range []string{"café", "Ελληνικά", "[U+6F22]", "[U+1F680]", ".unknown LITERAL"} {
		if !strings.Contains(text.String(), s) {
			t.Fatal("missing ", s, text.String())
		}
	}
	if _, e = p.Image(0); e != nil {
		t.Fatal(e)
	}
}
