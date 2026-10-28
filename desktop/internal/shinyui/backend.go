//go:build (linux && !android) || windows || (darwin && cgo)

package shinyui

import (
	"fmt"
	"image"
	"unicode/utf8"

	"github.com/arran4/golang-gtroff/viewer"
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

func BackendError() error { return nil }
func Show(session *viewer.Session) error {
	var failure error
	driver.Main(func(s screen.Screen) { failure = run(s, session) })
	return failure
}
func run(s screen.Screen, session *viewer.Session) error {
	state := session.Controller.State()
	title := "gtroff-view - " + session.Title()
	if utf8.RuneCountInString(title) > 200 {
		title = string([]rune(title)[:200])
	}
	w, err := s.NewWindow(&screen.NewWindowOptions{Title: title, Width: state.Viewport.X, Height: state.Viewport.Y})
	if err != nil {
		return fmt.Errorf("open Shiny window: %w", err)
	}
	defer w.Release()
	b, err := s.NewBuffer(state.Viewport)
	if err != nil {
		return fmt.Errorf("allocate initial Shiny viewport: %w", err)
	}
	defer func() {
		if b != nil {
			b.Release()
		}
	}()
	pending := false
	redraw := func() {
		if !pending {
			pending = true
			w.Send(paint.Event{})
		}
	}
	redraw()
	for {
		changed := false
		switch e := w.NextEvent().(type) {
		case lifecycle.Event:
			if e.To == lifecycle.StageDead {
				_, _ = session.Controller.Apply(viewer.Action{Kind: "close"})
				return nil
			}
			if e.To < lifecycle.StageFocused {
				changed, _ = session.Controller.Apply(viewer.Action{Kind: "drag-end"})
			}
		case size.Event:
			_, err := session.Controller.Apply(viewer.Action{Kind: "resize", Width: e.WidthPx, Height: e.HeightPx})
			if err != nil {
				return err
			}
			if b != nil {
				b.Release()
				b = nil
			}
			if e.WidthPx > 0 && e.HeightPx > 0 {
				b, err = s.NewBuffer(image.Pt(e.WidthPx, e.HeightPx))
				if err != nil {
					return fmt.Errorf("allocate Shiny viewport: %w", err)
				}
			}
			changed = true
		case paint.Event:
			pending = false
			if b == nil {
				continue
			}
			if err := session.Draw(b.RGBA()); err != nil {
				return err
			}
			w.Upload(image.Point{}, b, b.Bounds())
			w.Publish()
		case key.Event:
			changed, err = Key(session.Controller, e)
			if err != nil {
				session.SetError(err)
				changed = true
			}
		case mouse.Event:
			changed, err = Mouse(session.Controller, e)
			if err != nil {
				session.SetError(err)
				changed = true
			}
		case error:
			session.SetError(e)
			changed = true
		}
		if session.Controller.State().Closed {
			return nil
		}
		if changed {
			redraw()
		}
	}
}
