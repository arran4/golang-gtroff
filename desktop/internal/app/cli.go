package app

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/renderimage"
	"github.com/arran4/golang-gtroff/viewer"
)

type Environment struct {
	In                    io.Reader
	Out, Err              io.Writer
	Open                  func(string) (io.ReadCloser, error)
	Show                  func(*viewer.Session) error
	BackendError          error
	Version, Commit, Date string
}

const Help = `Usage: gtroff-view [OPTIONS] FILE

Open a man source file in the optional Shiny viewer. FILE '-' reads stdin.
No includes, network resources or system man command are loaded.
  --title TEXT            Optional document/window title override
  --section HEADING       Display matching semantic sections
  --width PIXELS          Render page width, default 1000
  --page-height PIXELS    Render page height, default 1400
  --margin PIXELS         Page margin, default 48 (0 supported)
  --font-size POINTS      Default 15 at --dpi 96
  --dpi NUMBER            Document pixels/inch, default 96
  --theme light|dark      Default light
  --font FILE             Explicit regular TTF/OTF; other faces stay bundled
  --font-bold FILE        Explicit bold TTF/OTF
  --font-italic FILE      Explicit italic TTF/OTF
  --font-mono FILE        Explicit mono TTF/OTF
  --window-width PIXELS   Initial viewport, default 1000
  --window-height PIXELS  Initial viewport, default 800
  --strict               Reject conversion diagnostics before opening a window
  --version              Show build information without opening a window

Wheel/arrows scroll; left drag pans. +/- or Ctrl-wheel zoom. F fits width;
0 resets. N/P or brackets switch rendered pages. PageUp/PageDown/Space scroll
a screen. Home/End jump within a page. D shows notes, ?/F1 help, Q/Escape closes.
Viewport allocation is bounded to 4096x4096 / 16 Mi pixels; zoom is 25%..400%.
Exit codes: 0 normal close/help, 1 strict conversion, 2 usage, 3 IO/backend/resource.
`

func Run(args []string, e Environment) int {
	if e.In == nil {
		e.In = strings.NewReader("")
	}
	if e.Out == nil {
		e.Out = io.Discard
	}
	if e.Err == nil {
		e.Err = io.Discard
	}
	fail := func(code int, err error) int { fmt.Fprintln(e.Err, "gtroff-view:", err); return code }
	var o renderimage.Options
	var paths [4]string
	var width, height int
	var version bool
	fs := flag.NewFlagSet("gtroff-view", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.Title, "title", "", "")
	fs.StringVar(&o.Section, "section", "", "")
	fs.IntVar(&o.Width, "width", 1000, "")
	fs.IntVar(&o.PageHeight, "page-height", 1400, "")
	fs.IntVar(&o.Margin, "margin", 48, "")
	fs.Float64Var(&o.FontSize, "font-size", 15, "")
	fs.Float64Var(&o.DPI, "dpi", 96, "")
	fs.StringVar(&o.Theme, "theme", "light", "")
	fs.BoolVar(&o.Strict, "strict", false, "")
	fs.StringVar(&paths[0], "font", "", "")
	fs.StringVar(&paths[1], "font-bold", "", "")
	fs.StringVar(&paths[2], "font-italic", "", "")
	fs.StringVar(&paths[3], "font-mono", "", "")
	fs.IntVar(&width, "window-width", 1000, "")
	fs.IntVar(&height, "window-height", 800, "")
	fs.BoolVar(&version, "version", false, "")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if _, err := io.WriteString(e.Out, Help); err != nil {
				return fail(3, err)
			}
			return 0
		}
		return fail(2, err)
	}
	if version {
		if fs.NArg() != 0 {
			return fail(2, fmt.Errorf("version takes no input"))
		}
		if _, err := fmt.Fprintf(e.Out, "gtroff-view %s (commit %s, built %s)\n", e.Version, e.Commit, e.Date); err != nil {
			return fail(3, err)
		}
		return 0
	}
	if fs.NArg() != 1 {
		return fail(2, fmt.Errorf("supply one man file or '-' for stdin; use --help for controls"))
	}
	if width < 1 || height < 1 || width > 4096 || height > 4096 || int64(width)*int64(height) > viewer.MaxViewportPixels {
		return fail(2, viewer.ErrViewport)
	}
	if o.Width < 128 || o.PageHeight < 128 || o.Margin < 0 || o.FontSize < 6 || o.DPI < 36 {
		return fail(2, fmt.Errorf("render geometry outside supported range"))
	}
	o.NoMargin = o.Margin == 0
	if err := renderimage.ValidateOptions(o); err != nil {
		return fail(2, err)
	}
	if e.BackendError != nil {
		return fail(3, e.BackendError)
	}
	if e.Show == nil {
		return fail(3, fmt.Errorf("desktop capability unavailable"))
	}
	faces := []*[]byte{&o.Fonts.Regular, &o.Fonts.Bold, &o.Fonts.Italic, &o.Fonts.Mono}
	for i, path := range paths {
		if path == "" {
			continue
		}
		if e.Open == nil {
			return fail(3, fmt.Errorf("font input unavailable"))
		}
		f, err := e.Open(path)
		if err != nil {
			return fail(3, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, fontset.MaxFaceBytes+1))
		closeErr := f.Close()
		if readErr != nil {
			return fail(3, readErr)
		}
		if closeErr != nil {
			return fail(3, closeErr)
		}
		if len(data) == 0 || len(data) > fontset.MaxFaceBytes {
			return fail(3, fmt.Errorf("font exceeds face budget or is empty"))
		}
		*faces[i] = data
	}
	reader := e.In
	var owned io.ReadCloser
	name := fs.Arg(0)
	if name != "-" {
		if e.Open == nil {
			return fail(3, fmt.Errorf("file input unavailable"))
		}
		f, err := e.Open(name)
		if err != nil {
			return fail(3, err)
		}
		owned = f
		reader = f
	} else {
		name = "<stdin>"
	}
	session, err := viewer.Load(reader, name, o, image.Pt(width, height))
	if owned != nil {
		if closeErr := owned.Close(); closeErr != nil {
			if session != nil {
				_ = session.Close()
			}
			return fail(3, closeErr)
		}
	}
	if err != nil {
		if errors.Is(err, viewer.ErrSelection) {
			return fail(2, err)
		}
		if errors.Is(err, renderimage.ErrStrict) {
			return fail(1, err)
		}
		return fail(3, err)
	}
	defer session.Close()
	for _, d := range session.Diagnostics() {
		if _, err := fmt.Fprintf(e.Err, "%s:%d: %s: %s\n", d.Position.Filename, d.Position.Line, d.Severity, d.Message); err != nil {
			return 3
		}
	}
	if err := e.Show(session); err != nil {
		return fail(3, err)
	}
	return 0
}
