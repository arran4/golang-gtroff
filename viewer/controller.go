// Package viewer provides bounded desktop-independent page navigation and painting.
// It reuses renderimage plans and has no Shiny, window, filesystem or network dependency.
package viewer

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/arran4/golang-gtroff/renderimage"
)

var ErrViewport = errors.New("viewer viewport exceeds resource limits")

const MaxViewportPixels int64 = 16 << 20
const ToolbarHeight = 72
const MinZoom = .25
const MaxZoom = 4.0

type Action struct {
	Kind                 string
	X, Y, DX, DY, Factor float64
	Width, Height        int
}
type State struct {
	Page, Pages                                   int
	Zoom                                          float64
	PanX, PanY                                    float64
	Viewport                                      image.Point
	FitWidth, Help, Diagnostics, Dragging, Closed bool
}
type Controller struct {
	state        State
	pages        []image.Point
	lastX, lastY float64
}

func NewController(pages []image.Point, viewport image.Point) (*Controller, error) {
	if len(pages) == 0 || len(pages) > renderimage.HardMaxPages {
		return nil, fmt.Errorf("invalid page count")
	}
	var pixels int64
	for _, p := range pages {
		if p.X < 1 || p.Y < 1 || p.X > 32768 || p.Y > 32768 {
			return nil, fmt.Errorf("invalid page bounds")
		}
		pixels += int64(p.X) * int64(p.Y)
		if pixels > renderimage.HardMaxPixels {
			return nil, renderimage.ErrLimit
		}
	}
	c := &Controller{state: State{Pages: len(pages), Zoom: 1}, pages: append([]image.Point(nil), pages...)}
	if _, err := c.Apply(Action{Kind: "resize", Width: viewport.X, Height: viewport.Y}); err != nil {
		return nil, err
	}
	return c, nil
}
func (c *Controller) State() State { return c.state }
func viewportOK(width, height int) bool {
	return width >= 0 && height >= 0 && width <= 4096 && height <= 4096 && int64(width)*int64(height) <= MaxViewportPixels
}
func (c *Controller) Content() image.Rectangle {
	s := c.state
	bottom := s.Viewport.Y
	if s.Diagnostics {
		bottom -= min(180, max(0, (bottom-ToolbarHeight)/2))
	}
	if bottom < ToolbarHeight {
		return image.Rectangle{}
	}
	return image.Rect(0, ToolbarHeight, s.Viewport.X, bottom)
}
func (c *Controller) Destination() image.Rectangle {
	content := c.Content()
	p := c.pages[c.state.Page]
	width := int(math.Round(float64(p.X) * c.state.Zoom))
	height := int(math.Round(float64(p.Y) * c.state.Zoom))
	x := content.Min.X + max(0, (content.Dx()-width)/2) - int(math.Round(c.state.PanX))
	y := content.Min.Y + max(0, (content.Dy()-height)/2) - int(math.Round(c.state.PanY))
	return image.Rect(x, y, x+width, y+height)
}
func (c *Controller) clamp() {
	s := &c.state
	content := c.Content()
	p := c.pages[s.Page]
	// Explicit rounding prevents architecture-dependent multiply/add fusion in
	// the public state snapshot (for example 592 vs 592.0000000000001 on arm64).
	s.PanX = max(0, min(s.PanX, float64(float64(p.X)*s.Zoom)-float64(content.Dx())))
	s.PanY = max(0, min(s.PanY, float64(float64(p.Y)*s.Zoom)-float64(content.Dy())))
}
func (c *Controller) fit() {
	if !c.Content().Empty() {
		c.state.Zoom = max(MinZoom, min(MaxZoom, float64(c.Content().Dx())/float64(c.pages[c.state.Page].X)))
	}
	c.state.PanX = 0
	c.clamp()
}

// Apply uses window pixel coordinates. Invalid input leaves state unchanged.
// Zoom preserves the source point under its anchor where clamping permits.
func (c *Controller) Apply(a Action) (bool, error) {
	if c.state.Closed {
		return false, nil
	}
	for _, v := range []float64{a.X, a.Y, a.DX, a.DY, a.Factor} {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e9 {
			return false, fmt.Errorf("invalid viewer gesture")
		}
	}
	if a.Kind == "resize" && !viewportOK(a.Width, a.Height) {
		return false, ErrViewport
	}
	if a.Kind == "zoom" && a.Factor <= 0 {
		return false, fmt.Errorf("zoom factor must be positive")
	}
	before := c.state
	s := &c.state
	switch a.Kind {
	case "resize":
		s.Viewport = image.Pt(a.Width, a.Height)
		s.Dragging = false
		if s.FitWidth {
			c.fit()
		}
	case "scroll":
		s.PanX += a.DX
		s.PanY += a.DY
	case "zoom":
		dest := c.Destination()
		sourceX := (a.X - float64(dest.Min.X)) / s.Zoom
		sourceY := (a.Y - float64(dest.Min.Y)) / s.Zoom
		s.Zoom = max(MinZoom, min(MaxZoom, s.Zoom*a.Factor))
		s.FitWidth = false
		content := c.Content()
		p := c.pages[s.Page]
		centerX := max(0, (float64(content.Dx())-float64(float64(p.X)*s.Zoom))/2)
		centerY := max(0, (float64(content.Dy())-float64(float64(p.Y)*s.Zoom))/2)
		s.PanX = float64(content.Min.X) + centerX + float64(sourceX*s.Zoom) - a.X
		s.PanY = float64(content.Min.Y) + centerY + float64(sourceY*s.Zoom) - a.Y
	case "fit":
		s.FitWidth = true
		c.fit()
	case "reset":
		s.Zoom = 1
		s.PanX = 0
		s.PanY = 0
		s.FitWidth = false
	case "next", "previous", "first", "last":
		target := s.Page
		switch a.Kind {
		case "next":
			target++
		case "previous":
			target--
		case "first":
			target = 0
		case "last":
			target = len(c.pages) - 1
		}
		target = max(0, min(len(c.pages)-1, target))
		if target != s.Page {
			s.Page = target
			s.PanX = 0
			s.PanY = 0
			s.Dragging = false
			if s.FitWidth {
				c.fit()
			}
		}
	case "home":
		s.PanX = 0
		s.PanY = 0
	case "end":
		s.PanY = float64(c.pages[s.Page].Y) * s.Zoom
	case "drag-start":
		if image.Pt(int(a.X), int(a.Y)).In(c.Content()) {
			s.Dragging = true
			c.lastX = a.X
			c.lastY = a.Y
		}
	case "drag":
		if s.Dragging {
			s.PanX += c.lastX - a.X
			s.PanY += c.lastY - a.Y
			c.lastX = a.X
			c.lastY = a.Y
		}
	case "drag-end":
		s.Dragging = false
	case "notes":
		s.Diagnostics = !s.Diagnostics
	case "help":
		s.Help = !s.Help
	case "close":
		s.Closed = true
		s.Dragging = false
	default:
		return false, fmt.Errorf("unknown viewer action %q", a.Kind)
	}
	c.clamp()
	return before != c.state, nil
}

type Button struct {
	Label, Action string
	Bounds        image.Rectangle
}

func (c *Controller) Buttons() []Button {
	var out []Button
	x := 8
	for _, b := range []struct {
		label, action string
		width         int
	}{{"< Prev", "previous", 68}, {"Next >", "next", 68}, {"-", "zoom-out", 28}, {"+", "zoom-in", 28}, {"Fit", "fit", 40}, {"Notes", "notes", 56}, {"Help", "help", 52}} {
		r := image.Rect(x, 32, x+b.width, 62)
		if r.Max.X <= c.state.Viewport.X {
			out = append(out, Button{b.label, b.action, r})
		}
		x += b.width + 6
	}
	return out
}
func (c *Controller) Click(x, y float64) (bool, error) {
	for _, b := range c.Buttons() {
		if image.Pt(int(x), int(y)).In(b.Bounds) {
			a := Action{Kind: b.Action}
			if a.Kind == "zoom-in" || a.Kind == "zoom-out" {
				a.Kind = "zoom"
				a.Factor = 1.1
				if b.Action == "zoom-out" {
					a.Factor = 1 / 1.1
				}
				content := c.Content()
				a.X = float64(content.Min.X + content.Dx()/2)
				a.Y = float64(content.Min.Y + content.Dy()/2)
			}
			return c.Apply(a)
		}
	}
	return c.Apply(Action{Kind: "drag-start", X: x, Y: y})
}
