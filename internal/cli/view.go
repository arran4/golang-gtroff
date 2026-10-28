package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/arran4/golang-gtroff/cmd"
)

const viewHelp = `Usage: gtroff view [OPTIONS] FILE

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

Exit codes: 0 normal close/help, 1 strict conversion, 2 usage, 3 IO/backend/resource.
`

// View is a subcommand `gtroff view` Open a man source file in the optional Shiny viewer
//
// Usage: gtroff view [OPTIONS] FILE
//
// Open a man source file in the optional Shiny viewer. FILE '-' reads stdin.
// No includes, network resources or system man command are loaded.
//
//   --title TEXT            Optional document/window title override
//   --section HEADING       Display matching semantic sections
//   --width PIXELS          Render page width, default 1000
//   --page-height PIXELS    Render page height, default 1400
//   --margin PIXELS         Page margin, default 48 (0 supported)
//   --font-size POINTS      Default 15 at --dpi 96
//   --dpi NUMBER            Document pixels/inch, default 96
//   --theme light|dark      Default light
//   --font FILE             Explicit regular TTF/OTF; other faces stay bundled
//   --font-bold FILE        Explicit bold TTF/OTF
//   --font-italic FILE      Explicit italic TTF/OTF
//   --font-mono FILE        Explicit mono TTF/OTF
//   --window-width PIXELS   Initial viewport, default 1000
//   --window-height PIXELS  Initial viewport, default 800
//   --strict               Reject conversion diagnostics before opening a window
//   --version              Show build information without opening a window
//
// Exit codes: 0 normal close/help, 1 strict conversion, 2 usage, 3 IO/backend/resource.
//
// Flags:
//
//	title: --title (default: "") Optional document/window title override
//	section: --section (default: "") Display matching semantic sections
//	width: --width (default: 1000) Render page width, default 1000
//	pageHeight: --page-height (default: 1400) Render page height, default 1400
//	margin: --margin (default: 48) Page margin, default 48
//	fontSize: --font-size (default: 15) Default 15 at dpi 96
//	dpi: --dpi (default: 96) Document pixels/inch, default 96
//	theme: --theme (default: "light") Default light
//	strict: --strict (default: false) Reject conversion diagnostics before opening a window
//	fontRegular: --font (default: "") Explicit regular TTF/OTF
//	fontBold: --font-bold (default: "") Explicit bold TTF/OTF
//	fontItalic: --font-italic (default: "") Explicit italic TTF/OTF
//	fontMono: --font-mono (default: "") Explicit mono TTF/OTF
//	windowWidth: --window-width (default: 1000) Initial viewport, default 1000
//	windowHeight: --window-height (default: 800) Initial viewport, default 800
//	version: --version (default: false) Show build information without opening a window
//	files: ... Input man source file
func View(title, section string, width, pageHeight, margin int, fontSize, dpi float64, theme string, strict bool, fontRegular, fontBold, fontItalic, fontMono string, windowWidth, windowHeight int, version bool, files ...string) error {
	code := runView(title, section, width, pageHeight, margin, fontSize, dpi, theme, strict, fontRegular, fontBold, fontItalic, fontMono, windowWidth, windowHeight, version, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func runView(title, section string, width, pageHeight, margin int, fontSize, dpi float64, theme string, strict bool, fontRegular, fontBold, fontItalic, fontMono string, windowWidth, windowHeight int, version bool, files []string, e Environment) int {
	if version {
		return writeString(e, fmt.Sprintf("gtroff view (backend: gtroff-view %s, commit %s, built %s)\n", e.Build.Version, e.Build.Commit, e.Build.Date))
	}
	if len(files) == 0 {
		return usage(e, "view requires a file argument")
	}
	if len(files) > 1 {
		return usage(e, "view expects exactly one input file")
	}
	bin, err := findViewerBinary()
	if err != nil {
		return failure(e, err.Error())
	}

	var args []string
	if title != "" {
		args = append(args, "--title", title)
	}
	if section != "" {
		args = append(args, "--section", section)
	}
	if width != 1000 && width != 0 {
		args = append(args, "--width", strconv.Itoa(width))
	}
	if pageHeight != 1400 && pageHeight != 0 {
		args = append(args, "--page-height", strconv.Itoa(pageHeight))
	}
	if margin != 48 && margin != 0 {
		args = append(args, "--margin", strconv.Itoa(margin))
	}
	if fontSize != 15 && fontSize != 0 {
		args = append(args, "--font-size", strconv.FormatFloat(fontSize, 'f', -1, 64))
	}
	if dpi != 96 && dpi != 0 {
		args = append(args, "--dpi", strconv.FormatFloat(dpi, 'f', -1, 64))
	}
	if theme != "" && theme != "light" {
		args = append(args, "--theme", theme)
	}
	if strict {
		args = append(args, "--strict")
	}
	if fontRegular != "" {
		args = append(args, "--font", fontRegular)
	}
	if fontBold != "" {
		args = append(args, "--font-bold", fontBold)
	}
	if fontItalic != "" {
		args = append(args, "--font-italic", fontItalic)
	}
	if fontMono != "" {
		args = append(args, "--font-mono", fontMono)
	}
	if windowWidth != 1000 && windowWidth != 0 {
		args = append(args, "--window-width", strconv.Itoa(windowWidth))
	}
	if windowHeight != 800 && windowHeight != 0 {
		args = append(args, "--window-height", strconv.Itoa(windowHeight))
	}
	args = append(args, files[0])

	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	command := exec.CommandContext(ctx, bin, args...)
	command.Stdin = e.In
	command.Stdout = e.Out
	command.Stderr = e.Err
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		return failure(e, "launch viewer: "+err.Error())
	}
	return ExitOK
}

func findViewerBinary() (string, error) {
	if custom := os.Getenv("GTROFF_VIEW_BIN"); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "gtroff-view")
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("gtroff-view"); err == nil {
		return path, nil
	}
	return "", errors.New("gtroff-view is unavailable; install or build the optional desktop viewer (cd desktop && go build ./cmd/gtroff-view)")
}

func viewCommand(args []string, e Environment) int {
	if len(args) == 0 {
		return usage(e, "view requires a file argument")
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return writeString(e, viewHelp)
	}
	if len(args) == 1 && (args[0] == "--version") {
		return writeString(e, fmt.Sprintf("gtroff view (backend: gtroff-view %s, commit %s, built %s)\n", e.Build.Version, e.Build.Commit, e.Build.Date))
	}
	var title, section string
	var width, pageHeight, margin int
	var fontSize, dpi float64
	var theme string
	var strict, version bool
	var fontRegular, fontBold, fontItalic, fontMono string
	var windowWidth, windowHeight int

	flags := flag.NewFlagSet("view", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&title, "title", "", "")
	flags.StringVar(&section, "section", "", "")
	flags.IntVar(&width, "width", 1000, "")
	flags.IntVar(&pageHeight, "page-height", 1400, "")
	flags.IntVar(&margin, "margin", 48, "")
	flags.Float64Var(&fontSize, "font-size", 15, "")
	flags.Float64Var(&dpi, "dpi", 96, "")
	flags.StringVar(&theme, "theme", "light", "")
	flags.BoolVar(&strict, "strict", false, "")
	flags.StringVar(&fontRegular, "font", "", "")
	flags.StringVar(&fontRegular, "font-regular", "", "")
	flags.StringVar(&fontBold, "font-bold", "", "")
	flags.StringVar(&fontItalic, "font-italic", "", "")
	flags.StringVar(&fontMono, "font-mono", "", "")
	flags.IntVar(&windowWidth, "window-width", 1000, "")
	flags.IntVar(&windowHeight, "window-height", 800, "")
	flags.BoolVar(&version, "version", false, "")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, viewHelp)
		}
		return usage(e, err.Error())
	}
	if version {
		return writeString(e, fmt.Sprintf("gtroff view (backend: gtroff-view %s, commit %s, built %s)\n", e.Build.Version, e.Build.Commit, e.Build.Date))
	}
	if flags.NArg() == 0 {
		return usage(e, "view requires a file argument")
	}
	if flags.NArg() > 1 {
		return usage(e, "view expects exactly one input file")
	}
	return runView(title, section, width, pageHeight, margin, fontSize, dpi, theme, strict, fontRegular, fontBold, fontItalic, fontMono, windowWidth, windowHeight, version, flags.Args(), e)
}
