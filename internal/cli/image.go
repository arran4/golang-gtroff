package cli

import (
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
)

const imageHelp = `Usage: gtroff image COMMAND [OPTIONS] [FILE]

Export a man page to paginated PNG or JPEG images with bundled Go fonts.

Commands:
  generate  Export a man page to paginated PNG or JPEG images

Run 'gtroff help image generate' for options.
`

const imageGenerateHelp = `Usage: gtroff image generate [--format png|jpg] [OPTIONS] [FILE]

Export common Linux man syntax to paginated PNG or JPEG with bundled Go fonts.
FILE defaults to '-' (stdin). No includes or external resources are loaded.

  --format png|jpg        Output image format, default: png
  -o FILE                Output image file; '-' means stdout
  --theme light|dark      Default: light
  --title TEXT            Override the document title
  --section HEADING       Select all matching sections/subsections
  --node ID               Select an AST source ID shown by gtroff ast
  --strict               Reject any semantic/render diagnostic before writing
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
  --font FILE             Alias for --font-regular
  --font-regular FILE     Explicit custom TTF/OTF; other faces keep bundled defaults
  --font-bold FILE        Explicit custom bold font
  --font-italic FILE      Explicit custom italic font
  --font-mono FILE        Explicit custom mono font

Images on stdout require one page. Multi-page foo.png output uses foo-001.png,
foo-002.png, etc. Files are all staged before per-file atomic replacements.
Diagnostics go to stderr. Exit codes: 0 success/warnings, 1 syntax/strict failure,
2 usage/selection, 3 input/output/resource failure.
`

// Image is a subcommand `gtroff image` Export man pages to paginated raster images
//
// Usage: gtroff image COMMAND [OPTIONS] [FILE]
//
// Export a man page to paginated PNG or JPEG images with bundled Go fonts.
//
// Commands:
//   generate  Export a man page to paginated PNG or JPEG images
//
// Run 'gtroff help image generate' for options.
func Image() error {
	return cmd.ErrPrintHelp
}

// ImageGenerate is a subcommand `gtroff image generate` Export a man page to PNG or JPEG images
//
// Usage: gtroff image generate [--format png|jpg] [OPTIONS] [FILE]
//
// Export common Linux man syntax to paginated PNG or JPEG with bundled Go fonts.
// FILE defaults to '-' (stdin). No includes or external resources are loaded.
//
//   --format png|jpg        Output image format, default: png
//   -o FILE                Output image file; '-' means stdout
//   --theme light|dark      Default: light
//   --title TEXT            Override the document title
//   --section HEADING       Select all matching sections/subsections
//   --node ID               Select an AST source ID shown by gtroff ast
//   --strict               Reject any semantic/render diagnostic before writing
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
//   --font FILE             Alias for --font-regular
//   --font-regular FILE     Explicit custom TTF/OTF; other faces keep bundled defaults
//   --font-bold FILE        Explicit custom bold font
//   --font-italic FILE      Explicit custom italic font
//   --font-mono FILE        Explicit custom mono font
//
// Images on stdout require one page. Multi-page foo.png output uses foo-001.png,
// foo-002.png, etc. Files are all staged before per-file atomic replacements.
// Diagnostics go to stderr. Exit codes: 0 success/warnings, 1 syntax/strict failure,
// 2 usage/selection, 3 input/output/resource failure.
//
// Flags:
//
//	format: --format (default: "png") Output format: png or jpg
//	output: -o (default: "-") Output image file; '-' means stdout
//	theme: --theme (default: "light") Default: light
//	title: --title (default: "") Override the document title
//	section: --section (default: "") Select all matching sections/subsections
//	node: --node (default: "") Select an AST source ID shown by gtroff ast
//	strict: --strict (default: false) Reject any semantic/render diagnostic before writing
//	width: --width (default: 1000) 128..8192, default 1000
//	margin: --margin (default: 48) Nonnegative, default 48
//	fontSize: --font-size (default: 15) Points 6..72, default 15
//	dpi: --dpi (default: 96) 36..300, default 96 pixels/inch
//	lineHeight: --line-height (default: 1.35) 1..3, default 1.35 for images
//	pageHeight: --page-height (default: 1400) 128..8192, default 1400
//	single: --single (default: false) Derive one image's height within guarded limits
//	maxHeight: --max-height (default: 32768) 128..32768, default 32768
//	maxPixels: --max-pixels (default: 67108864) Total across all pages; default/hard ceiling 67108864
//	quality: --quality (default: 90) JPEG 1..100, default 90
//	fontRegular: --font-regular, --font (default: "") Explicit custom TTF/OTF font
//	fontBold: --font-bold (default: "") Explicit custom bold font
//	fontItalic: --font-italic (default: "") Explicit custom italic font
//	fontMono: --font-mono (default: "") Explicit custom mono font
//	files: ... Input file, '-' or omitted for stdin
func ImageGenerate(format, output, theme, title, section, node string, strict bool, width, margin int, fontSize, dpi, lineHeight float64, pageHeight int, single bool, maxHeight int, maxPixels int64, quality int, fontRegular, fontBold, fontItalic, fontMono string, files ...string) error {
	code := runImageGenerate(format, output, theme, title, section, node, strict, width, margin, fontSize, dpi, lineHeight, pageHeight, single, maxHeight, maxPixels, quality, fontRegular, fontBold, fontItalic, fontMono, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func runImageGenerate(format, output, theme, title, section, node string, strict bool, width, margin int, fontSize, dpi, lineHeight float64, pageHeight int, single bool, maxHeight int, maxPixels int64, quality int, fontRegular, fontBold, fontItalic, fontMono string, files []string, e Environment) int {
	if format == "" {
		format = "png"
	}
	format = strings.ToLower(format)
	if format != "png" && format != "jpg" && format != "jpeg" {
		return usage(e, "image format must be png or jpg")
	}
	if len(files) > 1 {
		return usage(e, "expected at most one input file")
	}
	file := "-"
	if len(files) == 1 {
		file = files[0]
	}
	args := []string{"--format", format, "-o", output}
	if theme != "" {
		args = append(args, "--theme", theme)
	}
	if title != "" {
		args = append(args, "--title", title)
	}
	if section != "" {
		args = append(args, "--section", section)
	}
	if node != "" {
		args = append(args, "--node", node)
	}
	if strict {
		args = append(args, "--strict")
	}
	if width != 0 {
		args = append(args, "--width", strconv.Itoa(width))
	}
	if margin != 0 {
		args = append(args, "--margin", strconv.Itoa(margin))
	}
	if fontSize != 0 {
		args = append(args, "--font-size", strconv.FormatFloat(fontSize, 'f', -1, 64))
	}
	if dpi != 0 {
		args = append(args, "--dpi", strconv.FormatFloat(dpi, 'f', -1, 64))
	}
	if lineHeight != 0 {
		args = append(args, "--line-height", strconv.FormatFloat(lineHeight, 'f', -1, 64))
	}
	if pageHeight != 0 {
		args = append(args, "--page-height", strconv.Itoa(pageHeight))
	}
	if single {
		args = append(args, "--single")
	}
	if maxHeight != 0 {
		args = append(args, "--max-height", strconv.Itoa(maxHeight))
	}
	if maxPixels != 0 {
		args = append(args, "--max-pixels", strconv.FormatInt(maxPixels, 10))
	}
	if quality != 0 {
		args = append(args, "--quality", strconv.Itoa(quality))
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
	if file != "" {
		args = append(args, file)
	}
	return renderCommand(args, e)
}

func imageCommand(args []string, e Environment) int {
	if len(args) == 0 {
		return writeString(e, imageHelp)
	}
	switch args[0] {
	case "help", "--help", "-h":
		if len(args) > 1 && args[1] == "generate" {
			return writeString(e, imageGenerateHelp)
		}
		return writeString(e, imageHelp)
	case "generate":
		var f string
		for i, a := range args[1:] {
			if a == "--format" && i+1 < len(args[1:]) {
				f = args[1:][i+1]
			} else if strings.HasPrefix(a, "--format=") {
				f = strings.TrimPrefix(a, "--format=")
			}
		}
		if f != "" && f != "png" && f != "jpg" && f != "jpeg" {
			return usage(e, "image format must be png or jpg")
		}
		return renderCommand(append([]string{"--format", "png"}, args[1:]...), e)
	default:
		return usage(e, "unknown command "+strconv.Quote("image "+args[0]))
	}
}
