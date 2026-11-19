package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/arran4/golang-gtroff/cmd"
	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/renderhtml"
)

const convertHelp = `Usage: gtroff convert --from html|roff --to roff|html [-i FILE] [-o FILE] [--strict]

Convert a documented semantic subset of HTML and Linux man pages.
-i and -o default to '-' (stdin/stdout). Formats must differ.
HTML output uses the render defaults, including four bundled Go fonts.
--fragment emits a styled article; --assets omit-fonts omits embedded fonts.
Unsupported HTML/CSS is diagnosed and retained as literal text. Strict mode
rejects any diagnostics before writing. Scripts, includes and resources are never
executed or loaded. Source bytes, canonical roff and normalized semantics differ.
Named output uses atomic regular-file replacement and cannot alias the input.
Exit codes: 0 success/warnings, 1 syntax/strict failure, 2 usage,
3 input/output/resource failure. See docs/convert.md for the supported profile.
`

// Convert is a subcommand `gtroff convert` Convert supported HTML and man structure
//
// Usage: gtroff convert --from html|roff --to roff|html [-i FILE] [-o FILE] [--strict]
//
// Convert a documented semantic subset of HTML and Linux man pages.
// -i and -o default to '-' (stdin/stdout). Formats must differ.
// HTML output uses the render defaults, including four bundled Go fonts.
// --fragment emits a styled article; --assets omit-fonts omits embedded fonts.
// Unsupported HTML/CSS is diagnosed and retained as literal text. Strict mode
// rejects any diagnostics before writing. Scripts, includes and resources are never
// executed or loaded. Source bytes, canonical roff and normalized semantics differ.
// Named output uses atomic regular-file replacement and cannot alias the input.
// Exit codes: 0 success/warnings, 1 syntax/strict failure, 2 usage,
// 3 input/output/resource failure. See docs/convert.md for the supported profile.
//
// Flags:
//
//	from: --from Choose html or roff
//	to: --to Choose roff or html
//	input: -i (default: "-") Input file path; '-' means stdin
//	output: -o (default: "-") Output destination; '-' means stdout
//	strict: --strict Reject any diagnostics before writing (default: false)
//	fragment: --fragment Emit a styled article for HTML output (default: false)
//	assets: --assets Font assets policy for HTML: embed or omit-fonts (default: "embed")
func Convert(from, to, input, output string, strict, fragment bool, assets string) error {
	code := runConvert(from, to, input, output, strict, fragment, assets, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func runConvert(from, to, input, destination string, strict, fragment bool, assets string, e Environment) int {
	if !((from == "html" && to == "roff") || (from == "roff" && to == "html")) {
		return usage(e, "choose --from html --to roff or --from roff --to html")
	}
	if input == "" || destination == "" {
		return usage(e, "input/output paths cannot be empty")
	}
	if assets == "" {
		assets = "embed"
	}
	var o conversion.Options
	o.Strict = strict
	o.HTML = renderhtml.Options{Fragment: fragment, Assets: renderhtml.AssetPolicy(assets)}
	if err := renderhtml.ValidateOptions(o.HTML); err != nil {
		return usage(e, err.Error())
	}
	if to != "html" && (fragment || assets != "embed") {
		return usage(e, "fragment/assets options require HTML output")
	}
	if destination != "-" && e.WriteOutput == nil {
		return failure(e, "output file writing is unavailable")
	}
	reader := e.In
	sourcePath := ""
	o.SourceName = "<stdin>"
	var owned io.ReadCloser
	if input != "-" {
		if e.Open == nil {
			return failure(e, "file input is unavailable")
		}
		f, err := e.Open(input)
		if err != nil {
			return failure(e, "open "+strconv.Quote(input)+": "+err.Error())
		}
		reader = f
		owned = f
		sourcePath = input
		o.SourceName = input
	}
	var out bytes.Buffer
	var res conversion.Result
	var err error
	if from == "html" {
		res, err = conversion.HTMLToMan(reader, &out, o)
	} else {
		res, err = conversion.ManToHTML(reader, &out, o)
	}
	if !sourceio.IsNil(owned) {
		if closeErr := owned.Close(); closeErr != nil {
			return failure(e, errors.Join(err, fmt.Errorf("close input: %w", closeErr)).Error())
		}
	}
	if diagErr := writeDiagnostics(e.Err, res.Diagnostics); diagErr != nil {
		return ExitIO
	}
	if err != nil {
		if errors.Is(err, conversion.ErrStrict) || errors.Is(err, renderhtml.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, "convert: "+err.Error())
	}
	if destination == "-" {
		if err := writeBytes(e.Out, out.Bytes()); err != nil {
			return failure(e, "write stdout: "+err.Error())
		}
	} else if err := e.WriteOutput(destination, sourcePath, out.Bytes()); err != nil {
		return failure(e, fmt.Sprintf("write %q: %v", destination, err))
	}
	_, code := assessment(res.Diagnostics)
	return code
}

func convertCommand(args []string, e Environment) int {
	var from, to, input, destination string
	var strict, fragment bool
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&from, "from", "", "")
	fs.StringVar(&to, "to", "", "")
	fs.StringVar(&input, "i", "-", "")
	fs.StringVar(&destination, "o", "-", "")
	fs.BoolVar(&strict, "strict", false, "")
	fs.BoolVar(&fragment, "fragment", false, "")
	assets := fs.String("assets", "embed", "")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, convertHelp)
		}
		return usage(e, err.Error())
	}
	if fs.NArg() != 0 {
		return usage(e, "convert uses -i FILE, not positional files")
	}
	return runConvert(from, to, input, destination, strict, fragment, *assets, e)
}
