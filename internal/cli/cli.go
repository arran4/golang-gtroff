// Package cli implements the headless command boundary using explicit streams
// and caller-supplied file/root openers. Source loading uses explicitly bounded
// roots; no command evaluates roff.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/scriptapi"
)

const (
	ExitOK      = 0
	ExitInvalid = 1
	ExitUsage   = 2
	ExitIO      = 3
)

// BuildInfo is supplied by the executable, without ambient environment lookups.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// OutputFile is one explicitly named encoded image page.
type OutputFile struct {
	Path string
	Data []byte
}

// Environment contains only the capabilities used by these commands. Open must
// return an owned reader; named input is closed before any output is written.
// In is borrowed and is never closed. Out/Err are borrowed writers.
type Environment struct {
	// ScriptSession supplies the optional persistent engine; nil disables REPL.
	ScriptSession scriptapi.SessionFactory
	Script        scriptapi.Backend
	Context       context.Context
	OpenRoot      func(string) (manlib.Root, io.Closer, error)
	Listen        func(string) (net.Listener, error)
	SystemSearch  func() (manlib.SearchProvider, error)
	In            io.Reader
	Out, Err      io.Writer
	Open          func(string) (io.ReadCloser, error)
	WriteOutput   func(destination, source string, data []byte) error
	// WriteSourceOutput protects all loaded source paths during explicit export.
	WriteSourceOutput func(destination string, sources []string, data []byte) error
	CommitEdits       func(context.Context, []atomicfile.SourceEdit, []atomicfile.SourceGuard) error
	WriteEditedOutput func(context.Context, string, []atomicfile.SourceGuard, []byte) error
	WriteOutputs      func(source string, files []OutputFile) error
	Build             BuildInfo
}

var defaultEnv = DefaultEnvironment()

// SetDefaultEnvironment configures the fallback process environment for generated command entrypoints.
func SetDefaultEnvironment(env Environment) {
	defaultEnv = env
}

// DefaultEnvironment constructs a baseline environment using standard process streams and files.
func DefaultEnvironment() Environment {
	return Environment{
		In:   os.Stdin,
		Out:  os.Stdout,
		Err:  os.Stderr,
		Open: func(path string) (io.ReadCloser, error) { return os.Open(path) },
	}
}

func getEnv(ops ...any) Environment {
	for _, op := range ops {
		if e, ok := op.(Environment); ok {
			return e
		}
	}
	return defaultEnv
}

const rootHelp = `Usage: gtroff COMMAND [OPTIONS] [FILE]

Commands:
  validate  Check roff syntax and report diagnostics
  ast       Inspect a source-aware syntax tree
  sources   Inspect bounded includes and source provenance
  query     Select source-aware documentation
  extract   Export fragments and structured documentation
  edit      Plan in-memory transformations and explicit writes
  script    Run optional Tengo with explicit document capabilities
  render    Export a semantic man page to HTML or paginated images
  convert   Convert supported HTML and man structure
  serve     Browse and search explicit man-page directories over HTTP
  man       Generate a man page from command help
  completion  Generate shell completions
  version   Show build information
  help      Show help for a command

Use FILE '-' or omit FILE to read stdin. Output is written to stdout.
Run 'gtroff help validate' or 'gtroff help ast' for options.
Validation checks syntax only; it does not establish typesetting success.
No macros are evaluated. sources resolves bounded local includes; no network access.
`
const validateHelp = `Usage: gtroff validate [--strict] [--package man|ms|roff] [--format text|json] [FILE]

Check syntax, preserving unknown and malformed input in diagnostics.
FILE defaults to '-' (stdin). The default package is man and format is text.
Strict mode rejects warnings, including unknown controls and unloaded includes.
Permissive unknown syntax reports partial support.
This does not validate man-page semantics or guarantee successful typesetting.
Includes, macros and external commands are never executed.

Exit codes: 0 accepted syntax (possibly warnings), 1 invalid syntax,
2 usage/selection error, 3 input/output/resource failure.
JSON schemaVersion 1 includes diagnostics; text diagnostics go to stderr.
Output goes to stdout only; use shell redirection to save it.
`
const astHelp = `Usage: gtroff ast [--strict] [--package man|ms|roff] [--format json|text] [--select SELECTOR] [FILE]

Inspect syntax with original source, stable document-local IDs and byte spans.
FILE defaults to '-' (stdin). The default package is man and format is json.
Selectors: node:ID, path:/0/1, section:HEADING. Repeated headings select all matches.
Selection is exact and case-sensitive; quote a selector containing spaces.
Selected subtrees retain original IDs and parent links; external parents are listed
in externalParentIds. The source field always contains the complete input snapshot.
JSON schemaVersion 1 includes diagnostics and an ordered node table.
Text output lists nodes in preorder; diagnostics go to stderr.
Includes, macros and external commands are never executed.

Exit codes: 0 accepted syntax (possibly warnings), 1 invalid syntax,
2 usage/selection error, 3 input/output/resource failure.
Even invalid syntax produces a preserved AST. Output is stdout only.
`

// Run executes one command without reading or mutating process globals.
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
	if len(args) == 0 {
		return writeString(e, rootHelp)
	}
	command := args[0]
	args = args[1:]
	switch command {
	case "help", "--help", "-h":
		if command != "help" && len(args) > 0 {
			return usage(e, "help takes no arguments")
		}
		if len(args) == 0 {
			return writeString(e, rootHelp)
		}
		if len(args) == 2 && args[0] == "image" && args[1] == "generate" {
			return writeString(e, imageGenerateHelp)
		}
		if len(args) != 1 {
			return usage(e, "help expects at most one command")
		}
		return commandHelp(e, args[0])
	case "version", "--version":
		if len(args) != 0 {
			return usage(e, "version takes no arguments")
		}
		return writeString(e, fmt.Sprintf("gtroff %s (commit %s, built %s)\n", e.Build.Version, e.Build.Commit, e.Build.Date))
	case "convert":
		return convertCommand(args, e)
	case "serve":
		return serveCommand(args, e)
	case "render":
		return renderCommand(args, e)
	case "image":
		return imageCommand(args, e)
	case "view":
		return viewCommand(args, e)
	case "commands":
		return commandsCommand(args, e)
	case "sources":
		return sourcesCommand(args, e)
	case "query":
		return queryCommand(args, e)
	case "extract":
		return extractCommand(args, e)
	case "edit":
		return editCommand(args, e)
	case "script":
		return scriptCommand(args, e)
	case "validate", "ast":
		return inspectCommand(command, args, e)
	case "man":
		return manCommand(args, e)
	case "completion":
		return completionCommand(args, e)
	default:
		return usage(e, "unknown command "+strconv.Quote(command))
	}
}
func commandHelp(e Environment, command string) int {
	switch command {
	case "validate":
		return writeString(e, validateHelp)
	case "ast":
		return writeString(e, astHelp)
	case "sources":
		return writeString(e, sourcesHelp)
	case "query":
		return writeString(e, queryHelp)
	case "extract":
		return writeString(e, extractHelp)
	case "edit":
		return writeString(e, editHelp)
	case "script":
		return writeString(e, scriptHelp)
	case "convert":
		return writeString(e, convertHelp)
	case "serve":
		return writeString(e, serveHelp)
	case "render":
		return writeString(e, renderHelp)
	case "image":
		return writeString(e, imageHelp)
	case "view":
		return writeString(e, viewHelp)
	case "commands":
		return writeString(e, commandsHelp)
	case "version":
		return writeString(e, "Usage: gtroff version\nShow build version, commit and date.\n")
	case "man":
		return writeString(e, manHelp)
	case "completion":
		return writeString(e, completionHelp)
	default:
		return usage(e, "unknown command "+strconv.Quote(command))
	}
}
func writeString(e Environment, s string) int {
	if err := writeBytes(e.Out, []byte(s)); err != nil {
		return failure(e, "write stdout: "+err.Error())
	}
	return ExitOK
}
func failure(e Environment, message string) int {
	fmt.Fprintln(e.Err, "gtroff: "+message)
	return ExitIO
}
func usage(e Environment, message string) int {
	if err := writeBytes(e.Err, []byte("gtroff: "+message+"\nRun 'gtroff help' for usage.\n")); err != nil {
		return ExitIO
	}
	return ExitUsage
}

type inspectOptions struct {
	strict                bool
	pkg, format, selector string
}

func inspectCommand(command string, args []string, e Environment) int {
	o := inspectOptions{pkg: "man", format: "text"}
	if command == "ast" {
		o.format = "json"
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&o.strict, "strict", false, "")
	flags.StringVar(&o.pkg, "package", o.pkg, "")
	flags.StringVar(&o.format, "format", o.format, "")
	if command == "ast" {
		flags.StringVar(&o.selector, "select", "", "")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return commandHelp(e, command)
		}
		return usage(e, err.Error())
	}
	return runInspect(command, o, flags.Args(), e)
}

func runInspect(command string, o inspectOptions, files []string, e Environment) int {
	if len(files) > 1 {
		return usage(e, "expected at most one input file")
	}
	if o.pkg == "" {
		o.pkg = "man"
	}
	if o.pkg != "man" && o.pkg != "ms" && o.pkg != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if o.format == "" {
		if command == "ast" {
			o.format = "json"
		} else {
			o.format = "text"
		}
	}
	if o.format != "json" && o.format != "text" {
		return usage(e, "format must be json or text")
	}
	if command == "ast" {
		if err := validateSelector(o.selector); err != nil {
			return usage(e, err.Error())
		}
	}
	source := "<stdin>"
	var reader io.Reader = e.In
	var owned io.ReadCloser
	if len(files) == 1 && files[0] != "-" {
		source = files[0]
		if e.Open == nil {
			return failure(e, "open "+strconv.Quote(source)+": file access is unavailable")
		}
		f, err := e.Open(source)
		if err != nil {
			return failure(e, "open "+strconv.Quote(source)+": "+err.Error())
		}
		owned = f
		reader = f
	}
	tree, parseErr := groff.ParseAST(reader, groff.WithPackage(o.pkg), groff.WithStrict(false), groff.WithSourceName(source))
	if owned != nil {
		if err := owned.Close(); err != nil {
			if parseErr == nil {
				parseErr = fmt.Errorf("close input: %w", err)
			} else {
				parseErr = fmt.Errorf("%v; close input: %w", parseErr, err)
			}
		}
	}
	if tree == nil {
		return failure(e, fmt.Sprintf("read input: %v", parseErr))
	}
	diagnostics := tree.Diagnostics()
	// A successful generated token is not evidence that an include was loaded.
	// Report preserved include operations as unsupported without opening targets.
	for _, n := range tree.Root().FindKind(groff.ASTLine) {
		attrs := n.Attributes()
		if attrs.Opaque {
			continue
		}
		if attrs.Name == ".so" || attrs.Name == ".mso" {
			diagnostics = append(diagnostics, groff.Diagnostic{Message: "include preserved; target was not loaded", Position: n.Span().Start, Severity: "warning"})
		}
	}
	sortDiagnostics(diagnostics)
	status, code := assessment(diagnostics)
	// ParseAST's permissive return error is reserved for I/O/resource failure;
	// syntax errors remain structured diagnostics. Strictness is a CLI policy.
	if parseErr != nil {
		status = "incomplete"
		code = ExitIO
	}
	if o.strict && status == "partial" {
		status = "invalid"
		code = ExitInvalid
	}
	if command == "validate" {
		return outputValidation(e, source, o, diagnostics, status, code, parseErr)
	}
	roots, err := selectRoots(tree, o.selector)
	if err != nil {
		return usage(e, err.Error())
	}
	return outputAST(e, source, o, tree, roots, diagnostics, status, code, parseErr)
}

// Root is a subcommand `gtroff` Linux groff and man page processing tool
//
// Extended Help:
//
// Usage: gtroff COMMAND [OPTIONS] [FILE]
//
// Commands:
//
//	validate  Check roff syntax and report diagnostics
//	ast       Inspect a source-aware syntax tree
//	sources   Inspect bounded includes and source provenance
//	query     Select source-aware documentation
//	extract   Export fragments and structured documentation
//	edit      Plan in-memory transformations and explicit writes
//	script    Run optional Tengo with explicit document capabilities
//	render    Export a semantic man page to HTML or paginated images
//	convert   Convert supported HTML and man structure
//	serve     Browse and search explicit man-page directories over HTTP
//	man       Generate a man page from command help
//	completion  Generate shell completions
//	version   Show build information
//	help      Show help for a command
//
// Use FILE '-' or omit FILE to read stdin. Output is written to stdout.
// Run 'gtroff help validate' or 'gtroff help ast' for options.
// Validation checks syntax only; it does not establish typesetting success.
// No macros are evaluated. sources resolves bounded local includes; no network access.
func Root() error {
	return nil
}

// Validate is a subcommand `gtroff validate` Check roff syntax and report diagnostics
//
// Usage: gtroff validate [--strict] [--package man|ms|roff] [--format text|json] [FILE]
//
// Check syntax, preserving unknown and malformed input in diagnostics.
// FILE defaults to '-' (stdin). The default package is man and format is text.
// Strict mode rejects warnings, including unknown controls and unloaded includes.
// Permissive unknown syntax reports partial support.
// This does not validate man-page semantics or guarantee successful typesetting.
// Includes, macros and external commands are never executed.
//
// Exit codes: 0 accepted syntax (possibly warnings), 1 invalid syntax,
// 2 usage/selection error, 3 input/output/resource failure.
// JSON schemaVersion 1 includes diagnostics; text diagnostics go to stderr.
// Output goes to stdout only; use shell redirection to save it.
//
// Flags:
//
//	strict: --strict (default: false) Reject warnings, including unknown controls and unloaded includes
//	pkg: --package (default: "man") Package must be man, ms or roff
//	format: --format (default: "text") Format must be text or json
//	files: ... Input file, defaults to stdin ('-')
func Validate(strict bool, pkg, format string, files ...string) error {
	code := runInspect("validate", inspectOptions{strict: strict, pkg: pkg, format: format}, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

// Ast is a subcommand `gtroff ast` Inspect a source-aware syntax tree
//
// Usage: gtroff ast [--strict] [--package man|ms|roff] [--format json|text] [--select SELECTOR] [FILE]
//
// Inspect syntax with original source, stable document-local IDs and byte spans.
// FILE defaults to '-' (stdin). The default package is man and format is json.
// Selectors: node:ID, path:/0/1, section:HEADING. Repeated headings select all matches.
// Selection is exact and case-sensitive; quote a selector containing spaces.
// Selected subtrees retain original IDs and parent links; external parents are listed
// in externalParentIds. The source field always contains the complete input snapshot.
// JSON schemaVersion 1 includes diagnostics and an ordered node table.
// Text output lists nodes in preorder; diagnostics go to stderr.
// Includes, macros and external commands are never executed.
//
// Exit codes: 0 accepted syntax (possibly warnings), 1 invalid syntax,
// 2 usage/selection error, 3 input/output/resource failure.
// Even invalid syntax produces a preserved AST. Output is stdout only.
//
// Flags:
//
//	strict: --strict (default: false) Reject warnings, including unknown controls and unloaded includes
//	pkg: --package (default: "man") Package must be man, ms or roff
//	format: --format (default: "json") Format must be json or text
//	selector: --select (default: "") Select syntax nodes: node:ID, path:/0/1, section:HEADING
//	files: ... Input file, defaults to stdin ('-')
func Ast(strict bool, pkg, format, selector string, files ...string) error {
	code := runInspect("ast", inspectOptions{strict: strict, pkg: pkg, format: format, selector: selector}, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func assessment(diagnostics []groff.Diagnostic) (string, int) {
	status, code := "valid", ExitOK
	for _, d := range diagnostics {
		if d.Severity == "error" {
			return "invalid", ExitInvalid
		}
		if d.Severity == "warning" {
			status = "partial"
		}
	}
	return status, code
}
func validateSelector(s string) error {
	if s == "" {
		return nil
	}
	typ, value, ok := strings.Cut(s, ":")
	if !ok || value == "" {
		return fmt.Errorf("select must be node:ID, path:/0/1 or section:HEADING")
	}
	switch typ {
	case "node":
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil || id == 0 {
			return fmt.Errorf("node selector requires a positive ID")
		}
	case "path":
		if value == "/" {
			return nil
		}
		if !strings.HasPrefix(value, "/") {
			return fmt.Errorf("path selector must start with '/'")
		}
		for _, part := range strings.Split(value[1:], "/") {
			if _, err := strconv.ParseUint(part, 10, 31); err != nil {
				return fmt.Errorf("path selector requires nonnegative child indexes")
			}
		}
	case "section":
	default:
		return fmt.Errorf("unknown selector %q", typ)
	}
	return nil
}
func selectRoots(a *groff.AST, s string) ([]*groff.ASTNode, error) {
	if s == "" {
		return []*groff.ASTNode{a.Root()}, nil
	}
	typ, value, _ := strings.Cut(s, ":")
	var roots []*groff.ASTNode
	switch typ {
	case "node":
		id, _ := strconv.ParseUint(value, 10, 64)
		if n := a.Node(groff.NodeID(id)); n != nil {
			roots = append(roots, n)
		}
	case "section":
		roots = a.Root().FindSection(value, false)
	case "path":
		n := a.Root()
		if value != "/" {
			for _, part := range strings.Split(value[1:], "/") {
				index, _ := strconv.ParseUint(part, 10, 31)
				children := n.Children()
				if index >= uint64(len(children)) {
					n = nil
					break
				}
				n = children[index]
			}
		}
		if n != nil {
			roots = append(roots, n)
		}
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("selection %q matched no nodes", s)
	}
	return roots, nil
}
