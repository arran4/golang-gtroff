package cli

import (
	"context"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/lint"
	"github.com/arran4/golang-gtroff/source"
)

const lintHelp = `Usage: gtroff lint [OPTIONS] [FILE ...]

Apply structural man documentation policy to one file or an explicit collection.
Sources, parser diagnostics, capability analysis and lint findings stay distinct.
Named files resolve bounded local includes; stdin needs an explicit --root.
Rules do not execute roff or external programs and never modify source files.

  --format text|json|jsonl  Default: text; JSONL summary then findings/suppressions
  --rules FILE             Explicit schema-version-1 JSON policy
  --enable IDS             Comma-separated rule IDs to enable (including optional)
  --disable IDS            Comma-separated rule IDs to disable; disable wins
  --required HEADINGS      Comma-separated required sections; config [] disables
  --fail-on auto|error|warning|info|never  Default: error, warning with --strict
  --strict                 Warning threshold, stricter conventions/capabilities,
                           and rejection of all source diagnostics
  --closed-world           Check .MR targets only within a complete named collection
  --no-includes            Inspect one source per input, retaining references
  --root DIR               Allowed root; default: each input's directory
  --package man|ms|roff    Default parser context: man; profile detected separately
  --max-bytes COUNT        Collection captured bytes; default: 8388608
  --max-nodes COUNT        Collection input AST nodes; default: 100000
  --max-files COUNT        Collection source occurrences; default: 256
  --max-depth COUNT        Include depth; default: 16, ceiling 128
  --max-documents COUNT    Root document cap; default: 256
  --max-lint-nodes COUNT   Collection analysis AST nodes; default: 200000
  --max-findings COUNT     Emitted findings including suppressed; default: 100000
  --max-work COUNT         Shared charged work; default: 67108864

Default required sections: NAME, DESCRIPTION; SYNOPSIS warning for manual 1/8.
Strict adds SYNOPSIS for manual 2/3 and unsupported optional-man warnings.
Foreign ms/mdoc/mixed profiles skip man conventions with an info finding.
Incomplete/dynamic source downgrades missing sections to heuristic warnings.
References outside a collection are allowed unless --closed-world is explicit.
Suppress with real source comments: .\" gtroff-lint-disable RULE[,RULE]
Use enable or disable-next-line similarly; all addresses every registered rule.
Suppressions are file-local, retained in reports, and cannot hide input errors.
Exit codes: 0 passes threshold, 1 policy/input invalid, 2 usage/configuration,
3 input/output/resource/cancellation failure; partial reports when available.
`

// Lint is a subcommand `gtroff lint` Apply structural man documentation policy to explicit source snapshots
//
// Usage: gtroff lint [OPTIONS] [FILE ...]
//
// Check a single file or explicit collection without executing roff or editing.
// Named files resolve bounded local includes; stdin needs an explicit root.
// Policy findings remain separate from source diagnostics and capability facts.
// Default man conventions require NAME/DESCRIPTION and suggest command SYNOPSIS.
// Strict uses a warning threshold, stricter conventions and source diagnostics.
// Closed-world reference checks are opt-in and require complete collection input.
// Exit codes: 0 pass, 1 policy/input invalid, 2 usage/config, 3 I/O/resource failure.
//
//	Flags:
//
//	format: --format (default: "text") Output: text, json or jsonl
//	rules: --rules (default: "") Explicit JSON policy file
//	enable: --enable (default: "") Comma-separated rule IDs to enable
//	disable: --disable (default: "") Comma-separated rule IDs to disable
//	required: --required (default: "") Comma-separated required section headings
//	failOn: --fail-on (default: "auto") Threshold: auto, error, warning, info or never
//	strict: --strict (default: false) Stricter policy and all source diagnostics
//	closedWorld: --closed-world (default: false) Check manual references within a complete collection
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	root: --root (default: "") Explicit allowed source root
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative collection captured bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative collection input AST nodes
//	maxFiles: --max-files (default: 256) Cumulative source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxDocuments: --max-documents (default: 256) Root document cap
//	maxLintNodes: --max-lint-nodes (default: 200000) Cumulative analysis AST node budget
//	maxFindings: --max-findings (default: 100000) Maximum findings including suppressed
//	maxWork: --max-work (default: 67108864) Shared charged work budget
//	files: ... Input files or '-' for stdin at most once
func Lint(format string, rules string, enable string, disable string, required string, failOn string, strict bool, closedWorld bool, noIncludes bool, root string, pkg string, maxBytes int, maxNodes int, maxFiles int, maxDepth int, maxDocuments int, maxLintNodes int, maxFindings int, maxWork int, files ...string) error {
	input := sourceOptions{root: root, Options: source.Options{NoIncludes: noIncludes, Strict: strict, Package: pkg, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	options := lint.Options{Strict: strict, FailOn: lint.Severity(failOn), MaxDocuments: maxDocuments, MaxNodes: maxLintNodes, MaxFindings: maxFindings, MaxWork: maxWork}
	code := runLint(lint.Format(format), rules, enable, disable, required, closedWorld, input, options, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}
func lintCommand(args []string, e Environment) int {
	input := sourceOptions{}
	options := lint.Options{}
	var format, rules, enable, disable, required, failOn string
	var closedWorld bool
	f := flag.NewFlagSet("lint", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", "text", "")
	f.StringVar(&rules, "rules", "", "")
	f.StringVar(&enable, "enable", "", "")
	f.StringVar(&disable, "disable", "", "")
	f.StringVar(&required, "required", "", "")
	f.StringVar(&failOn, "fail-on", "auto", "")
	f.BoolVar(&input.Strict, "strict", false, "")
	f.BoolVar(&closedWorld, "closed-world", false, "")
	f.BoolVar(&input.NoIncludes, "no-includes", false, "")
	f.StringVar(&input.root, "root", "", "")
	f.StringVar(&input.Package, "package", "man", "")
	f.IntVar(&input.MaxBytes, "max-bytes", 8<<20, "")
	f.IntVar(&input.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&input.MaxFiles, "max-files", 256, "")
	f.IntVar(&input.MaxDepth, "max-depth", 16, "")
	f.IntVar(&options.MaxDocuments, "max-documents", 256, "")
	f.IntVar(&options.MaxNodes, "max-lint-nodes", 200000, "")
	f.IntVar(&options.MaxFindings, "max-findings", 100000, "")
	f.IntVar(&options.MaxWork, "max-work", 64<<20, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, lintHelp)
		}
		return usage(e, err.Error())
	}
	options.Strict = input.Strict
	options.FailOn = lint.Severity(failOn)
	return runLint(lint.Format(format), rules, enable, disable, required, closedWorld, input, options, f.Args(), e)
}
func runLint(format lint.Format, rulesFile, enable, disable, required string, closedWorld bool, input sourceOptions, options lint.Options, files []string, e Environment) int {
	if err := lint.ValidateFormat(format); err != nil {
		return usage(e, err.Error())
	}
	if options.FailOn == "auto" {
		options.FailOn = ""
	}
	if options.FailOn != "" && options.FailOn != lint.Error && options.FailOn != lint.Warning && options.FailOn != lint.Info && options.FailOn != lint.Never {
		return usage(e, "fail-on must be auto, error, warning, info or never")
	}
	if input.Package != "man" && input.Package != "ms" && input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if input.MaxBytes < 1 || input.MaxBytes > 64<<20 || input.MaxNodes < 6 || input.MaxNodes > 1000000 || input.MaxFiles < 1 || input.MaxFiles > 4096 || input.MaxDepth < 1 || input.MaxDepth > 128 || options.MaxDocuments < 1 || options.MaxDocuments > 4096 || options.MaxNodes < 1 || options.MaxNodes > 1000000 || options.MaxFindings < 1 || options.MaxFindings > 1000000 || options.MaxWork < 1 || options.MaxWork > 1<<30 {
		return usage(e, "lint/source limits outside supported ranges")
	}
	if len(files) == 0 {
		files = []string{"-"}
	}
	if len(files) > options.MaxDocuments {
		return usage(e, "too many root documents")
	}
	stdin := 0
	for _, file := range files {
		if file == "-" || file == "" {
			stdin++
		}
	}
	if stdin > 1 {
		return usage(e, "stdin may be used at most once")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return failure(e, ctx.Err().Error())
	}
	if rulesFile != "" {
		if e.Open == nil {
			return failure(e, "policy file access unavailable")
		}
		reader, err := e.Open(rulesFile)
		if err != nil {
			return failure(e, err.Error())
		}
		if sourceio.IsNil(reader) {
			return failure(e, "policy opener returned nil reader")
		}
		policy, parseErr := lint.ParsePolicy(reader)
		closeErr := reader.Close()
		if closeErr != nil {
			return failure(e, errors.Join(parseErr, closeErr).Error())
		}
		if parseErr != nil {
			if errors.Is(parseErr, lint.ErrConfig) {
				return usage(e, parseErr.Error())
			}
			return failure(e, parseErr.Error())
		}
		options.Policy = policy
	}
	if enable != "" {
		options.Policy.Enable = append(options.Policy.Enable, commaList(enable)...)
	}
	if disable != "" {
		options.Policy.Disable = append(options.Policy.Disable, commaList(disable)...)
	}
	if required != "" {
		options.Policy.RequiredSections = commaList(required)
	}
	if closedWorld {
		options.Policy.ClosedWorld = true
	}
	if err := lint.ValidatePolicy(options.Policy, options.Registry); err != nil {
		return usage(e, err.Error())
	}
	remainingBytes, remainingNodes, remainingFiles := input.MaxBytes, input.MaxNodes, input.MaxFiles
	var documents []lint.Document
	var inputErr error
	for _, file := range files {
		root := input.root
		if root == "" && file != "-" && file != "" {
			root = filepath.Dir(file)
		}
		doc := lint.Document{Name: file, Root: root}
		if file == "-" || file == "" {
			doc.Name = "<stdin>"
		}
		if remainingBytes < 1 || remainingNodes < 6 || remainingFiles < 1 {
			doc.InputError = source.ErrLimit
			documents = append(documents, doc)
			inputErr = errors.Join(inputErr, source.ErrLimit)
			break
		}
		limits := input
		limits.MaxBytes = remainingBytes
		limits.MaxNodes = remainingNodes
		limits.MaxFiles = remainingFiles
		set, loadErr := loadSources(file, limits, e)
		doc.Sources = set
		doc.InputError = loadErr
		documents = append(documents, doc)
		if set != nil {
			for _, captured := range set.Files() {
				remainingFiles--
				remainingBytes -= len(captured.AST().Source())
				captured.AST().Root().Walk(func(*groff.ASTNode) groff.WalkAction { remainingNodes--; return groff.WalkContinue })
			}
		}
		if loadErr != nil && !errors.Is(loadErr, source.ErrStrict) {
			inputErr = errors.Join(inputErr, loadErr)
		}
		if ctx.Err() != nil || errors.Is(loadErr, source.ErrLimit) {
			break
		}
	}
	report, lintErr := lint.Run(ctx, documents, options)
	if err := report.Write(e.Out, format); err != nil {
		return failure(e, "write lint report: "+err.Error())
	}
	if lintErr != nil {
		if errors.Is(lintErr, lint.ErrConfig) {
			return usage(e, lintErr.Error())
		}
		return failure(e, lintErr.Error())
	}
	if inputErr != nil {
		return failure(e, inputErr.Error())
	}
	if report.Failed() {
		return ExitInvalid
	}
	return ExitOK
}
func commaList(value string) []string {
	list := strings.Split(value, ",")
	for i := range list {
		list[i] = strings.TrimSpace(list[i])
	}
	return list
}
