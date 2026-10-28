package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/arran4/golang-gtroff/cmd"
	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
	"github.com/arran4/golang-gtroff/renderimage"
)

// Render is a subcommand `gtroff render` Export a semantic man page to HTML or paginated images
//
// Usage: gtroff render --format html|png|jpg [OPTIONS] [FILE]
//
// Export common Linux man syntax to HTML or paginated PNG/JPEG with bundled Go fonts.
// FILE defaults to '-' (stdin). No includes or external resources are loaded.
//
//   --section HEADING       Select all matching sections/subsections
//   --node ID               Select an AST source ID shown by gtroff ast
//   --theme light|dark      Default: light
//   --title TEXT            Override the document title
//   --fragment              Write a styled article without an HTML document shell
//   --no-anchors            Omit heading permalink anchors
//   --assets embed|omit-fonts  Default: embed (four bundled Go fonts)
//   --font-size PIXELS      8..72, default 18
//   --line-height RATIO     1..3, default 1.5
//   --max-width PIXELS      320..2400, default 960
//   --font FILE             Alias for --font-regular
//   --font-regular FILE     Explicit custom TTF/OTF; other faces keep bundled defaults
//   --font-bold FILE        Explicit custom bold font
//   --font-italic FILE      Explicit custom italic font
//   --font-mono FILE        Explicit custom mono font
//   --strict               Reject any semantic/render diagnostic before writing
//   -o FILE                Atomic regular-file replacement; '-' means stdout
//
// Image options (device pixels except --font-size in points and --dpi):
//   --width PIXELS          128..8192, default 1000
//   --margin PIXELS         Nonnegative, default 48 (0 supported)
//   --font-size POINTS      6..72, default 15 for images
//   --dpi NUMBER            36..300, default 96 pixels/inch
//   --line-height RATIO     1..3, default 1.35 for images
//   --page-height PIXELS    128..8192, default 1400
//   --single                Derive one image's height within guarded limits
//   --max-height PIXELS     128..32768, default 32768
//   --max-pixels COUNT      Total across all pages; default/hard ceiling 67108864
//   --quality NUMBER        JPEG 1..100, default 90
//
// Images on stdout require one page. Multi-page foo.png output uses foo-001.png,
// foo-002.png, etc. Files are all staged before per-file atomic replacements;
// a rename failure can leave earlier pages replaced. No collection-wide atomicity.
// Missing Unicode glyphs use visible code-point labels and diagnostics.
//
// Diagnostics go to stderr. Malformed input may produce fallback HTML with exit 1;
// strict, resource and input/output failures leave existing output files intact.
// Exit codes: 0 success/warnings, 1 syntax/strict failure, 2 usage/selection,
// 3 input/output/resource failure. No full groff typesetting promise is made.
//
// Flags:
//
//	format: --format Output format: html, png or jpg (default: "html")
//	section: --section Select all matching sections/subsections (default: "")
//	node: --node Select an AST source ID shown by gtroff ast (default: "")
//	output: -o (default: "-") Atomic regular-file replacement; '-' means stdout
//	theme: --theme Default: light (default: "light")
//	title: --title Override the document title (default: "")
//	fragment: --fragment Write a styled article without an HTML document shell (default: false)
//	noAnchors: --no-anchors Omit heading permalink anchors (default: false)
//	strict: --strict Reject any semantic/render diagnostic before writing (default: false)
//	assets: --assets Default: embed (four bundled Go fonts) (default: "embed")
//	fontSize: --font-size 8..72, default 18 (default: 18)
//	lineHeight: --line-height 1..3, default 1.5 (default: 1.5)
//	maxWidth: --max-width 320..2400, default 960 (default: 960)
//	fontRegular: --font-regular (alias: --font) Explicit custom TTF/OTF font
//	fontBold: --font-bold Explicit custom bold font
//	fontItalic: --font-italic Explicit custom italic font
//	fontMono: --font-mono Explicit custom mono font
//	width: --width 128..8192, default 1000 (default: 1000)
//	margin: --margin Nonnegative, default 48 (default: 48)
//	dpi: --dpi 36..300, default 96 pixels/inch (default: 96)
//	pageHeight: --page-height 128..8192, default 1400 (default: 1400)
//	single: --single Derive one image's height within guarded limits (default: false)
//	maxHeight: --max-height 128..32768, default 32768 (default: 32768)
//	maxPixels: --max-pixels Total across all pages; default/hard ceiling 67108864 (default: 67108864)
//	quality: --quality JPEG 1..100, default 90 (default: 90)
//	files: ... Input file, '-' or omitted for stdin
func Render(format, section, node, output, theme, title string, fragment, noAnchors, strict bool, assets string, fontSize, lineHeight float64, maxWidth int, fontRegular, fontBold, fontItalic, fontMono string, width, margin int, dpi float64, pageHeight int, single bool, maxHeight int, maxPixels int64, quality int, files ...string) error {
	var args []string
	if format != "" {
		args = append(args, "--format", format)
	}
	if section != "" {
		args = append(args, "--section", section)
	}
	if node != "" {
		args = append(args, "--node", node)
	}
	if output != "" {
		args = append(args, "-o", output)
	}
	if theme != "" {
		args = append(args, "--theme", theme)
	}
	if title != "" {
		args = append(args, "--title", title)
	}
	if fragment {
		args = append(args, "--fragment")
	}
	if noAnchors {
		args = append(args, "--no-anchors")
	}
	if strict {
		args = append(args, "--strict")
	}
	if assets != "" && assets != "embed" {
		args = append(args, "--assets", assets)
	}
	if fontSize != 0 && fontSize != 18 {
		args = append(args, "--font-size", strconv.FormatFloat(fontSize, 'f', -1, 64))
	}
	if lineHeight != 0 && lineHeight != 1.5 {
		args = append(args, "--line-height", strconv.FormatFloat(lineHeight, 'f', -1, 64))
	}
	if maxWidth != 0 && maxWidth != 960 {
		args = append(args, "--max-width", strconv.Itoa(maxWidth))
	}
	if fontRegular != "" {
		args = append(args, "--font-regular", fontRegular)
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
	if width != 0 && width != 1000 {
		args = append(args, "--width", strconv.Itoa(width))
	}
	if margin != 0 && margin != 48 {
		args = append(args, "--margin", strconv.Itoa(margin))
	}
	if dpi != 0 && dpi != 96 {
		args = append(args, "--dpi", strconv.FormatFloat(dpi, 'f', -1, 64))
	}
	if pageHeight != 0 && pageHeight != 1400 {
		args = append(args, "--page-height", strconv.Itoa(pageHeight))
	}
	if single {
		args = append(args, "--single")
	}
	if maxHeight != 0 && maxHeight != 32768 {
		args = append(args, "--max-height", strconv.Itoa(maxHeight))
	}
	if maxPixels != 0 && maxPixels != renderimage.HardMaxPixels {
		args = append(args, "--max-pixels", strconv.FormatInt(maxPixels, 10))
	}
	if quality != 0 && quality != 90 {
		args = append(args, "--quality", strconv.Itoa(quality))
	}
	args = append(args, files...)
	code := renderCommand(args, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

const renderHelp = `Usage: gtroff render --format html|png|jpg [OPTIONS] [FILE]

Export common Linux man syntax to HTML or paginated PNG/JPEG with bundled Go fonts.
FILE defaults to '-' (stdin). No includes or external resources are loaded.

  --section HEADING       Select all matching sections/subsections
  --node ID               Select an AST source ID shown by gtroff ast
  --theme light|dark      Default: light
  --title TEXT            Override the document title
  --fragment              Write a styled article without an HTML document shell
  --no-anchors            Omit heading permalink anchors
  --assets embed|omit-fonts  Default: embed (four bundled Go fonts)
  --font-size PIXELS      8..72, default 18
  --line-height RATIO     1..3, default 1.5
  --max-width PIXELS      320..2400, default 960
  --font FILE             Alias for --font-regular
  --font-regular FILE     Explicit custom TTF/OTF; other faces keep bundled defaults
  --font-bold FILE        Explicit custom bold font
  --font-italic FILE      Explicit custom italic font
  --font-mono FILE        Explicit custom mono font
  --strict               Reject any semantic/render diagnostic before writing
  -o FILE                Atomic regular-file replacement; '-' means stdout

Image options (device pixels except --font-size in points and --dpi):
  --width PIXELS          128..8192, default 1000
  --margin PIXELS         Nonnegative, default 48 (0 supported)
  --font-size POINTS      6..72, default 15 for images
  --dpi NUMBER            36..300, default 96 pixels/inch
  --line-height RATIO     1..3, default 1.35 for images
  --page-height PIXELS    128..8192, default 1400
  --single                Derive one image's height within guarded limits
  --max-height PIXELS     128..32768, default 32768
  --max-pixels COUNT      Total across all pages; default/hard ceiling 67108864
  --quality NUMBER        JPEG 1..100, default 90

Images on stdout require one page. Multi-page foo.png output uses foo-001.png,
foo-002.png, etc. Files are all staged before per-file atomic replacements;
a rename failure can leave earlier pages replaced. No collection-wide atomicity.
Missing Unicode glyphs use visible code-point labels and diagnostics.

Diagnostics go to stderr. Malformed input may produce fallback HTML with exit 1;
strict, resource and input/output failures leave existing output files intact.
Exit codes: 0 success/warnings, 1 syntax/strict failure, 2 usage/selection,
3 input/output/resource failure. No full groff typesetting promise is made.
`

func renderCommand(args []string, e Environment) int {
	var o renderhtml.Options
	var raster renderimage.Options
	var format, node, output string
	var sourceNode groff.NodeID
	var paths struct{ regular, bold, italic, mono string }
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&format, "format", "html", "")
	flags.StringVar(&o.Section, "section", "", "")
	flags.StringVar(&node, "node", "", "")
	flags.StringVar(&output, "o", "-", "")
	flags.StringVar(&o.Theme, "theme", "light", "")
	flags.StringVar(&o.Title, "title", "", "")
	flags.BoolVar(&o.Fragment, "fragment", false, "")
	flags.BoolVar(&o.NoAnchors, "no-anchors", false, "")
	flags.BoolVar(&o.Strict, "strict", false, "")
	assets := flags.String("assets", "embed", "")
	flags.Float64Var(&o.FontSize, "font-size", 18, "")
	flags.Float64Var(&o.LineHeight, "line-height", 1.5, "")
	flags.IntVar(&o.MaxWidth, "max-width", 960, "")
	flags.StringVar(&paths.regular, "font-regular", "", "")
	flags.StringVar(&paths.regular, "font", "", "")
	flags.IntVar(&raster.Width, "width", 1000, "")
	flags.IntVar(&raster.Margin, "margin", 48, "")
	flags.Float64Var(&raster.DPI, "dpi", 96, "")
	flags.IntVar(&raster.PageHeight, "page-height", 1400, "")
	flags.IntVar(&raster.MaxHeight, "max-height", 32768, "")
	flags.Int64Var(&raster.MaxPixels, "max-pixels", renderimage.HardMaxPixels, "")
	flags.IntVar(&raster.Quality, "quality", 90, "")
	flags.BoolVar(&raster.Single, "single", false, "")
	flags.StringVar(&paths.bold, "font-bold", "", "")
	flags.StringVar(&paths.italic, "font-italic", "", "")
	flags.StringVar(&paths.mono, "font-mono", "", "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, renderHelp)
		}
		return usage(e, err.Error())
	}
	if flags.NArg() > 1 {
		return usage(e, "expected at most one input file")
	}
	if format != "html" && format != "png" && format != "jpg" && format != "jpeg" {
		return usage(e, "render format must be html, png or jpg")
	}
	if node != "" {
		id, err := strconv.ParseUint(node, 10, 64)
		if err != nil || id == 0 {
			return usage(e, "node requires a positive AST ID")
		}
		sourceNode = groff.NodeID(id)
	}
	if sourceNode != 0 && o.Section != "" {
		return usage(e, "section and node selection are mutually exclusive")
	}
	visited := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if format != "html" {
		for _, name := range []string{"fragment", "no-anchors", "assets", "max-width"} {
			if visited[name] {
				return usage(e, name+" is an HTML-only option")
			}
		}
		raster.Format = format
		raster.Section = o.Section
		raster.Strict = o.Strict
		raster.Theme = o.Theme
		raster.Title = o.Title
		raster.FontSize = 15
		if visited["font-size"] {
			raster.FontSize = o.FontSize
		}
		raster.LineHeight = 1.35
		if visited["line-height"] {
			raster.LineHeight = o.LineHeight
		}
		raster.NoMargin = raster.Margin == 0
		name := "-"
		if flags.NArg() == 1 {
			name = flags.Arg(0)
		}
		return renderRaster(e, name, output, sourceNode, raster, paths)
	}
	for _, name := range []string{"width", "margin", "dpi", "page-height", "max-height", "max-pixels", "quality", "single"} {
		if visited[name] {
			return usage(e, name+" is an image-only option")
		}
	}
	o.Assets = renderhtml.AssetPolicy(*assets)
	if o.FontSize < 8 || o.FontSize > 72 {
		return usage(e, "font size must be 8..72 pixels")
	}
	if o.LineHeight < 1 || o.LineHeight > 3 {
		return usage(e, "line height must be 1..3")
	}
	if o.MaxWidth < 320 || o.MaxWidth > 2400 {
		return usage(e, "max width must be 320..2400 pixels")
	}
	// Validate ordinary options before opening any files.
	if err := renderhtml.ValidateOptions(o); err != nil {
		return usage(e, err.Error())
	}
	custom := paths.regular != "" || paths.bold != "" || paths.italic != "" || paths.mono != ""
	if custom && o.Assets != renderhtml.EmbedFonts {
		return usage(e, "custom fonts require embedded assets")
	}
	if output == "" {
		return usage(e, "output path cannot be empty")
	}
	if output != "-" && e.WriteOutput == nil {
		return failure(e, "output file writing is unavailable")
	}
	source := "<stdin>"
	sourcePath := ""
	reader := e.In
	var owned io.ReadCloser
	if flags.NArg() == 1 && flags.Arg(0) != "-" {
		source = flags.Arg(0)
		sourcePath = source
		if e.Open == nil {
			return failure(e, "file input is unavailable")
		}
		f, err := e.Open(source)
		if err != nil {
			return failure(e, "open "+strconv.Quote(source)+": "+err.Error())
		}
		reader = f
		owned = f
	}
	tree, parseErr := groff.ParseAST(reader, groff.WithPackage("man"), groff.WithSourceName(source))
	if owned != nil {
		if err := owned.Close(); err != nil {
			return failure(e, "close input: "+err.Error())
		}
	}
	if parseErr != nil {
		return failure(e, "read input: "+parseErr.Error())
	}
	doc, buildErr := man.Build(tree, man.Options{})
	if buildErr != nil {
		return failure(e, "semantic input: "+buildErr.Error())
	}
	// Resolve selection before loading optional user-requested fonts.
	var selection *man.Selection
	var selectionErr error
	switch {
	case o.Section != "":
		selection, selectionErr = doc.SelectSection(o.Section, false)
	case sourceNode != 0:
		selected := tree.Node(sourceNode)
		if selected == nil {
			selectionErr = fmt.Errorf("AST node %d not found", sourceNode)
		} else {
			if selected.Kind() == groff.ASTInline {
				selected = selected.Parent()
			}
			selection, selectionErr = doc.SelectAST(selected.ID())
		}
	default:
		selection, selectionErr = doc.SelectNode(1)
	}
	if selectionErr != nil {
		return usage(e, selectionErr.Error())
	}
	for _, face := range []struct {
		path string
		data *[]byte
	}{{paths.regular, &o.Fonts.Regular}, {paths.bold, &o.Fonts.Bold}, {paths.italic, &o.Fonts.Italic}, {paths.mono, &o.Fonts.Mono}} {
		if face.path == "" {
			continue
		}
		if e.Open == nil {
			return failure(e, "font file access is unavailable")
		}
		f, err := e.Open(face.path)
		if err != nil {
			return failure(e, "open font: "+err.Error())
		}
		data, readErr := io.ReadAll(io.LimitReader(f, fontset.MaxFaceBytes+1))
		closeErr := f.Close()
		if readErr != nil {
			return failure(e, "read font: "+readErr.Error())
		}
		if closeErr != nil {
			return failure(e, "close font: "+closeErr.Error())
		}
		if len(data) == 0 || len(data) > fontset.MaxFaceBytes {
			return failure(e, "font exceeds size limit or is empty")
		}
		*face.data = data
	}
	if custom {
		if err := o.Fonts.WithDefaults().Validate(); err != nil {
			return failure(e, "font: "+err.Error())
		}
	}
	var out bytes.Buffer
	result, err := renderhtml.RenderSelection(&out, doc, selection, o)
	if diagnosticErr := writeDiagnostics(e.Err, result.Diagnostics); diagnosticErr != nil {
		return ExitIO
	}
	if err != nil {
		if errors.Is(err, renderhtml.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, "render HTML: "+err.Error())
	}
	if output == "-" {
		if err := writeBytes(e.Out, out.Bytes()); err != nil {
			return failure(e, "write stdout: "+err.Error())
		}
	} else if err := e.WriteOutput(output, sourcePath, out.Bytes()); err != nil {
		return failure(e, fmt.Sprintf("write %q: %v", output, err))
	}
	_, code := assessment(result.Diagnostics)
	return code
}
