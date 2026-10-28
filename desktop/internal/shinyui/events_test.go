package shinyui

import (
	"image"
	"testing"

	"github.com/arran4/golang-gtroff/viewer"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

func TestNativeEventMapping(t *testing.T) {
	c, e := viewer.NewController([]image.Point{{X: 1000, Y: 1400}, {X: 1000, Y: 1400}}, image.Pt(800, 600))
	if e != nil {
		t.Fatal(e)
	}
	_, _ = Key(c, key.Event{Code: key.CodeDownArrow, Direction: key.DirPress})
	if c.State().PanY != 40 {
		t.Fatal(c.State())
	}
	_, _ = Key(c, key.Event{Rune: '+', Direction: key.DirPress})
	if c.State().Zoom <= 1 {
		t.Fatal("zoom ignored")
	}
	_, _ = Mouse(c, mouse.Event{Direction: mouse.DirStep, Button: mouse.ButtonWheelDown, Modifiers: key.ModControl, X: 400, Y: 300})
	if c.State().Zoom >= 1.1 {
		t.Fatal("control wheel ignored")
	}
	_, _ = Key(c, key.Event{Rune: 'n', Direction: key.DirPress})
	if c.State().Page != 1 {
		t.Fatal("next ignored")
	}
	_, _ = Mouse(c, mouse.Event{Direction: mouse.DirPress, Button: mouse.ButtonLeft, X: 30, Y: 45})
	if c.State().Page != 0 {
		t.Fatal("toolbar previous ignored")
	}
	_, _ = Key(c, key.Event{Rune: 'f', Direction: key.DirRelease})
	if c.State().FitWidth {
		t.Fatal("release triggered action")
	}
	_, _ = Key(c, key.Event{Code: key.CodeEscape, Direction: key.DirPress})
	if !c.State().Closed {
		t.Fatal("close ignored")
	}
}
