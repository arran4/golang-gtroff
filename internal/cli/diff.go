package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/docdiff"
	"github.com/arran4/golang-gtroff/source"
)

const diffHelp = `Usage: gtroff diff [OPTIONS] BEFORE AFTER

Compare man content projections or original source bytes with both source locations.
Sections/subsections, definitions, metadata, references and preamble content use
existing query/extraction models. Default semantic view ignores font, indentation
and flowing whitespace; preformatted text and opaque bytes remain significant.
Unique identities can move across scopes; sibling reorder uses relative ordering.
Duplicates are reported as ambiguity, without pairing by ordinal. Unsupported
regions remain explicit source evidence, not assertions about runtime meaning.

  --view semantic|source  Default: semantic
  --format text|json|jsonl Default: text; JSONL summary/change/ambiguity records
  --strict                Reject source/model/opaque/ambiguity diagnostics
  --root DIR              Shared allowed root; default: each input's directory
  --no-includes           Compare only the explicit root sources and references
  --package man|ms|roff   Default parser context: man; projection is the man subset
  --max-bytes COUNT       Input bytes across BOTH loads; default: 8388608
  --max-nodes COUNT       Input AST nodes across BOTH loads; default: 100000
  --max-files COUNT       Source occurrences across BOTH loads; default: 256
  --max-depth COUNT       Include depth; default: 16, ceiling 128
  --max-diff-nodes COUNT  Per-revision projection bound; default: 200000
  --max-objects COUNT     Per-revision object cap; default: 10000
  --max-changes COUNT     Changes plus ambiguities; default: 10000
  --max-data-bytes COUNT  Comparison data budget; default: 33554432
  --max-result-bytes COUNT Comparison result data; default: 16777216
  --max-work COUNT        Shared input/projection/matching work; default: 67108864
  --max-output-bytes COUNT Serialized output; default: 33554432

Stdin '-' is accepted for one side. Named roots resolve bounded local includes;
stdin needs --root for includes. No roff execution, network fetching or writes.
Missing/truncated/broken includes refuse comparison rather than invent removals;
use --no-includes explicitly for a root-only source view. Semantic source errors
also refuse comparison; source view can compare fully captured malformed bytes.
Root filenames are labels, so old.1/new.1 does not create a semantic rename.
Included occurrences use directive ancestry/target/occurrence context.
Exit codes: 0 equal in the selected assessment, 1 differences/ambiguity/strict,
2 usage, 3 input/output/resource/cancellation failure with a partial report.
`

// Diff is a subcommand `gtroff diff` Compare documentation revisions with original-source provenance
//
// Usage: gtroff diff [OPTIONS] BEFORE AFTER
//
// Semantic view compares existing man content projections; source view compares bytes.
// Known formatting-only edits are ignored in semantic view. Opaque source and
// ambiguous duplicate identities remain explicit. Named roots resolve bounded
// local includes; incomplete input does not become invented content removals.
// Comparison never executes macros or external commands and never writes files.
// Exit codes: 0 equal, 1 changed/ambiguous/strict, 2 usage, 3 I/O/resource failure.
//
//	Flags:
//
//	view: --view (default: "semantic") Compare semantic man content or source bytes
//	format: --format (default: "text") Output: text, json or jsonl
//	strict: --strict (default: false) Reject source/model/ambiguity diagnostics
//	root: --root (default: "") Shared explicit allowed source root
//	noIncludes: --no-includes (default: false) Compare explicit root sources without loading includes
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Input bytes across both roots
//	maxNodes: --max-nodes (default: 100000) Input AST nodes across both roots
//	maxFiles: --max-files (default: 256) Source occurrences across both roots
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxDiffNodes: --max-diff-nodes (default: 200000) Per-revision projection node bound
//	maxObjects: --max-objects (default: 10000) Per-revision object cap
//	maxChanges: --max-changes (default: 10000) Changes plus ambiguities cap
//	maxDataBytes: --max-data-bytes (default: 33554432) Comparison data budget
//	maxResultBytes: --max-result-bytes (default: 16777216) Materialized change data budget
//	maxWork: --max-work (default: 67108864) Shared comparison work budget
//	maxOutputBytes: --max-output-bytes (default: 33554432) Serialized output byte budget
//	files: ... Exactly two inputs, '-' allowed once
func Diff(view string, format string, strict bool, root string, noIncludes bool, pkg string, maxBytes int, maxNodes int, maxFiles int, maxDepth int, maxDiffNodes int, maxObjects int, maxChanges int, maxDataBytes int, maxResultBytes int, maxWork int, maxOutputBytes int, files ...string) error {
	input := sourceOptions{root: root, Options: source.Options{Strict: strict, NoIncludes: noIncludes, Package: pkg, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	options := docdiff.Options{View: docdiff.View(view), Strict: strict, MaxNodes: maxDiffNodes, MaxObjects: maxObjects, MaxChanges: maxChanges, MaxDataBytes: maxDataBytes, MaxResultBytes: maxResultBytes, MaxWork: maxWork}
	code := runDiff(docdiff.Format(format), maxOutputBytes, input, options, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}
func diffCommand(args []string, e Environment) int {
	input := sourceOptions{}
	options := docdiff.Options{}
	var view, format string
	var maxOutputBytes int
	f := flag.NewFlagSet("diff", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&view, "view", "semantic", "")
	f.StringVar(&format, "format", "text", "")
	f.BoolVar(&input.Strict, "strict", false, "")
	f.StringVar(&input.root, "root", "", "")
	f.BoolVar(&input.NoIncludes, "no-includes", false, "")
	f.StringVar(&input.Package, "package", "man", "")
	f.IntVar(&input.MaxBytes, "max-bytes", 8<<20, "")
	f.IntVar(&input.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&input.MaxFiles, "max-files", 256, "")
	f.IntVar(&input.MaxDepth, "max-depth", 16, "")
	f.IntVar(&options.MaxNodes, "max-diff-nodes", 200000, "")
	f.IntVar(&options.MaxObjects, "max-objects", 10000, "")
	f.IntVar(&options.MaxChanges, "max-changes", 10000, "")
	f.IntVar(&options.MaxDataBytes, "max-data-bytes", 32<<20, "")
	f.IntVar(&options.MaxResultBytes, "max-result-bytes", 16<<20, "")
	f.IntVar(&options.MaxWork, "max-work", 64<<20, "")
	f.IntVar(&maxOutputBytes, "max-output-bytes", 32<<20, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, diffHelp)
		}
		return usage(e, err.Error())
	}
	options.View = docdiff.View(view)
	options.Strict = input.Strict
	return runDiff(docdiff.Format(format), maxOutputBytes, input, options, f.Args(), e)
}
func runDiff(format docdiff.Format, maxOutput int, input sourceOptions, options docdiff.Options, files []string, e Environment) int {
	if err := docdiff.ValidateFormat(format); err != nil {
		return usage(e, err.Error())
	}
	if len(files) != 2 {
		return usage(e, "diff requires exactly BEFORE and AFTER")
	}
	if files[0] == "" || files[1] == "" {
		return usage(e, "input names must be nonempty; use - for stdin")
	}
	if files[0] == "-" && files[1] == "-" {
		return usage(e, "stdin may be used for only one side")
	}
	if options.View != docdiff.Semantic && options.View != docdiff.Source {
		return usage(e, "view must be semantic or source")
	}
	if input.Package != "man" && input.Package != "ms" && input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if input.MaxBytes < 1 || input.MaxBytes > 64<<20 || input.MaxNodes < 6 || input.MaxNodes > 1000000 || input.MaxFiles < 1 || input.MaxFiles > 4096 || input.MaxDepth < 1 || input.MaxDepth > 128 || options.MaxNodes < 1 || options.MaxNodes > 1000000 || options.MaxObjects < 1 || options.MaxObjects > 100000 || options.MaxChanges < 1 || options.MaxChanges > 100000 || options.MaxDataBytes < 1 || options.MaxDataBytes > 64<<20 || options.MaxResultBytes < 1 || options.MaxResultBytes > 64<<20 || options.MaxWork < 1 || options.MaxWork > 1<<30 || maxOutput < 1 || maxOutput > 64<<20 {
		return usage(e, "diff/source/output limits outside supported ranges")
	}
	left := docdiff.Input{Name: files[0]}
	right := docdiff.Input{Name: files[1]}
	remaining := input
	left.Sources, left.Error = loadSources(files[0], remaining, e)
	consumeSourceSnapshot(&remaining, left.Sources)
	if left.Error != nil && !errors.Is(left.Error, source.ErrStrict) {
		right.Error = fmt.Errorf("after input was not loaded following before-input failure")
	} else if remaining.MaxBytes < 1 || remaining.MaxNodes < 6 || remaining.MaxFiles < 1 {
		right.Error = source.ErrLimit
	} else {
		right.Sources, right.Error = loadSources(files[1], remaining, e)
	}
	report, compareErr := docdiff.Compare(e.Context, left, right, options)
	if err := report.Write(e.Context, e.Out, format, maxOutput); err != nil {
		return failure(e, "write diff: "+err.Error())
	}
	if compareErr != nil {
		if errors.Is(compareErr, docdiff.ErrStrict) || errors.Is(compareErr, source.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, compareErr.Error())
	}
	if report.Different || len(report.Ambiguities) > 0 {
		return ExitInvalid
	}
	return ExitOK
}
