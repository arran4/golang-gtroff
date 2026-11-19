package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

const extractHelp = `Usage: gtroff extract [OPTIONS] SELECTOR [FILE]

Export selected source fragments or source-linked structured documentation records.
Example: gtroff extract --format csv --fields term,description,file,line 'section[name="ENVIRONMENT"] definition' document.1
Uses the query selector IR; selection is separate from extraction/serialization.
Term and description, heading context, original spans/IDs/digests and unsupported
content remain distinct. Semantic grouping and text are local to each source.
Named files resolve bounded local includes; stdin needs explicit --root.

  --view semantic|syntax  Default: semantic
  --format FORMAT         json, jsonl, text, raw, grep, csv, tsv or html; default: jsonl
  --fields FIELD,...      CSV/TSV columns; default includes term/description/location
  --normalize             Fold whitespace in text output, preserving case
  -o FILE                 Explicit atomic output; default: - (stdout)
  --strict                Reject source and extraction diagnostics; named files untouched
  --root DIR              Allowed root; default: input directory
  --no-includes           Inspect one file, retaining include references
  --package man|ms|roff   Default parser context: man
  --max-bytes COUNT       Cumulative input bytes; default: 8388608
  --max-nodes COUNT       Cumulative input AST nodes; default: 100000
  --max-files COUNT       Source occurrence cap; default: 256
  --max-depth COUNT       Include depth; default: 16, ceiling 128
  --max-extract-nodes COUNT  Query projection/construction nodes; default: 200000
  --max-records COUNT     Record cap; default: 10000
  --max-record-bytes COUNT  Materialized record-data budget; default: 16777216
  --max-work COUNT        Shared query/construction work; default: 67108864
  --max-output-bytes COUNT  Serialized output budget; default: 33554432

Raw concatenates exact selected per-file spans, with repeated overlap preserved.
Text is syntax text or decoded optional-man text; normalization is not source-exact.
JSON includes the full report; JSONL begins with a summary, then record envelopes.
HTML uses the existing man renderer for selected semantic blocks/terms only.
Unsupported material stays literal with diagnostics. No roff evaluation or edits.
Explicit file output rejects aliases of all loaded input sources and is performed
only after successful construction/serialization. Strict failures retain JSON/
JSONL reports on stdout; other strict output is withheld. Permissive stdout can
contain partial output on resource/I/O errors; always check the exit status.
Exit codes: 0 complete, 1 strict diagnostics, 2 usage/selector, 3 I/O/resource failure.
`

// Extract is a subcommand `gtroff extract` Export selected source fragments and structured documentation records
//
// Usage: gtroff extract [OPTIONS] SELECTOR [FILE]
//
// Uses stateless query selectors; exports original spans or optional man records.
// Definitions retain separate term/description, heading context and provenance.
// Named files resolve bounded includes; stdin needs an explicit allowed root.
// JSONL has a summary and record envelopes. HTML uses the existing man renderer.
// File output is explicit, atomic and protects all loaded original sources.
// Exit codes: 0 complete, 1 strict diagnostics, 2 usage, 3 I/O/resource failure.
//
//	Flags:
//
//	selector: @1 Structural selector expression
//	view: --view (default: "semantic") Select semantic man structure or syntax
//	format: --format (default: "jsonl") json, jsonl, text, raw, grep, csv, tsv or html
//	fields: --fields (default: "") Comma-separated CSV/TSV fields
//	normalize: --normalize (default: false) Fold whitespace in text output, preserving case
//	output: -o (default: "-") Explicit atomic output file or '-' for stdout
//	strict: --strict (default: false) Reject source and extraction diagnostics
//	root: --root (default: "") Explicit allowed source root
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative input bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative input AST nodes
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxExtractNodes: --max-extract-nodes (default: 200000) Projection/construction node budget
//	maxRecords: --max-records (default: 10000) Maximum records before a resource error
//	maxRecordBytes: --max-record-bytes (default: 16777216) Materialized record-data budget
//	maxWork: --max-work (default: 67108864) Shared query/construction work budget
//	maxOutputBytes: --max-output-bytes (default: 33554432) Serialized output byte budget
//	files: ... Input file or '-' for stdin
func Extract(selector string, view string, format string, fields string, normalize bool, output string, strict bool, root string, noIncludes bool, pkg string, maxBytes int, maxNodes int, maxFiles int, maxDepth int, maxExtractNodes int, maxRecords int, maxRecordBytes int, maxWork int, maxOutputBytes int, files ...string) error {
	input := sourceOptions{root: root, Options: source.Options{Package: pkg, Strict: strict, NoIncludes: noIncludes, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	options := extract.Options{Query: query.Options{View: query.View(view)}, Strict: strict, HTML: format == "html", MaxNodes: maxExtractNodes, MaxRecords: maxRecords, MaxBytes: maxRecordBytes, MaxWork: maxWork}
	writeOptions := extract.WriteOptions{Normalize: normalize, MaxBytes: maxOutputBytes}
	if fields != "" {
		writeOptions.Fields = strings.Split(fields, ",")
	}
	code := runExtract(selector, extract.Format(format), output, input, options, writeOptions, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}
func extractCommand(args []string, e Environment) int {
	input := sourceOptions{}
	options := extract.Options{}
	wo := extract.WriteOptions{}
	var view, format, fields, output string
	f := flag.NewFlagSet("extract", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&view, "view", "semantic", "")
	f.StringVar(&format, "format", "jsonl", "")
	f.StringVar(&fields, "fields", "", "")
	f.BoolVar(&wo.Normalize, "normalize", false, "")
	f.StringVar(&output, "o", "-", "")
	f.BoolVar(&input.Strict, "strict", false, "")
	f.StringVar(&input.root, "root", "", "")
	f.BoolVar(&input.NoIncludes, "no-includes", false, "")
	f.StringVar(&input.Package, "package", "man", "")
	f.IntVar(&input.MaxBytes, "max-bytes", 8<<20, "")
	f.IntVar(&input.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&input.MaxFiles, "max-files", 256, "")
	f.IntVar(&input.MaxDepth, "max-depth", 16, "")
	f.IntVar(&options.MaxNodes, "max-extract-nodes", 200000, "")
	f.IntVar(&options.MaxRecords, "max-records", 10000, "")
	f.IntVar(&options.MaxBytes, "max-record-bytes", 16<<20, "")
	f.IntVar(&options.MaxWork, "max-work", 64<<20, "")
	f.IntVar(&wo.MaxBytes, "max-output-bytes", 32<<20, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, extractHelp)
		}
		return usage(e, err.Error())
	}
	if f.NArg() < 1 {
		return usage(e, "extract requires a selector")
	}
	options.Query.View = query.View(view)
	options.Strict = input.Strict
	options.HTML = format == "html"
	if fields != "" {
		wo.Fields = strings.Split(fields, ",")
	}
	return runExtract(f.Arg(0), extract.Format(format), output, input, options, wo, f.Args()[1:], e)
}
func runExtract(expression string, format extract.Format, destination string, input sourceOptions, options extract.Options, wo extract.WriteOptions, files []string, e Environment) int {
	selector, err := query.ParseSelector(expression)
	if err != nil {
		return usage(e, err.Error())
	}
	if err := extract.ValidateFormat(format, wo); err != nil {
		return usage(e, err.Error())
	}
	if len(files) > 1 || options.Query.View != query.Semantic && options.Query.View != query.Syntax {
		return usage(e, "extract expects one input root and view semantic or syntax")
	}
	if format == extract.HTML && options.Query.View != query.Semantic {
		return usage(e, "HTML requires semantic selection")
	}
	if input.Package != "man" && input.Package != "ms" && input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if input.MaxBytes < 1 || input.MaxBytes > 64<<20 || input.MaxNodes < 6 || input.MaxNodes > 1000000 || input.MaxFiles < 1 || input.MaxFiles > 4096 || input.MaxDepth < 1 || input.MaxDepth > 128 || options.MaxNodes < 1 || options.MaxNodes > 1000000 || options.MaxRecords < 1 || options.MaxRecords > 100000 || options.MaxBytes < 1 || options.MaxBytes > 64<<20 || options.MaxWork < 1 || options.MaxWork > 1<<30 || wo.MaxBytes < 1 || wo.MaxBytes > 64<<20 {
		return usage(e, "extraction/source/output limits outside supported ranges")
	}
	if destination == "" {
		return usage(e, "output destination must be nonempty")
	}
	if destination != "-" && e.WriteSourceOutput == nil {
		return failure(e, "protected atomic output capability unavailable")
	}
	file := "-"
	if len(files) == 1 {
		file = files[0]
	}
	set, loadErr := loadSources(file, input, e)
	if set == nil {
		return failure(e, fmt.Sprint(loadErr))
	}
	report, extractErr := extract.Select(e.Context, set, selector, options)
	if loadErr != nil {
		report.InputError = loadErr.Error()
	}
	code := ExitOK
	var operational error
	if extractErr != nil {
		if errors.Is(extractErr, extract.ErrOptions) {
			return usage(e, extractErr.Error())
		}
		if errors.Is(extractErr, extract.ErrStrict) {
			code = ExitInvalid
		} else {
			code = ExitIO
			operational = extractErr
		}
	}
	if loadErr != nil {
		if errors.Is(loadErr, source.ErrStrict) {
			if code == ExitOK {
				code = ExitInvalid
			}
		} else {
			code = ExitIO
			operational = errors.Join(operational, loadErr)
		}
	}
	if format != extract.JSON && format != extract.JSONL || destination != "-" {
		for _, d := range report.Diagnostics {
			if err := writeBytes(e.Err, []byte(fmt.Sprintf("%s:%d:%d: %s %s: %s\n", d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Code, d.Message))); err != nil {
				return ExitIO
			}
		}
	}
	if destination == "-" {
		if code != ExitInvalid || format == extract.JSON || format == extract.JSONL {
			if err := report.Write(e.Context, e.Out, format, wo); err != nil {
				return failure(e, "write extraction: "+err.Error())
			}
		}
	} else if code == ExitOK {
		var data bytes.Buffer
		if err := report.Write(e.Context, &data, format, wo); err != nil {
			return failure(e, "serialize extraction: "+err.Error())
		}
		paths := extractionSourcePaths(file, input.root, set, e.OpenRoot != nil)
		if err := e.WriteSourceOutput(destination, paths, data.Bytes()); err != nil {
			return failure(e, err.Error())
		}
	}
	if operational != nil {
		return failure(e, operational.Error())
	}
	return code
}
func extractionSourcePaths(file, root string, set *source.Set, rooted bool) []string {
	stdin := file == "" || file == "-"
	if root == "" && !stdin {
		root = filepath.Dir(file)
	}
	var paths []string
	seen := map[string]bool{}
	for _, captured := range set.Files() {
		if stdin && captured.ID() == 1 {
			continue
		}
		path := captured.Name()
		if rooted {
			path = filepath.Join(root, filepath.FromSlash(path))
		}
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	return paths
}
