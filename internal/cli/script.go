package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/scriptapi"
	"github.com/arran4/golang-gtroff/source"
)

const scriptHelp = `Usage: gtroff script [OPTIONS] SCRIPT [DOCUMENT]

Run native Tengo with captured roff query/extraction objects. Tengo is isolated in
an optional module/build; the default core binary explains how to enable it.
Build the self-contained variant: go -C script build -o ../gtroff ./cli
No external script runtime, shell, network access, modules or ambient file imports.

SCRIPT and DOCUMENT are explicit user inputs; at most one may be '-' for stdin.
An omitted DOCUMENT gives no current document. Read-only queries are the default.
The doc object provides query, extract, source, sources, quote, preview, emit and
emit_json. Use --allow-edits for doc.edit/create; --include-edits additionally
allows explicitly selected included sources. Selectors remain stateless.

Writes require separate authorization. --write-target FILE permits a staged
explicit doc.write(FILE) request; --in-place explicitly commits pending changes
to original named files. Both run only after successful execution and stdout
serialization. --dry-run suppresses filesystem callbacks. These two destinations
are exclusive. All loaded source hashes and aliases remain protected. New
virtual documents need a copy destination. Each failed run withholds partial
edits, output and requests. Filesystem commit failures report recovery details.

Named documents resolve bounded local includes; --root sets the allowed root
(default: document directory), --no-includes keeps a single-source snapshot.
Script bytes, tokens, nesting, constants, object allocations, host calls, output
and a deadline are bounded. Allocation counts are not heap-byte quotas; native
Tengo has no exact instruction counter or OS sandbox. Native map iteration has
unspecified order; host query arrays and ordinary JSON records are deterministic.

Exit: 0 success, 1 script/selection/capability failure, 2 usage,
3 input/output/resource/cancellation failure. Filesystem writing is explicit.
`

// Script is a subcommand `gtroff script` Execute Tengo queries and optional in-memory transforms
//
// Usage: gtroff script [OPTIONS] SCRIPT [DOCUMENT]
//
// The default core build has no Tengo dependency. Build the optional self-contained
// variant with: go -C script build -o ../gtroff ./cli
// Inputs are explicit and at most one may read stdin. doc query/extract are read
// only; allow-edits grants memory mutation, not file access. File writes require
// an authorized staged doc.write or explicit in-place commit after success.
// Module/file imports, network and shell are not exposed. The VM is not an OS
// sandbox; allocation counts are not heap quotas or exact instruction budgets.
//
// Flags:
//
//	allowEdits: --allow-edits (default: false) Grant in-memory editing; never grants filesystem writes
//	includeEdits: --include-edits (default: false) Allow selected included-file edits with allow-edits
//	writeTarget: --write-target (default: "") Authorize one explicit destination for doc.write requests
//	inPlace: --in-place (default: false) Explicitly commit changed named original files after success
//	dryRun: --dry-run (default: false) Suppress filesystem output, including authorized requests
//	strict: --strict (default: false) Reject source diagnostics and ambiguous edit selections
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	sourceRoot: --root (default: "") Explicit allowed source root for the document
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative input and working document bytes
//	maxNodes: --max-nodes (default: 100000) Cumulative source AST nodes
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxScriptBytes: --max-script-bytes (default: 65536) Script source byte cap
//	maxHostCalls: --max-host-calls (default: 1000) Maximum host method calls
//	maxSteps: --max-steps (default: 100000) Statement/block checkpoints, not VM instructions
//	maxAllocs: --max-allocs (default: 100000) Tengo object allocation count, not heap bytes
//	durationMS: --timeout-ms (default: 1000) Compile and execution deadline in milliseconds
//	maxOutputBytes: --max-output-bytes (default: 16777216) Staged emissions and write-request bytes
//	files: ... Explicit script and optional document; at most one stdin
func Script(allowEdits bool, includeEdits bool, writeTarget string, inPlace bool, dryRun bool, strict bool, noIncludes bool, sourceRoot string, pkg string, maxBytes int, maxNodes int, maxFiles int, maxDepth int, maxScriptBytes int, maxHostCalls int, maxAllocs int, maxSteps int, durationMS int, maxOutputBytes int, files ...string) error {
	c := scriptCLI{}
	c.options.AllowEdits = allowEdits
	c.options.Edit.IncludeEdits = includeEdits
	c.writeTarget = writeTarget
	c.inPlace = inPlace
	c.dryRun = dryRun
	c.input.Strict = strict
	c.input.NoIncludes = noIncludes
	c.input.root = sourceRoot
	c.input.Package = pkg
	c.input.MaxBytes = maxBytes
	c.input.MaxNodes = maxNodes
	c.input.MaxFiles = maxFiles
	c.input.MaxDepth = maxDepth
	c.options.MaxScriptBytes = maxScriptBytes
	c.options.MaxHostCalls = maxHostCalls
	c.maxAllocs = maxAllocs
	c.options.MaxSteps = maxSteps
	c.durationMS = durationMS
	c.options.MaxOutputBytes = maxOutputBytes
	code := runScript(c, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

type scriptCLI struct {
	options               scriptapi.Options
	input                 sourceOptions
	writeTarget           string
	inPlace, dryRun       bool
	maxAllocs, durationMS int
}

func scriptCommand(args []string, e Environment) int {
	c := scriptCLI{}
	f := flag.NewFlagSet("script", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&c.options.AllowEdits, "allow-edits", false, "")
	f.BoolVar(&c.options.Edit.IncludeEdits, "include-edits", false, "")
	f.StringVar(&c.writeTarget, "write-target", "", "")
	f.BoolVar(&c.inPlace, "in-place", false, "")
	f.BoolVar(&c.dryRun, "dry-run", false, "")
	f.BoolVar(&c.input.Strict, "strict", false, "")
	f.BoolVar(&c.input.NoIncludes, "no-includes", false, "")
	f.StringVar(&c.input.root, "root", "", "")
	f.StringVar(&c.input.Package, "package", "man", "")
	f.IntVar(&c.input.MaxBytes, "max-bytes", 8388608, "")
	f.IntVar(&c.input.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&c.input.MaxFiles, "max-files", 256, "")
	f.IntVar(&c.input.MaxDepth, "max-depth", 16, "")
	f.IntVar(&c.options.MaxScriptBytes, "max-script-bytes", 65536, "")
	f.IntVar(&c.options.MaxHostCalls, "max-host-calls", 1000, "")
	f.IntVar(&c.options.MaxSteps, "max-steps", 100000, "")
	f.IntVar(&c.maxAllocs, "max-allocs", 100000, "")
	f.IntVar(&c.durationMS, "timeout-ms", 1000, "")
	f.IntVar(&c.options.MaxOutputBytes, "max-output-bytes", 16777216, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, scriptHelp)
		}
		return usage(e, err.Error())
	}
	return runScript(c, f.Args(), e)
}

func runScript(c scriptCLI, files []string, e Environment) int {
	if len(files) < 1 || len(files) > 2 {
		return usage(e, "script expects SCRIPT and optional DOCUMENT")
	}
	for _, name := range files {
		if name == "" {
			return usage(e, "script/document names must be nonempty")
		}
	}
	if len(files) == 2 && files[0] == "-" && files[1] == "-" {
		return usage(e, "script and document cannot both read stdin")
	}
	if c.inPlace && c.writeTarget != "" {
		return usage(e, "in-place and write-target are exclusive")
	}
	if (c.inPlace || c.options.Edit.IncludeEdits) && !c.options.AllowEdits {
		return usage(e, "in-place/include-edits require allow-edits")
	}
	if c.input.Package != "man" && c.input.Package != "ms" && c.input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if c.input.MaxBytes < 1 || c.input.MaxBytes > 64<<20 || c.input.MaxNodes < 6 || c.input.MaxNodes > 1000000 || c.input.MaxFiles < 1 || c.input.MaxFiles > 4096 || c.input.MaxDepth < 1 || c.input.MaxDepth > 128 || c.options.MaxScriptBytes < 1 || c.options.MaxScriptBytes > 1<<20 || c.options.MaxHostCalls < 1 || c.options.MaxHostCalls > 100000 || c.options.MaxOutputBytes < 1 || c.options.MaxOutputBytes > 64<<20 || c.maxAllocs < 1 || c.maxAllocs > 1000000 || c.options.MaxSteps < 1 || c.options.MaxSteps > 10000000 || c.durationMS < 1 || c.durationMS > 30000 {
		return usage(e, "script/source/resource limits outside supported ranges")
	}
	if c.writeTarget == "-" {
		return usage(e, "write-target must name a file; doc.emit writes stdout")
	}
	if e.Script == nil {
		return failure(e, "Tengo is not included in this build; build the self-contained variant with: go -C script build -o ../gtroff ./cli")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return failure(e, err.Error())
	}
	code, err := readScript(ctx, files[0], c.options.MaxScriptBytes, e)
	if err != nil {
		return failure(e, err.Error())
	}
	var set *source.Set
	inputFile := ""
	if len(files) == 2 {
		inputFile = files[1]
		set, err = loadSources(inputFile, c.input, e)
		if err != nil {
			return editFailure(e, err)
		}
	}
	c.options.AllowWriteRequests = c.writeTarget != ""
	c.options.Edit.Source = c.input.Options
	c.options.Edit.Strict = c.input.Strict
	c.options.Edit.MaxBytes = c.input.MaxBytes
	c.options.MaxAllocs = int64(c.maxAllocs)
	c.options.Duration = time.Duration(c.durationMS) * time.Millisecond
	name := filepath.ToSlash(files[0])
	if name == "-" {
		name = "<stdin-script>"
	}
	result, err := e.Script(ctx, scriptapi.Request{Name: name, Code: code, Sources: set}, c.options)
	if err != nil {
		return scriptFailure(e, err)
	}
	if result == nil || len(result.Output) > c.options.MaxOutputBytes {
		return failure(e, "script backend returned invalid/oversized result")
	}
	if c.inPlace && (result.Working == nil || result.Created) {
		return invalid(e, "in-place requires an original named working document; use write-target for a new document")
	}
	var guards []atomicfile.SourceGuard
	paths := map[string]string{}
	if set != nil {
		paths = editSourcePaths(inputFile, c.input, e, set)
		seen := map[string]bool{}
		for _, f := range set.Files() {
			if seen[f.Name()] {
				continue
			}
			seen[f.Name()] = true
			if path := paths[f.Name()]; path != "" {
				b := []byte(f.AST().Source())
				guards = append(guards, atomicfile.SourceGuard{Path: path, Digest: editSourceDigest(b), Bytes: len(b)})
			}
		}
	}
	if files[0] != "-" {
		programGuard := atomicfile.SourceGuard{Path: files[0], Digest: editSourceDigest(code), Bytes: len(code)}
		programPath, pathErr := filepath.Abs(files[0])
		if pathErr != nil {
			return failure(e, pathErr.Error())
		}
		duplicate := false
		for _, guard := range guards {
			inputPath, pathErr := filepath.Abs(guard.Path)
			if pathErr != nil {
				return failure(e, pathErr.Error())
			}
			if inputPath == programPath {
				if guard.Digest != programGuard.Digest {
					return failure(e, "script/document captures disagree on one input")
				}
				duplicate = true
				break
			}
		}
		if !duplicate {
			guards = append(guards, programGuard)
		}
	}
	if len(result.Writes) > 1 {
		return invalid(e, "one write-target permits at most one explicit copy request")
	}
	for _, request := range result.Writes {
		if c.writeTarget == "" || result.Working == nil {
			return invalid(e, "unapproved script write request")
		}
		allowed, ea := filepath.Abs(c.writeTarget)
		requested, er := filepath.Abs(request.Destination)
		if ea != nil || er != nil || allowed != requested {
			return invalid(e, "script write destination was not explicitly authorized")
		}
		current, err := result.Working.Data(request.Source)
		if err != nil || !bytes.Equal(current, request.Data) {
			return invalid(e, "stale script write source")
		}
	}
	if c.inPlace {
		for _, change := range result.Working.Changes() {
			if paths[change.Source] == "" {
				return invalid(e, "in-place requires named original sources; stdin/new sources need a copy destination")
			}
		}
	}
	if !c.dryRun {
		if len(result.Writes) > 0 && e.WriteEditedOutput == nil {
			return failure(e, "guarded script output capability unavailable")
		}
		if c.inPlace && e.CommitEdits == nil {
			return failure(e, "guarded script commit capability unavailable")
		}
	}
	for _, d := range result.Diagnostics {
		if _, err = fmt.Fprintf(e.Err, "script edit %s: %s", d.Code, d.Message); err == nil {
			if d.Source != "" {
				_, err = fmt.Fprintf(e.Err, " [%s match %d]", d.Source, d.Match)
			}
			if err == nil {
				_, err = fmt.Fprintln(e.Err)
			}
		}
		if err != nil {
			return failure(e, err.Error())
		}
	}

	for _, d := range result.SourceDiagnostics {
		if _, err = fmt.Fprintf(e.Err, "%s:%d:%d: %s: %s\n", d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Message); err != nil {
			return failure(e, err.Error())
		}
	}
	if err = writeEditPreview(e.Out, result.Output); err != nil {
		return failure(e, err.Error())
	}
	if !c.dryRun {
		for _, request := range result.Writes {
			if err = e.WriteEditedOutput(ctx, c.writeTarget, guards, request.Data); err != nil {
				return failure(e, err.Error())
			}
		}
		if c.inPlace {
			err = result.Working.Commit(ctx, func(ctx context.Context, batch edit.CommitBatch) error {
				updates := make([]atomicfile.SourceEdit, 0, len(batch.Changes))
				for _, change := range batch.Changes {
					updates = append(updates, atomicfile.SourceEdit{Path: paths[change.Source], Digest: change.BeforeDigest, Data: change.After})
				}
				return e.CommitEdits(ctx, updates, guards)
			})
			if err != nil {
				return failure(e, err.Error())
			}
		}
	}
	return ExitOK
}
func scriptFailure(e Environment, err error) int {
	if errors.Is(err, scriptapi.ErrLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return failure(e, err.Error())
	}
	return invalid(e, err.Error())
}
func readScript(ctx context.Context, name string, maxBytes int, e Environment) (data []byte, err error) {
	var reader io.Reader = e.In
	if name != "-" {
		if e.Open == nil {
			return nil, fmt.Errorf("script-file read capability unavailable")
		}
		file, openErr := e.Open(name)
		if openErr != nil {
			return nil, openErr
		}
		if sourceio.IsNil(file) {
			return nil, fmt.Errorf("nil script-file reader")
		}
		defer func() { err = errors.Join(err, file.Close()) }()
		reader = file
	}
	if sourceio.IsNil(reader) {
		return nil, fmt.Errorf("nil script reader")
	}
	checked := sourceio.NewReader(reader, func(n, size int, cause error) error {
		return errors.Join(fmt.Errorf("invalid script reader count %d/%d", n, size), cause)
	})
	data, err = io.ReadAll(io.LimitReader(scriptContextReader{ctx, checked}, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, scriptapi.ErrLimit
	}
	return data, ctx.Err()
}

type scriptContextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r scriptContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
