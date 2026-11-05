package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/source"
)

const sourcesHelp = `Usage: gtroff sources [OPTIONS] [FILE]

Inspect a non-executing source graph and original-file provenance.
Named files resolve literal local .so/.mso paths by default, relative to each
referring file and confined to --root (default: the input file's directory).
Stdin has no filesystem context unless --root is explicit. Use --no-includes
for a single-file snapshot; directives remain visible with diagnostics.
No macro execution, system macro search, network access or external tools.

  --no-includes           Disable include loading, retaining references
  --root DIR              Explicit allowed source root
  --strict                Reject any syntax/include diagnostics
  --package man|ms|roff   Default: man
  --format json|text      Default: json
  --max-bytes COUNT       Cumulative captured bytes, default: 8388608
  --max-nodes COUNT       Cumulative AST nodes, default: 100000
  --max-files COUNT       Source occurrence cap, default: 256
  --max-depth COUNT       Include depth, default: 16; ceiling: 128

JSON schemaVersion 1 contains sourceSet: files, include edges, logical lines and
diagnostics. Each file retains its original source and AST. Repeated includes
have separate occurrence IDs; byte offsets and node IDs are local to that file.
Logical lines retain the directive before child lines, without executing it.
JSON follows UTF-8 replacement rules; library AST.Source retains exact bytes.
Exit codes: 0 permissive report, 1 strict diagnostics, 2 usage,
3 input/output/resource/cancellation failure (partial report when available).
`

// sourceOptions is the common input policy for workbench commands. Each new
// query/analyze/lint/extract handler should call loadSources rather than invent
// its own include resolver. Existing syntax-only inspection stays separate.
type sourceOptions struct {
	root string
	source.Options
}

// loadSources resolves only capabilities supplied by Environment. The process
// adapter supplies an os.Root-backed OpenRoot; tests can supply virtual roots.
// Without a root capability, named streams retain unresolved include references.
func loadSources(file string, o sourceOptions, e Environment) (*source.Set, error) {
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if file == "" || file == "-" {
		var closer io.Closer
		if o.root != "" && !o.NoIncludes {
			if e.OpenRoot == nil {
				return nil, fmt.Errorf("source root access is unavailable")
			}
			root, c, err := e.OpenRoot(o.root)
			if err != nil {
				return nil, err
			}
			closer = c
			o.FS = root.FS
		}
		set, err := source.Read(ctx, e.In, "<stdin>", o.Options)
		if closer != nil {
			err = errors.Join(err, closer.Close())
		}
		return set, err
	}
	rootName, name := o.root, filepath.Base(file)
	if rootName == "" {
		rootName = filepath.Dir(file)
	} else {
		rootAbs, err := filepath.Abs(rootName)
		if err != nil {
			return nil, err
		}
		fileAbs, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(rootAbs, fileAbs)
		if err != nil || !filepath.IsLocal(relative) {
			return nil, fmt.Errorf("input file is outside the allowed source root")
		}
		name = relative
	}
	if e.OpenRoot != nil {
		root, closer, err := e.OpenRoot(rootName)
		if err != nil {
			return nil, err
		}
		set, err := source.Load(ctx, root.FS, filepath.ToSlash(name), o.Options)
		if closer != nil {
			err = errors.Join(err, closer.Close())
		}
		return set, err
	}
	if o.root != "" {
		return nil, fmt.Errorf("source root access is unavailable")
	}
	if e.Open == nil {
		return nil, fmt.Errorf("file access is unavailable")
	}
	r, err := e.Open(file)
	if err != nil {
		return nil, err
	}
	if sourceio.IsNil(r) {
		return nil, fmt.Errorf("file opener returned a nil reader")
	}
	set, err := source.Read(ctx, r, filepath.ToSlash(file), o.Options)
	return set, errors.Join(err, r.Close())
}

// Sources is a subcommand `gtroff sources` Inspect bounded local includes and source provenance
//
// Usage: gtroff sources [OPTIONS] [FILE]
//
// Named files resolve literal local .so/.mso paths relative to the referring file,
// confined to --root (default: the input file's directory). Stdin needs an explicit
// --root for includes. --no-includes retains a single-file snapshot and references.
// No macros, system macro search, network resources or external tools are executed.
// JSON includes original-file ASTs and provenance; text summarizes the source graph.
// Exit codes: 0 permissive report, 1 strict diagnostics, 2 usage, 3 I/O/resource failure.
//
// Flags:
//
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	root: --root (default: "") Explicit allowed source root
//	strict: --strict (default: false) Reject syntax/include diagnostics
//	pkg: --package (default: "man") Package: man, ms or roff
//	format: --format (default: "json") Output format: json or text
//	maxBytes: --max-bytes (default: 8388608) Cumulative captured source bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative AST node budget
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	files: ... Input file or '-' for stdin
func Sources(noIncludes bool, root string, strict bool, pkg, format string, maxBytes, maxNodes, maxFiles, maxDepth int, files ...string) error {
	o := sourceOptions{root: root, Options: source.Options{Package: pkg, Strict: strict, NoIncludes: noIncludes, MaxBytes: maxBytes, MaxNodes: maxNodes, MaxFiles: maxFiles, MaxDepth: maxDepth}}
	code := runSources(o, format, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

func sourcesCommand(args []string, e Environment) int {
	o := sourceOptions{}
	format := "json"
	f := flag.NewFlagSet("sources", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&o.NoIncludes, "no-includes", false, "")
	f.StringVar(&o.root, "root", "", "")
	f.BoolVar(&o.Strict, "strict", false, "")
	f.StringVar(&o.Package, "package", "man", "")
	f.StringVar(&format, "format", "json", "")
	f.IntVar(&o.MaxBytes, "max-bytes", 8<<20, "")
	f.IntVar(&o.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&o.MaxFiles, "max-files", 256, "")
	f.IntVar(&o.MaxDepth, "max-depth", 16, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, sourcesHelp)
		}
		return usage(e, err.Error())
	}
	return runSources(o, format, f.Args(), e)
}

func runSources(o sourceOptions, format string, files []string, e Environment) int {
	if len(files) > 1 {
		return usage(e, "expected at most one input file")
	}
	if format != "json" && format != "text" {
		return usage(e, "format must be json or text")
	}
	if o.Package != "man" && o.Package != "ms" && o.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if o.MaxBytes < 1 || o.MaxBytes > 64<<20 || o.MaxNodes < 6 || o.MaxNodes > 1000000 || o.MaxFiles < 1 || o.MaxFiles > 4096 || o.MaxDepth < 1 || o.MaxDepth > 128 {
		return usage(e, "source limits outside supported ranges")
	}
	file := "-"
	if len(files) == 1 {
		file = files[0]
	}
	set, loadErr := loadSources(file, o, e)
	status, code := "complete", ExitOK
	if set != nil && len(set.Diagnostics()) > 0 {
		status = "partial"
	}
	if loadErr != nil {
		status = "incomplete"
		code = ExitIO
		if errors.Is(loadErr, source.ErrStrict) {
			status = "invalid"
			code = ExitInvalid
		}
	}
	errorText := ""
	if loadErr != nil {
		errorText = loadErr.Error()
	}
	if format == "json" {
		result := struct {
			SchemaVersion int         `json:"schemaVersion"`
			Command       string      `json:"command"`
			Status        string      `json:"status"`
			Set           *source.Set `json:"sourceSet"`
			Error         string      `json:"error,omitempty"`
		}{1, "sources", status, set, errorText}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return failure(e, err.Error())
		}
		if err := writeBytes(e.Out, append(data, '\n')); err != nil {
			return failure(e, "write source report: "+err.Error())
		}
	} else {
		var out strings.Builder
		fmt.Fprintf(&out, "sources: %s\n", status)
		if set != nil {
			for _, file := range set.Files() {
				fmt.Fprintf(&out, "%d %s depth=%d bytes=%d complete=%t\n", file.ID(), file.Name(), file.Depth(), len(file.AST().Source()), file.AST().SourceComplete())
			}
			for _, edge := range set.Edges() {
				fmt.Fprintf(&out, "%s:%d:%d: %s %s [%s] -> %d\n", edge.Span.Start.Filename, edge.Span.Start.Line, edge.Span.Start.Column, edge.Request, edge.Target, edge.State, edge.To)
			}
			for _, diag := range set.Diagnostics() {
				fmt.Fprintf(&out, "%s:%d:%d: %s: %s\n", diag.Position.Filename, diag.Position.Line, diag.Position.Column, diag.Code, diag.Message)
			}
		}
		if errorText != "" {
			fmt.Fprintln(&out, errorText)
		}
		if c := writeString(e, out.String()); c != ExitOK {
			return c
		}
	}
	return code
}
