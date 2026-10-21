package renderimage

import (
	"image"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"golang.org/x/image/math/fixed"
)

// Fragment identifies measured text. Source differs from Text for visible Unicode
// fallback labels and tab expansion, independently of original roff source.
type Fragment struct {
	Text, Source, Face string
	Style              man.Style
	Link, Reference    string `json:",omitempty"`
	X, Advance         float64
	Bounds             image.Rectangle
}
type Line struct {
	Node                  man.ID
	Kind                  man.Kind
	Top, Baseline, Bottom int
	HardBreak             bool `json:",omitempty"`
	Fragments             []Fragment
}
type Page struct {
	Number, Width, Height int
	Lines                 []Line
}
type Result struct {
	Pages       int
	Pixels      int64
	Bytes       int
	Diagnostics []groff.Diagnostic
}
type glyph struct {
	char            rune
	source          string
	face            faceKey
	style           man.Style
	link, reference string
	x, advance      fixed.Int26_6
	bounds          fixed.Rectangle26_6
}
type paintedLine struct {
	info        Line
	glyphs      []glyph
	baseline    fixed.Int26_6
	left, right int
}
type page struct {
	info  Page
	lines []paintedLine
}

// Plan owns an immutable display list and font bytes. Painting is independent per
// call and may run concurrently; returned page descriptions/images are copies.
type Plan struct {
	o      Options
	pages  []page
	data   map[string][]byte
	result Result
}

func (p *Plan) Pages() []Page {
	out := make([]Page, len(p.pages))
	for i, v := range p.pages {
		out[i] = copyPage(v.info)
	}
	return out
}
func copyPage(v Page) Page {
	v.Lines = append([]Line(nil), v.Lines...)
	for i := range v.Lines {
		v.Lines[i].Fragments = append([]Fragment(nil), v.Lines[i].Fragments...)
	}
	return v
}
func (p *Plan) Result() Result {
	r := p.result
	r.Diagnostics = append([]groff.Diagnostic(nil), r.Diagnostics...)
	return r
}
