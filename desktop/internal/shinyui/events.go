package shinyui

import (
	"github.com/arran4/golang-gtroff/viewer"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/mouse"
)

// Key translates native keyboard events to deterministic controller actions.
func Key(c *viewer.Controller, e key.Event) (bool, error) {
	if e.Direction != key.DirPress && e.Direction != key.DirNone {
		return false, nil
	}
	content := c.Content()
	a := viewer.Action{X: float64(content.Min.X + content.Dx()/2), Y: float64(content.Min.Y + content.Dy()/2)}
	switch e.Code {
	case key.CodeEscape:
		a.Kind = "close"
	case key.CodeDownArrow:
		a.Kind = "scroll"
		a.DY = 40
	case key.CodeUpArrow:
		a.Kind = "scroll"
		a.DY = -40
	case key.CodeLeftArrow:
		a.Kind = "scroll"
		a.DX = -40
	case key.CodeRightArrow:
		a.Kind = "scroll"
		a.DX = 40
	case key.CodePageDown:
		a.Kind = "scroll"
		a.DY = float64(max(40, content.Dy()-40))
		if e.Modifiers&key.ModControl != 0 {
			a.Kind = "next"
		}
	case key.CodePageUp:
		a.Kind = "scroll"
		a.DY = -float64(max(40, content.Dy()-40))
		if e.Modifiers&key.ModControl != 0 {
			a.Kind = "previous"
		}
	case key.CodeHome:
		a.Kind = "home"
	case key.CodeEnd:
		a.Kind = "end"
	case key.CodeF1:
		a.Kind = "help"
	default:
		switch e.Rune {
		case 'q', 'Q':
			a.Kind = "close"
		case 'j', 'J':
			a.Kind = "scroll"
			a.DY = 40
		case 'k', 'K':
			a.Kind = "scroll"
			a.DY = -40
		case 'h', 'H':
			a.Kind = "scroll"
			a.DX = -40
		case 'l', 'L':
			a.Kind = "scroll"
			a.DX = 40
		case 'n', 'N', ']':
			a.Kind = "next"
		case 'p', 'P', '[':
			a.Kind = "previous"
		case '+', '=':
			a.Kind = "zoom"
			a.Factor = 1.1
		case '-', '_':
			a.Kind = "zoom"
			a.Factor = 1 / 1.1
		case 'f', 'F':
			a.Kind = "fit"
		case '0':
			a.Kind = "reset"
		case 'd', 'D':
			a.Kind = "notes"
		case '?':
			a.Kind = "help"
		case ' ':
			a.Kind = "scroll"
			a.DY = float64(max(40, content.Dy()-40))
			if e.Modifiers&key.ModShift != 0 {
				a.DY = -a.DY
			}
		default:
			return false, nil
		}
	}
	return c.Apply(a)
}
func Mouse(c *viewer.Controller, e mouse.Event) (bool, error) {
	x, y := float64(e.X), float64(e.Y)
	if e.Direction == mouse.DirStep {
		a := viewer.Action{Kind: "scroll", X: x, Y: y}
		switch e.Button {
		case mouse.ButtonWheelUp:
			a.DY = -48
		case mouse.ButtonWheelDown:
			a.DY = 48
		case mouse.ButtonWheelLeft:
			a.DX = -48
		case mouse.ButtonWheelRight:
			a.DX = 48
		default:
			return false, nil
		}
		if e.Modifiers&(key.ModControl|key.ModMeta) != 0 {
			a.Kind = "zoom"
			a.Factor = 1.1
			if a.DY > 0 || a.DX > 0 {
				a.Factor = 1 / 1.1
			}
		} else if e.Modifiers&key.ModShift != 0 && a.DY != 0 {
			a.DX = a.DY
			a.DY = 0
		}
		return c.Apply(a)
	}
	if e.Button == mouse.ButtonLeft && e.Direction == mouse.DirPress {
		return c.Click(x, y)
	}
	if e.Button == mouse.ButtonLeft && e.Direction == mouse.DirRelease {
		return c.Apply(viewer.Action{Kind: "drag-end"})
	}
	if e.Direction == mouse.DirNone && c.State().Dragging {
		return c.Apply(viewer.Action{Kind: "drag", X: x, Y: y})
	}
	return false, nil
}
