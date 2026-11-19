package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

const queryHelp = `Usage: gtroff query [OPTIONS] SELECTOR [FILE]

Run a stateless structural selector over source syntax or an optional man hierarchy.
Example: gtroff query 'section[name="ENVIRONMENT"] definition' document.1
Steps use whitespace for descendants and > for direct children. Predicates use
[field=value], [field*=substring] or numeric comparisons such as [line>=2].
Quote values with spaces; escape double-quoted values using Go string syntax.
Kinds, fields, quoting and semantic boundaries are documented in docs/query.md.

  --view semantic|syntax  Default: semantic (man conventions)
  --format FORMAT         json, jsonl, text, raw, grep, csv or tsv; default: json
  --fields FIELD,...      Selected CSV/TSV fields; default: index,file,line,column,kind,term,text
  --no-includes           Disable local include loading, retaining references
  --root DIR              Allowed root; named-file default is its containing directory
  --strict                Reject syntax/include diagnostics while retaining results
  --package man|ms|roff   Default: man; semantic view uses man conventions
  --max-bytes COUNT       Cumulative input bytes; default: 8388608
  --max-nodes COUNT       Cumulative input AST nodes; default: 100000
  --max-files COUNT       Source occurrence cap; default: 256
  --max-depth COUNT       Include depth; default: 16, ceiling: 128
  --max-results COUNT     Match cap; default: 10000
  --max-work COUNT        Query work budget; default: 67108864
  --max-result-bytes COUNT  Materialized match-data budget; default: 16777216

Named files resolve bounded local includes by default; stdin needs explicit --root.
Each result retains the original file/occurrence, span, IDs, source digest and a
one-based ordinal. Included sources are descendants of their directive's owner;
semantic grouping stays local to each file. Unknown/opaque source remains visible.
Raw output concatenates selected source spans; overlapping matches repeat bytes.
JSONL contains matches only; full JSON also contains global diagnostics/completeness.
No query state, mutation, macro execution, network access or ambient library FS.
Exit codes: 0 query complete (possibly no matches), 1 strict input diagnostics,
2 selector/usage error, 3 I/O/resource/cancellation failure with a partial report.
`

// Query is a subcommand `gtroff query` Select source-aware documentation with stateless structural queries
//
// Usage: gtroff query [OPTIONS] SELECTOR [FILE]
//
// Example: gtroff query 'section[name="ENVIRONMENT"] definition' document.1
// Select semantic man structure or physical syntax without mutating documents.
// Named files resolve bounded local includes; stdin needs an explicit --root.
// Included sources retain original-file provenance; semantic grouping is file-local.
// Use whitespace for descendants, > for children and [field=value] predicates.
// Exit codes: 0 complete, 1 strict diagnostics, 2 usage/selector, 3 I/O/resource failure.
//
// Flags:
//
//	selector: @1 Structural selector expression
//	view: --view (default: "semantic") Select semantic man structure or syntax
//	format: --format (default: "json") json, jsonl, text, raw, grep, csv or tsv
//	fields: --fields (default: "") Comma-separated CSV/TSV fields
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	root: --root (default: "") Explicit allowed source root
//	strict: --strict (default: false) Reject syntax/include diagnostics
//	pkg: --package (default: "man") Package: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative input bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative input AST nodes
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxResults: --max-results (default: 10000) Maximum matches before a resource error
//	maxWork: --max-work (default: 67108864) Query node/predicate/text-byte work budget
//	maxResultBytes: --max-result-bytes (default: 16777216) Materialized match-data budget
//	files: ... Input file or '-' for stdin
func Query(selector, view, format, fields string, noIncludes bool, root string, strict bool, pkg string, maxBytes, maxNodes, maxFiles, maxDepth, maxResults, maxWork, maxResultBytes int, files ...string) error {
	input := sourceOptions{root: root, Options: source.Options{NoIncludes: noIncludes, Strict: strict, Package: pkg, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	options := query.Options{View: query.View(view), MaxMatches: maxResults, MaxWork: maxWork, MaxResultBytes: maxResultBytes}
	code := runQuery(selector, query.Format(format), fields, input, options, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}
func queryCommand(args []string, e Environment) int {
	input := sourceOptions{}
	options := query.Options{}
	var view, format, fields string
	flags := flag.NewFlagSet("query", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&view, "view", "semantic", "")
	flags.StringVar(&format, "format", "json", "")
	flags.StringVar(&fields, "fields", "", "")
	flags.BoolVar(&input.NoIncludes, "no-includes", false, "")
	flags.StringVar(&input.root, "root", "", "")
	flags.BoolVar(&input.Strict, "strict", false, "")
	flags.StringVar(&input.Package, "package", "man", "")
	flags.IntVar(&input.MaxBytes, "max-bytes", 8<<20, "")
	flags.IntVar(&input.MaxNodes, "max-nodes", 100000, "")
	flags.IntVar(&input.MaxFiles, "max-files", 256, "")
	flags.IntVar(&input.MaxDepth, "max-depth", 16, "")
	flags.IntVar(&options.MaxMatches, "max-results", 10000, "")
	flags.IntVar(&options.MaxWork, "max-work", 64<<20, "")
	flags.IntVar(&options.MaxResultBytes, "max-result-bytes", 16<<20, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, queryHelp)
		}
		return usage(e, err.Error())
	}
	if flags.NArg() == 0 {
		return usage(e, "query requires a selector")
	}
	options.View = query.View(view)
	return runQuery(flags.Arg(0), query.Format(format), fields, input, options, flags.Args()[1:], e)
}
func runQuery(expression string, format query.Format, fieldList string, input sourceOptions, options query.Options, files []string, e Environment) int {
	selector, err := query.ParseSelector(expression)
	if err != nil {
		return usage(e, err.Error())
	}
	var fields []string
	if fieldList != "" {
		fields = strings.Split(fieldList, ",")
	}
	if err := query.ValidateFormat(format, fields); err != nil {
		return usage(e, err.Error())
	}
	if len(files) > 1 || options.View != query.Syntax && options.View != query.Semantic {
		return usage(e, "query expects one file and view syntax or semantic")
	}
	if input.Package != "man" && input.Package != "ms" && input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if input.MaxBytes < 1 || input.MaxBytes > 64<<20 || input.MaxNodes < 6 || input.MaxNodes > 1000000 || input.MaxFiles < 1 || input.MaxFiles > 4096 || input.MaxDepth < 1 || input.MaxDepth > 128 || options.MaxMatches < 1 || options.MaxMatches > 100000 || options.MaxWork < 1 || options.MaxWork > 1<<30 || options.MaxResultBytes < 1 || options.MaxResultBytes > 64<<20 {
		return usage(e, "query/source limits outside supported ranges")
	}
	file := "-"
	if len(files) == 1 {
		file = files[0]
	}
	set, loadErr := loadSources(file, input, e)
	if set == nil {
		return failure(e, fmt.Sprint(loadErr))
	}
	result, queryErr := query.Execute(e.Context, set, selector, options)
	if loadErr != nil {
		result.InputError = loadErr.Error()
	}
	if err := result.Write(e.Out, format, fields); err != nil {
		return failure(e, "write query results: "+err.Error())
	}
	if format != query.JSON {
		for _, diagnostic := range result.Diagnostics {
			if _, err := fmt.Fprintf(e.Err, "%s:%d:%d: %s: %s\n", diagnostic.Position.Filename, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Code, diagnostic.Message); err != nil {
				return ExitIO
			}
		}
	}
	if queryErr != nil {
		return failure(e, queryErr.Error())
	}
	if loadErr != nil {
		if errors.Is(loadErr, source.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, loadErr.Error())
	}
	return ExitOK
}
