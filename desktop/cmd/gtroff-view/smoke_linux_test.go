//go:build linux && !android

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/renderimage"
	"github.com/arran4/golang-gtroff/viewer"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

var guiSmoke = flag.Bool("gui-smoke", false, "opt in to real X11 executable/input/resize smoke; requires DISPLAY and xdotool")
var screenshot = flag.String("screenshot", "", "optional explicit filename for a genuine X11 screenshot")

func TestX11Executable(t *testing.T) {
	if !*guiSmoke {
		t.Skip("real X11 smoke is explicit; use -gui-smoke under xvfb-run")
	}
	if os.Getenv("DISPLAY") == "" {
		t.Fatal("DISPLAY is required")
	}
	if _, e := exec.LookPath("xdotool"); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	work := t.TempDir()
	binaryPath := filepath.Join(work, "gtroff-view")
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build %v %s", e, out)
	}
	sourcePath, e := filepath.Abs("../../../examples/man-tool.1")
	if e != nil {
		t.Fatal(e)
	}
	source, e := os.ReadFile(sourcePath)
	if e != nil {
		t.Fatal(e)
	}
	conn, e := xgb.NewConn()
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	for _, selected := range []bool{false, true} {
		title := "MAN-TOOL(1)"
		if selected {
			title = "MAN-TOOL(1) OPTIONS"
		}
		tree, e := xproto.QueryTree(conn, xproto.Setup(conn).DefaultScreen(conn).Root).Reply()
		if e != nil {
			t.Fatal(e)
		}
		existing := map[xproto.Window]bool{}
		for _, w := range tree.Children {
			existing[w] = true
		}
		args := []string{"--title", title, "--width", "800", "--page-height", "700", "--window-width", "900", "--window-height", "600"}
		options := renderimage.Options{Width: 800, PageHeight: 700, Title: title}
		if selected {
			args = append(args, "--section", "OPTIONS")
			options.Section = "OPTIONS"
		}
		args = append(args, sourcePath)
		expected, e := viewer.Load(bytes.NewReader(source), sourcePath, options, image.Pt(900, 600))
		if e != nil {
			t.Fatal(e)
		}
		command := exec.CommandContext(ctx, binaryPath, args...)
		var log bytes.Buffer
		command.Stdout = &log
		command.Stderr = &log
		if e := command.Start(); e != nil {
			t.Fatal(e)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		stopped := false
		defer func() {
			if !stopped {
				_ = command.Process.Kill()
			}
		}()
		window, e := findWindow(ctx, conn, "gtroff-view - "+title, existing)
		if e != nil {
			_ = command.Process.Kill()
			<-done
			t.Fatalf("window unavailable: %v %s", e, log.String())
		}
		id := uint64(window)

		waitFrame(t, conn, window, expected)
		input := func(args ...string) {
			cmd := exec.CommandContext(ctx, "xdotool", args...)
			if out, e := cmd.CombinedOutput(); e != nil {
				t.Fatalf("xdotool %v: %v %s", args, e, out)
			}
		}
		keyInput := func(name string, a viewer.Action) {
			input("key", "--window", strconv.FormatUint(id, 10), name)
			if _, e := expected.Controller.Apply(a); e != nil {
				t.Fatal(e)
			}
			waitFrame(t, conn, window, expected)
		}
		if !selected {
			input("mousemove", "--window", strconv.FormatUint(id, 10), "400", "400", "click", "5")
			_, _ = expected.Controller.Apply(viewer.Action{Kind: "scroll", DY: 48})
			waitFrame(t, conn, window, expected)
			keyInput("plus", viewer.Action{Kind: "zoom", Factor: 1.1, X: 450, Y: 336})
			input("mousemove", "--window", strconv.FormatUint(id, 10), "400", "300", "mousedown", "1", "mousemove", "--window", strconv.FormatUint(id, 10), "300", "200", "mouseup", "1")
			_, _ = expected.Controller.Apply(viewer.Action{Kind: "drag-start", X: 400, Y: 300})
			_, _ = expected.Controller.Apply(viewer.Action{Kind: "drag", X: 300, Y: 200})
			_, _ = expected.Controller.Apply(viewer.Action{Kind: "drag-end"})
			waitFrame(t, conn, window, expected)
			input("windowsize", strconv.FormatUint(id, 10), "1100", "720")
			_, _ = expected.Controller.Apply(viewer.Action{Kind: "resize", Width: 1100, Height: 720})
			waitFrame(t, conn, window, expected)
			keyInput("f", viewer.Action{Kind: "fit"})
			keyInput("n", viewer.Action{Kind: "next"})
			keyInput("d", viewer.Action{Kind: "notes"})
			if *screenshot != "" {
				img, e := capture(conn, window)
				if e != nil {
					t.Fatal(e)
				}
				f, e := os.Create(*screenshot)
				if e != nil {
					t.Fatal(e)
				}
				encodeErr := png.Encode(f, img)
				closeErr := f.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			keyInput("p", viewer.Action{Kind: "previous"})
			keyInput("F1", viewer.Action{Kind: "help"})
		}
		// Closing on key press may destroy the window before xdotool sends its
		// release; process exit, rather than that synthetic release, is the contract.
		closeOutput, closeErr := exec.CommandContext(ctx, "xdotool", "key", "--window", strconv.FormatUint(id, 10), "Escape").CombinedOutput()
		select {
		case e := <-done:
			stopped = true
			if e != nil {
				t.Fatal(e, log.String())
			}
		case <-time.After(5 * time.Second):
			_ = command.Process.Kill()
			t.Fatalf("viewer failed to close: %v %s", closeErr, closeOutput)
		}
		_ = expected.Close()
	}
}
func waitFrame(t *testing.T, conn *xgb.Conn, window xproto.Window, expected *viewer.Session) {
	t.Helper()
	want, e := expected.Frame()
	if e != nil {
		t.Fatal(e)
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	var last error
	for {
		img, e := capture(conn, window)
		last = e
		if e == nil && img.Bounds() == want.Bounds() && bytes.Equal(img.Pix, want.Pix) {
			return
		}
		select {
		case <-timer.C:
			if last != nil {
				t.Fatal(last)
			}
			if *screenshot != "" {
				if actual, e := capture(conn, window); e == nil {
					saveFailure(t, *screenshot+".actual.png", actual)
				}
				saveFailure(t, *screenshot+".expected.png", want)
			}
			t.Fatalf("native frame did not match expected controller state %+v", expected.Controller.State())
		case <-tick.C:
		}
	}
}
func capture(conn *xgb.Conn, window xproto.Window) (*image.RGBA, error) {
	g, e := xproto.GetGeometry(conn, xproto.Drawable(window)).Reply()
	if e != nil {
		return nil, e
	}
	r, e := xproto.GetImage(conn, xproto.ImageFormatZPixmap, xproto.Drawable(window), 0, 0, g.Width, g.Height, ^uint32(0)).Reply()
	if e != nil {
		return nil, e
	}
	setup := xproto.Setup(conn)
	bpp, pad := 0, 0
	for _, f := range setup.PixmapFormats {
		if f.Depth == r.Depth {
			bpp = int(f.BitsPerPixel)
			pad = int(f.ScanlinePad)
		}
	}
	if bpp != 32 || pad == 0 {
		return nil, fmt.Errorf("unsupported smoke capture format %d", bpp)
	}
	var visual xproto.VisualInfo
	for _, s := range setup.Roots {
		for _, d := range s.AllowedDepths {
			for _, v := range d.Visuals {
				if v.VisualId == r.Visual {
					visual = v
				}
			}
		}
	}
	if visual.RedMask == 0 || visual.GreenMask == 0 || visual.BlueMask == 0 {
		return nil, fmt.Errorf("missing visual channel masks")
	}
	stride := ((int(g.Width)*bpp + pad - 1) / pad) * pad / 8
	out := image.NewRGBA(image.Rect(0, 0, int(g.Width), int(g.Height)))
	channel := func(pixel, mask uint32) uint8 {
		shift := bits.TrailingZeros32(mask)
		return uint8(((pixel & mask) >> shift) * 255 / (mask >> shift))
	}
	for y := 0; y < int(g.Height); y++ {
		for x := 0; x < int(g.Width); x++ {
			offset := y*stride + x*4
			if offset+4 > len(r.Data) {
				return nil, fmt.Errorf("short screenshot data")
			}
			p := binary.LittleEndian.Uint32(r.Data[offset:])
			if setup.ImageByteOrder == xproto.ImageOrderMSBFirst {
				p = binary.BigEndian.Uint32(r.Data[offset:])
			}
			dst := y*out.Stride + x*4
			out.Pix[dst] = channel(p, visual.RedMask)
			out.Pix[dst+1] = channel(p, visual.GreenMask)
			out.Pix[dst+2] = channel(p, visual.BlueMask)
			out.Pix[dst+3] = 255
		}
	}
	return out, nil
}

func findWindow(ctx context.Context, conn *xgb.Conn, title string, existing map[xproto.Window]bool) (xproto.Window, error) {
	atom, e := xproto.InternAtom(conn, false, uint16(len("_NET_WM_NAME")), "_NET_WM_NAME").Reply()
	if e != nil {
		return 0, e
	}
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		tree, e := xproto.QueryTree(conn, xproto.Setup(conn).DefaultScreen(conn).Root).Reply()
		if e != nil {
			return 0, e
		}
		for _, child := range tree.Children {
			if existing[child] {
				continue
			}
			p, e := xproto.GetProperty(conn, false, child, atom.Atom, xproto.GetPropertyTypeAny, 0, 1024).Reply()
			if e == nil && string(p.Value) == title {
				return child, nil
			}
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
			return 0, fmt.Errorf("native window title not found")
		case <-tick.C:
		}
	}
}

func saveFailure(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, e := os.Create(path)
	if e != nil {
		t.Log(e)
		return
	}
	defer f.Close()
	if e := png.Encode(f, img); e != nil {
		t.Log(e)
	}
}
