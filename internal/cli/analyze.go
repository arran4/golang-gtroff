package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/source"
)

const analyzeHelp = `Usage: gtroff analyze [OPTIONS] [FILE]

Classify roff syntax, profile evidence, feature usage and processing requirements.
Recognition, typed parsing, optional man interpretation, include resolution and
rendering are separate facts. Analysis never executes macros or external tools.
Named files resolve bounded local includes; stdin needs an explicit --root.

  --format text|json|jsonl  Default: text
  --details                Detailed text evidence; JSON always includes evidence
  --rules FILE             Explicit schema-version-1 declarative JSON questions
  --no-includes            Retain source references without loading them
  --root DIR               Allowed root; default: the input file's directory
  --strict                 Reject syntax/include diagnostics, retaining the report
  --package man|ms|roff    Default parser context: man; profiles are detected separately
  --max-bytes COUNT        Cumulative input bytes; default: 8388608
  --max-nodes COUNT        Cumulative input AST nodes; default: 100000
  --max-files COUNT        Source occurrence cap; default: 256
  --max-depth COUNT        Include depth; default: 16, ceiling: 128
  --max-analysis-nodes COUNT  Enumeration budget; default: 200000
  --max-findings COUNT     Finding cap; default: 100000
  --max-work COUNT         Node/rule/source-byte work budget; default: 67108864

Built-in detectors inventory controls, escapes, definitions, conditionals,
side-effect-looking requests, references, preprocessors, includes and nesting.
Opaque/stored syntax remains visible with active=false. Profile confidence is a
lexical hypothesis; no rendering or runtime compatibility guarantee is implied.
Rules add annotations; they cannot run scripts or override capability facts.
JSONL begins with a summary record, then finding records, each schemaVersion 1.
Exit codes: 0 enumeration complete, 1 strict input diagnostics, 2 usage/rule error,
3 input/output/resource/cancellation failure, with partial reports when available.
`

// Analyze is a subcommand `gtroff analyze` Classify syntax, profiles and processing requirements without execution
//
// Usage: gtroff analyze [OPTIONS] [FILE]
//
// Inventory source features and explain syntax/semantic/include capability facts.
// Named files resolve bounded local includes; stdin needs explicit --root.
// Profile confidence is lexical evidence, not successful execution or rendering.
// JSON/JSONL preserve detailed source evidence; text defaults to a concise summary.
// Exit codes: 0 complete, 1 strict diagnostics, 2 usage/rules, 3 I/O/resource failure.
//
// Flags:
//
//	format: --format (default: "text") Output: text, json or jsonl
//	details: --details (default: false) Detailed evidence in text output
//	rules: --rules (default: "") Explicit declarative JSON rules file
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	root: --root (default: "") Explicit allowed source root
//	strict: --strict (default: false) Reject syntax/include diagnostics
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative input bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative input AST nodes
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxAnalysisNodes: --max-analysis-nodes (default: 200000) Analysis enumeration node budget
//	maxFindings: --max-findings (default: 100000) Maximum findings before a resource error
//	maxWork: --max-work (default: 67108864) Node/rule/source-byte work budget
//	files: ... Input file or '-' for stdin
func Analyze(format string, details bool, rules string, noIncludes bool, root string, strict bool, pkg string, maxBytes, maxNodes, maxFiles, maxDepth, maxAnalysisNodes, maxFindings, maxWork int, files ...string) error {
	input := sourceOptions{root: root, Options: source.Options{NoIncludes: noIncludes, Strict: strict, Package: pkg, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	options := analyze.Options{MaxNodes: maxAnalysisNodes, MaxFindings: maxFindings, MaxWork: maxWork}
	code := runAnalyze(analyze.Format(format), details, rules, input, options, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}
func analyzeCommand(args []string, e Environment) int {
	input := sourceOptions{}
	options := analyze.Options{}
	var format, rules string
	var details bool
	flags := flag.NewFlagSet("analyze", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&format, "format", "text", "")
	flags.BoolVar(&details, "details", false, "")
	flags.StringVar(&rules, "rules", "", "")
	flags.BoolVar(&input.NoIncludes, "no-includes", false, "")
	flags.StringVar(&input.root, "root", "", "")
	flags.BoolVar(&input.Strict, "strict", false, "")
	flags.StringVar(&input.Package, "package", "man", "")
	flags.IntVar(&input.MaxBytes, "max-bytes", 8<<20, "")
	flags.IntVar(&input.MaxNodes, "max-nodes", 100000, "")
	flags.IntVar(&input.MaxFiles, "max-files", 256, "")
	flags.IntVar(&input.MaxDepth, "max-depth", 16, "")
	flags.IntVar(&options.MaxNodes, "max-analysis-nodes", 200000, "")
	flags.IntVar(&options.MaxFindings, "max-findings", 100000, "")
	flags.IntVar(&options.MaxWork, "max-work", 64<<20, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, analyzeHelp)
		}
		return usage(e, err.Error())
	}
	return runAnalyze(analyze.Format(format), details, rules, input, options, flags.Args(), e)
}
func runAnalyze(format analyze.Format, details bool, rulesFile string, input sourceOptions, options analyze.Options, files []string, e Environment) int {
	if err := analyze.ValidateFormat(format, details); err != nil {
		return usage(e, err.Error())
	}
	if len(files) > 1 {
		return usage(e, "expected at most one input file")
	}
	if input.Package != "man" && input.Package != "ms" && input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if input.MaxBytes < 1 || input.MaxBytes > 64<<20 || input.MaxNodes < 6 || input.MaxNodes > 1000000 || input.MaxFiles < 1 || input.MaxFiles > 4096 || input.MaxDepth < 1 || input.MaxDepth > 128 || options.MaxNodes < 1 || options.MaxNodes > 1000000 || options.MaxFindings < 1 || options.MaxFindings > 1000000 || options.MaxWork < 1 || options.MaxWork > 1<<30 {
		return usage(e, "analysis/source limits outside supported ranges")
	}
	if e.Context != nil && e.Context.Err() != nil {
		return failure(e, e.Context.Err().Error())
	}
	if rulesFile != "" {
		if e.Open == nil {
			return failure(e, "rule file access unavailable")
		}
		reader, err := e.Open(rulesFile)
		if err != nil {
			return failure(e, err.Error())
		}
		if sourceio.IsNil(reader) {
			return failure(e, "rules opener returned nil reader")
		}
		registry, parseErr := analyze.ParseRules(reader)
		closeErr := reader.Close()
		if closeErr != nil {
			return failure(e, errors.Join(parseErr, closeErr).Error())
		}
		if parseErr != nil {
			if errors.Is(parseErr, analyze.ErrRules) {
				return usage(e, parseErr.Error())
			}
			return failure(e, parseErr.Error())
		}
		options.Registry = registry
	}
	file := "-"
	if len(files) == 1 {
		file = files[0]
	}
	set, loadErr := loadSources(file, input, e)
	if set == nil {
		return failure(e, fmt.Sprint(loadErr))
	}
	report, analysisErr := analyze.Run(e.Context, set, options)
	if loadErr != nil {
		report.InputError = loadErr.Error()
	}
	if err := report.Write(e.Out, format, details); err != nil {
		return failure(e, "write analysis: "+err.Error())
	}
	if analysisErr != nil {
		return failure(e, analysisErr.Error())
	}
	if loadErr != nil {
		if errors.Is(loadErr, source.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, loadErr.Error())
	}
	return ExitOK
}
