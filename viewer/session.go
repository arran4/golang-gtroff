package viewer

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var ErrSelection = errors.New("viewer selection matched no content")

// Session is used serially by a desktop event loop. It caches only the current
// raster page and paints directly into a viewport-sized destination.
type Session struct {
	Controller   *Controller
	plan         *renderimage.Plan
	title, theme string
	source       string
	notes        []groff.Diagnostic
	face         font.Face
	page         image.Image
	pageIndex    int
	failure      string
}

// Load uses the source-aware parser/semantic/image pipeline without ambient IO.
func Load(r io.Reader, name string, o renderimage.Options, viewport image.Point) (*Session, error) {
	tree, err := groff.ParseAST(r, groff.WithPackage("man"), groff.WithSourceName(name))
	if err != nil {
		return nil, err
	}
	doc, err := man.Build(tree, man.Options{})
	if err != nil {
		return nil, err
	}
	if o.Section != "" {
		if _, e := doc.SelectSection(o.Section, false); e != nil {
			return nil, fmt.Errorf("%w: %v", ErrSelection, e)
		}
	}
	if o.Node != 0 {
		if _, e := doc.SelectNode(o.Node); e != nil {
			return nil, fmt.Errorf("%w: %v", ErrSelection, e)
		}
	}
	plan, err := renderimage.Layout(doc, o)
	if err != nil {
		return nil, err
	}
	title := o.Title
	if title == "" {
		title = doc.Metadata.Title
		if doc.Metadata.Section != "" {
			title += "(" + doc.Metadata.Section + ")"
		}
		if title == "" {
			title = name
		}
	}
	session, err := NewSession(plan, title, o.Theme, viewport)
	if session != nil {
		session.source = tree.Source()
	}
	return session, err
}
func NewSession(plan *renderimage.Plan, title, theme string, viewport image.Point) (*Session, error) {
	if plan == nil {
		return nil, fmt.Errorf("nil image plan")
	}
	if theme == "" {
		theme = "light"
	}
	if theme != "light" && theme != "dark" {
		return nil, fmt.Errorf("theme must be light or dark")
	}
	var sizes []image.Point
	for _, p := range plan.Pages() {
		sizes = append(sizes, image.Pt(p.Width, p.Height))
	}
	c, err := NewController(sizes, viewport)
	if err != nil {
		return nil, err
	}
	parsed, err := opentype.Parse(fontset.Default().Regular)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 10, DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	return &Session{Controller: c, plan: plan, title: title, theme: theme, notes: plan.Result().Diagnostics, face: face, pageIndex: -1}, nil
}
func (s *Session) Close() error { return s.face.Close() }

// Source returns original input bytes for Load sessions; NewSession plans do not carry a raw input snapshot.
func (s *Session) Source() string { return s.source }
func (s *Session) Title() string  { return s.title }
func (s *Session) Diagnostics() []groff.Diagnostic {
	return append([]groff.Diagnostic(nil), s.notes...)
}
func (s *Session) SetError(err error) {
	if err == nil {
		s.failure = ""
	} else {
		s.failure = err.Error()
	}
}
func (s *Session) Frame() (*image.RGBA, error) {
	v := s.Controller.State().Viewport
	if v.X <= 0 || v.Y <= 0 {
		return nil, fmt.Errorf("empty viewport")
	}
	img := image.NewRGBA(image.Rectangle{Max: v})
	return img, s.Draw(img)
}

// Draw never creates a full scaled page; scaling clips directly to the viewport.
func (s *Session) Draw(dst *image.RGBA) error {
	state := s.Controller.State()
	if state.Viewport.X <= 0 || state.Viewport.Y <= 0 || dst == nil || dst.Bounds() != (image.Rectangle{Max: state.Viewport}) {
		return fmt.Errorf("destination does not match viewport")
	}
	bg, fg, bar, button, accent := color.RGBA{219, 223, 228, 255}, color.RGBA{32, 33, 36, 255}, color.RGBA{245, 247, 250, 255}, color.RGBA{226, 232, 240, 255}, color.RGBA{23, 78, 166, 255}
	if s.theme == "dark" {
		bg = color.RGBA{12, 14, 17, 255}
		fg = color.RGBA{241, 243, 244, 255}
		bar = color.RGBA{31, 35, 41, 255}
		button = color.RGBA{48, 55, 65, 255}
		accent = color.RGBA{138, 180, 248, 255}
	}
	fill := func(r image.Rectangle, c color.RGBA) { draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src) }
	fill(dst.Bounds(), bg)
	if s.pageIndex != state.Page {
		img, err := s.plan.Image(state.Page)
		if err != nil {
			s.failure = err.Error()
			s.page = nil
			s.pageIndex = -1
		} else {
			s.page = img
			s.pageIndex = state.Page
		}
	}
	content := s.Controller.Content()
	if s.page != nil && !content.Empty() {
		xdraw.ApproxBiLinear.Scale(dst.SubImage(content).(*image.RGBA), s.Controller.Destination(), s.page, s.page.Bounds(), draw.Src, nil)
	}
	fill(image.Rect(0, 0, state.Viewport.X, ToolbarHeight), bar)
	status := fmt.Sprintf("%s  |  Page %d/%d  |  %.0f%%  |  %d notes", s.title, state.Page+1, state.Pages, state.Zoom*100, len(s.notes))
	s.text(dst, status, 8, 21, fg)
	for _, b := range s.Controller.Buttons() {
		fill(b.Bounds, button)
		s.text(dst, b.Label, b.Bounds.Min.X+5, b.Bounds.Min.Y+20, accent)
	}
	if state.Diagnostics {
		top := content.Max.Y
		fill(image.Rect(0, top, state.Viewport.X, state.Viewport.Y), bar)
		s.text(dst, "Diagnostics (D to close)", 8, top+18, accent)
		y := top + 37
		if len(s.notes) == 0 {
			s.text(dst, "No conversion diagnostics.", 8, y, fg)
		}
		for i, n := range s.notes {
			if y+16 > state.Viewport.Y {
				if i < len(s.notes) {
					s.text(dst, "More details are printed to stderr.", 8, state.Viewport.Y-6, accent)
				}
				break
			}
			s.text(dst, n.Severity+": "+n.Message, 8, y, fg)
			y += 18
		}
	}
	if s.failure != "" {
		fill(image.Rect(0, state.Viewport.Y-32, state.Viewport.X, state.Viewport.Y), color.RGBA{120, 30, 20, 255})
		s.text(dst, "Error: "+s.failure, 8, state.Viewport.Y-11, color.RGBA{255, 255, 255, 255})
	}
	if state.Help {
		panel := image.Rect(12, ToolbarHeight+12, min(state.Viewport.X-12, 660), min(state.Viewport.Y-12, ToolbarHeight+260))
		if !panel.Empty() {
			fill(panel, bar)
			lines := []string{"gtroff-view controls", "Wheel / arrows / H J K L: scroll; left drag: pan", "+ / - or Ctrl-wheel: zoom at pointer/center", "F: fit width (tracks resize); 0: reset zoom and pan", "N / P or [ / ]: next / previous rendered page", "PageDown / Space: scroll screen; PageUp: back", "Home / End: top / bottom; D: diagnostics", "? / F1: help; Q / Escape: close"}
			for i, line := range lines {
				y := panel.Min.Y + 24 + i*25
				if y < panel.Max.Y {
					s.text(dst, line, panel.Min.X+10, y, fg)
				}
			}
		}
	}
	return nil
}
func (s *Session) text(dst *image.RGBA, text string, x, y int, c color.Color) {
	// Source pages already use the renderer's glyph policy. UI labels replace
	// controls and limit work to the viewport, while full diagnostics go to stderr.
	if utf8.RuneCountInString(text) > 512 {
		text = string([]rune(text)[:512])
	}
	text = strings.Map(func(r rune) rune {
		if r < ' ' || r == 127 {
			return ' '
		}
		return r
	}, text)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "�")
	}
	d := font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: s.face, Dot: fixed.P(x, y)}
	for _, r := range text {
		advance, _ := s.face.GlyphAdvance(r)
		if d.Dot.X+advance > fixed.I(dst.Bounds().Max.X-4) {
			break
		}
		d.DrawString(string(r))
	}
}
