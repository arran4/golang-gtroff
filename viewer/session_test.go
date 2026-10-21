package viewer_test

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/renderimage"
	"github.com/arran4/golang-gtroff/viewer"
)

func TestSessionPaintingAndResize(t *testing.T) {
	source := ".TH TOOL 1\n.SH CONTENT\n" + strings.Repeat(".PP\nVisible paragraph.\n", 20) + ".UNKNOWN literal\n"
	s, e := viewer.Load(strings.NewReader(source), "test.1", renderimage.Options{Width: 500, PageHeight: 400, Margin: 24}, image.Pt(700, 500))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.Source() != source {
		t.Fatal("raw source changed during viewer loading")
	}
	if s.Controller.State().Pages < 2 || len(s.Diagnostics()) == 0 {
		t.Fatal("missing pages or diagnostics")
	}
	first, e := s.Frame()
	if e != nil {
		t.Fatal(e)
	}
	if first.Bounds() != image.Rect(0, 0, 700, 500) {
		t.Fatal("wrong viewport")
	}
	_, _ = s.Controller.Apply(viewer.Action{Kind: "next"})
	_, _ = s.Controller.Apply(viewer.Action{Kind: "notes"})
	_, _ = s.Controller.Apply(viewer.Action{Kind: "resize", Width: 900, Height: 600})
	resized, e := s.Frame()
	if e != nil || resized.Bounds() != image.Rect(0, 0, 900, 600) {
		t.Fatal("resize did not replace viewport", e)
	}
	// Overlay and page colors must produce actual nonuniform viewport pixels.
	varied := false
	for i := 4; i < len(resized.Pix); i += 4 {
		if string(resized.Pix[i:i+4]) != string(resized.Pix[:4]) {
			varied = true
			break
		}
	}
	if !varied {
		t.Fatal("blank frame")
	}
	if e := s.Draw(image.NewRGBA(image.Rect(0, 0, 10, 10))); e == nil {
		t.Fatal("wrong destination accepted")
	}
}
func TestControllerLimitsAndAnchor(t *testing.T) {
	c, e := viewer.NewController([]image.Point{{X: 1000, Y: 1400}}, image.Pt(800, 600))
	if e != nil {
		t.Fatal(e)
	}
	_, _ = c.Apply(viewer.Action{Kind: "scroll", DX: 100, DY: 200})
	before := c.Destination()
	sourceX := (400 - float64(before.Min.X)) / c.State().Zoom
	sourceY := (300 - float64(before.Min.Y)) / c.State().Zoom
	_, e = c.Apply(viewer.Action{Kind: "zoom", Factor: 2, X: 400, Y: 300})
	if e != nil {
		t.Fatal(e)
	}
	after := c.Destination()
	if math.Abs((400-float64(after.Min.X))/c.State().Zoom-sourceX) > 1 || math.Abs((300-float64(after.Min.Y))/c.State().Zoom-sourceY) > 1 {
		t.Fatal("zoom lost pointer anchor")
	}
	prior := c.State()
	if _, e = c.Apply(viewer.Action{Kind: "zoom", Factor: math.NaN()}); e == nil || c.State() != prior {
		t.Fatal("NaN changed model")
	}
	if _, e = viewer.NewController([]image.Point{{X: math.MaxInt, Y: math.MaxInt}}, image.Pt(800, 600)); e == nil {
		t.Fatal("huge page accepted")
	}
	if _, e = c.Apply(viewer.Action{Kind: "resize", Width: 4097, Height: 4097}); e == nil {
		t.Fatal("huge viewport accepted")
	}
}
func FuzzController(f *testing.F) {
	f.Add(1000, 1400, 800, 600, 1.1, 400.0, 300.0)
	f.Fuzz(func(t *testing.T, pw, ph, w, h int, factor, x, y float64) {
		c, e := viewer.NewController([]image.Point{{X: pw, Y: ph}}, image.Pt(w, h))
		if e != nil {
			return
		}
		_, _ = c.Apply(viewer.Action{Kind: "zoom", Factor: factor, X: x, Y: y})
		s := c.State()
		if math.IsNaN(s.Zoom) || math.IsNaN(s.PanX) || math.IsNaN(s.PanY) || s.Zoom < .25 || s.Zoom > 4 || s.PanX < 0 || s.PanY < 0 {
			t.Fatal("invalid model")
		}
	})
}
