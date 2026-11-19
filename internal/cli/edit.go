package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/cmd"
	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/sourceio"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

const editHelp = `Usage: gtroff edit [OPTIONS] [FILE]

Build an in-memory, source-backed patch plan and preview it on stdout.
Use --operation with --select, or --commands with a schema-v1 JSON command file.
Commands execute sequentially; later selectors see earlier candidate changes.
Default selection is source[source-id=1]. Non-strict broadcasts all matches with
warnings; --nth N, --range A:B, --first N or --all restrict one-based query indices.
Strict requires exactly one result after restrictions, including explicit --all.
Included files require --include-edits. Repeated physical patches apply once.

Operations: replace-source, find-replace-source (--find), delete, insert-before,
insert-after, move-before/move-after (--destination unique selector),
append-paragraph (--text), append-section (--heading, optional --text),
append-definition (--term, --text), replace-description (--text).
Source operations use exact bytes; named operations escape literal text and emit
canonical fragments. Complex descriptions require explicit --rewrite. Unchanged
bytes, unknown requests, comments and line endings remain exact.

Default --format diff is a quoted byte-range review, not a patch(1) script.
--format json includes base64 bytes, hashes, patches and diagnostics.
--format raw emits a single current physical source, never include expansion.
Use --source NAME to explicitly select a physical source for raw/copy output.
--create constructs a new document (optional --title and --section) without reads.

Writing is explicit: -o FILE writes one transformed source copy, rejecting aliases
of any loaded source; --in-place replaces changed originals with captured-hash
checks. -o and --in-place are exclusive. --dry-run prevents both from writing.
All originals are checked; aliases/symlinks/stale source/overlaps are errors.
Writes stage/sync first and roll back ordinary rename failures. Failed restores
retain recovery backups and report paths. Per-file rename is atomic; group crash
atomicity/concurrent-writer protection is not promised. Use exclusive write access.

Local includes load by default for named files, confined to --root (default:
input directory). --no-includes retains references; stdin needs explicit --root.
No macros, shell commands, network access or uncaptured files are executed/opened.
Exit: 0 success (possibly warnings), 1 selection/validation/ownership failure,
2 usage/selector/configuration, 3 input/output/resource/cancellation failure.
`

// Edit is a subcommand `gtroff edit` Plan source-preserving in-memory transformations and explicit writes
//
// Usage: gtroff edit [OPTIONS] [FILE]
//
// Default output previews a source-backed patch plan without writing files.
// Non-strict broadcasts all matches; strict requires one after restrictions.
// Includes require explicit edit ownership. Source bytes remain separate from
// canonical literal fragments. Use -o or --in-place for explicit guarded output;
// --dry-run suppresses writes. Per-file renames are atomic, not group crash-atomic.
// Operations: replace-source, find-replace-source, delete, insert-before,
// insert-after, move-before, move-after, append-paragraph, append-section,
// append-definition, replace-description. Named operations escape literal data;
// source operations edit exact bytes. Complex descriptions need --rewrite.
// No roff execution, shell, network or implicit filesystem capability.
//
// Flags:
//
//	selection: --select (default: "source[source-id=1]") Stateless selector for each command
//	operation: --operation (default: "") Transformation operation
//	text: --text (default: "") Literal data or exact source bytes, depending on operation
//	find: --find (default: "") Nonempty exact source bytes to replace
//	heading: --heading (default: "") Literal heading for append-section
//	term: --term (default: "") Literal term for append-definition
//	destination: --destination (default: "") Unique move destination selector
//	view: --view (default: "semantic") Select semantic man structure or syntax
//	nth: --nth (default: -1) One-based selected match; omit to leave unset
//	matchRange: --range (default: "") Inclusive one-based match range, e.g. 2:5
//	first: --first (default: -1) First N matches; omit to leave unset
//	all: --all (default: false) Explicitly select all matches
//	rewrite: --rewrite (default: false) Permit canonical replacement of complex description source
//	commands: --commands (default: "") Explicit schema-v1 JSON command file, maximum 64 KiB
//	create: --create (default: false) Create a new in-memory document without reading an input
//	title: --title (default: "") Literal title for a new document
//	section: --section (default: "") Manual section for a new document
//	format: --format (default: "diff") Review diff, JSON or exact raw source on stdout
//	sourceName: --source (default: "") Explicit captured physical source for raw/copy output
//	output: --o (default: "-") Explicit transformed source copy destination
//	inPlace: --in-place (default: false) Explicit guarded replacement of changed original files
//	dryRun: --dry-run (default: false) Preview without filesystem writes, even with output flags
//	includeEdits: --include-edits (default: false) Explicitly allow selected included-file edits
//	strict: --strict (default: false) Require one match and reject source diagnostics
//	noIncludes: --no-includes (default: false) Disable include loading, retaining references
//	sourceRoot: --root (default: "") Explicit allowed source root
//	pkg: --package (default: "man") Parser context: man, ms or roff
//	maxBytes: --max-bytes (default: 8388608) Cumulative source bytes and working-file budget
//	maxNodes: --max-nodes (default: 100000) Cumulative parser node budget
//	maxFiles: --max-files (default: 256) Source occurrence cap
//	maxDepth: --max-depth (default: 16) Include depth, ceiling 128
//	maxPatches: --max-patches (default: 10000) Maximum patches and query matches
//	maxWork: --max-work (default: 268435456) Cumulative query, byte-processing and reparse work
//	maxOutputBytes: --max-output-bytes (default: 33554432) Serialized preview/output byte cap
//	files: ... Input file, new source name with --create, or stdin
func Edit(selection string, operation string, text string, find string, heading string, term string, destination string, view string, nth int, matchRange string, first int, all bool, rewrite bool, commands string, create bool, title string, section string, format string, sourceName string, output string, inPlace bool, dryRun bool, includeEdits bool, strict bool, noIncludes bool, sourceRoot string, pkg string, maxBytes int, maxNodes int, maxFiles int, maxDepth int, maxPatches int, maxWork int, maxOutputBytes int, files ...string) error {
	c := editCLI{}
	c.spec.Selector = selection
	c.op = operation
	c.spec.Text = text
	c.spec.Find = find
	c.spec.Heading = heading
	c.spec.Term = term
	c.spec.Destination = destination
	c.spec.View = query.View(view)
	c.nth = nth
	c.matchRange = matchRange
	c.first = first
	c.spec.Restrict.All = all
	c.spec.Rewrite = rewrite
	c.commands = commands
	c.create = create
	c.title = title
	c.section = section
	c.format = format
	c.sourceName = sourceName
	c.output = output
	c.inPlace = inPlace
	c.dryRun = dryRun
	c.options.IncludeEdits = includeEdits
	c.input.Strict = strict
	c.input.NoIncludes = noIncludes
	c.input.root = sourceRoot
	c.input.Package = pkg
	c.input.MaxBytes = maxBytes
	c.input.MaxNodes = maxNodes
	c.input.MaxFiles = maxFiles
	c.input.MaxDepth = maxDepth
	c.options.MaxPatches = maxPatches
	c.options.MaxWork = maxWork
	c.maxOutputBytes = maxOutputBytes
	code := runEdit(c, files, getEnv())
	if code != ExitOK {
		return &cmd.ErrExitCode{Code: code}
	}
	return nil
}

type editCLI struct {
	spec                                       edit.CommandSpec
	op                                         string
	nth, first                                 int
	matchRange, commands                       string
	create                                     bool
	title, section, format, sourceName, output string
	inPlace, dryRun                            bool
	input                                      sourceOptions
	options                                    edit.Options
	maxOutputBytes                             int
}

func editCommand(args []string, e Environment) int {
	c := editCLI{}
	f := flag.NewFlagSet("edit", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var view string
	f.StringVar(&c.spec.Selector, "select", "source[source-id=1]", "")
	f.StringVar(&c.op, "operation", "", "")
	f.StringVar(&c.spec.Text, "text", "", "")
	f.StringVar(&c.spec.Find, "find", "", "")
	f.StringVar(&c.spec.Heading, "heading", "", "")
	f.StringVar(&c.spec.Term, "term", "", "")
	f.StringVar(&c.spec.Destination, "destination", "", "")
	f.StringVar(&view, "view", "semantic", "")
	f.IntVar(&c.nth, "nth", -1, "")
	f.StringVar(&c.matchRange, "range", "", "")
	f.IntVar(&c.first, "first", -1, "")
	f.BoolVar(&c.spec.Restrict.All, "all", false, "")
	f.BoolVar(&c.spec.Rewrite, "rewrite", false, "")
	f.StringVar(&c.commands, "commands", "", "")
	f.BoolVar(&c.create, "create", false, "")
	f.StringVar(&c.title, "title", "", "")
	f.StringVar(&c.section, "section", "", "")
	f.StringVar(&c.format, "format", "diff", "")
	f.StringVar(&c.sourceName, "source", "", "")
	f.StringVar(&c.output, "o", "-", "")
	f.BoolVar(&c.inPlace, "in-place", false, "")
	f.BoolVar(&c.dryRun, "dry-run", false, "")
	f.BoolVar(&c.options.IncludeEdits, "include-edits", false, "")
	f.BoolVar(&c.input.Strict, "strict", false, "")
	f.BoolVar(&c.input.NoIncludes, "no-includes", false, "")
	f.StringVar(&c.input.root, "root", "", "")
	f.StringVar(&c.input.Package, "package", "man", "")
	f.IntVar(&c.input.MaxBytes, "max-bytes", 8388608, "")
	f.IntVar(&c.input.MaxNodes, "max-nodes", 100000, "")
	f.IntVar(&c.input.MaxFiles, "max-files", 256, "")
	f.IntVar(&c.input.MaxDepth, "max-depth", 16, "")
	f.IntVar(&c.options.MaxPatches, "max-patches", 10000, "")
	f.IntVar(&c.options.MaxWork, "max-work", 268435456, "")
	f.IntVar(&c.maxOutputBytes, "max-output-bytes", 33554432, "")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeString(e, editHelp)
		}
		return usage(e, err.Error())
	}
	c.spec.View = query.View(view)
	return runEdit(c, f.Args(), e)
}

func runEdit(c editCLI, files []string, e Environment) int {
	if len(files) > 1 || c.output == "" || c.inPlace && c.output != "-" {
		return usage(e, "edit expects one input and exclusive -o/--in-place")
	}
	if c.format != "diff" && c.format != "json" && c.format != "raw" {
		return usage(e, "edit format must be diff, json or raw")
	}
	if c.input.Package != "man" && c.input.Package != "ms" && c.input.Package != "roff" {
		return usage(e, "package must be man, ms or roff")
	}
	if c.input.MaxBytes < 1 || c.input.MaxBytes > 64<<20 || c.input.MaxNodes < 6 || c.input.MaxNodes > 1000000 || c.input.MaxFiles < 1 || c.input.MaxFiles > 4096 || c.input.MaxDepth < 1 || c.input.MaxDepth > 128 || c.options.MaxPatches < 1 || c.options.MaxPatches > 100000 || c.options.MaxWork < 1 || c.options.MaxWork > 1<<30 || c.maxOutputBytes < 1 || c.maxOutputBytes > 128<<20 {
		return usage(e, "edit source/work/output limits outside supported ranges")
	}
	if c.nth != -1 && c.nth < 1 || c.first != -1 && c.first < 1 {
		return usage(e, "nth/first must be positive one-based indices")
	}
	if c.nth != -1 {
		c.spec.Restrict.Nth = c.nth
	}
	if c.first != -1 {
		c.spec.Restrict.FirstN = c.first
	}
	if c.matchRange != "" {
		parts := strings.Split(c.matchRange, ":")
		if len(parts) != 2 {
			return usage(e, "range must be inclusive A:B")
		}
		a, ea := strconv.Atoi(parts[0])
		b, eb := strconv.Atoi(parts[1])
		if ea != nil || eb != nil || a < 1 || b < a {
			return usage(e, "range must use positive increasing A:B")
		}
		c.spec.Restrict.From = a
		c.spec.Restrict.To = b
	}
	if c.create && c.inPlace {
		return usage(e, "new documents require explicit -o destination, not --in-place")
	}
	if !c.create && (c.title != "" || c.section != "") {
		return usage(e, "title/section require --create")
	}
	ctx := e.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return failure(e, err.Error())
	}
	var commands []edit.Command
	if c.commands != "" {
		if c.op != "" || c.spec.Selector != "source[source-id=1]" || c.spec.Text != "" || c.spec.Find != "" || c.spec.Heading != "" || c.spec.Term != "" || c.spec.Destination != "" || c.spec.Rewrite || c.spec.View != query.Semantic || c.nth != -1 || c.first != -1 || c.matchRange != "" || c.spec.Restrict.All {
			return usage(e, "commands file is exclusive with single-command flags")
		}
		if e.Open == nil {
			return failure(e, "command-file read capability unavailable")
		}
		r, err := e.Open(c.commands)
		if err != nil {
			return failure(e, err.Error())
		}
		if sourceio.IsNil(r) {
			return failure(e, "nil command-file reader")
		}
		b, readErr := io.ReadAll(io.LimitReader(r, 65537))
		err = errors.Join(readErr, r.Close())
		if err != nil {
			return failure(e, err.Error())
		}
		commands, err = edit.ParseCommands(b)
		if err != nil {
			return usage(e, err.Error())
		}
	} else {
		c.spec.Operation = edit.Operation(c.op)
		command, err := c.spec.Command()
		if err != nil {
			return usage(e, err.Error())
		}
		commands = []edit.Command{command}
	}
	c.options.Source = c.input.Options
	c.options.Strict = c.input.Strict
	c.options.MaxBytes = c.input.MaxBytes
	inputFile := "-"
	if len(files) == 1 {
		inputFile = files[0]
		if inputFile == "" {
			return usage(e, "input name must be nonempty")
		}
	}
	var w *edit.Working
	var set *source.Set
	var err error
	if c.create {
		name := inputFile
		if name == "-" {
			name = "untitled.1"
		}
		w, err = edit.Create(ctx, filepath.ToSlash(name), man.Metadata{Title: c.title, Section: c.section}, c.options)
	} else {
		set, err = loadSources(inputFile, c.input, e)
		if err == nil {
			w, err = edit.New(set, c.options)
		}
	}
	if err != nil {
		return editFailure(e, err)
	}
	plan, err := w.Plan(ctx, commands...)
	if err != nil {
		return editFailure(e, err)
	}
	// Serialize the complete review before any explicit output capability runs.
	var preview bytes.Buffer
	if c.format != "raw" {
		if err = plan.WritePreview(ctx, &preview, c.format, c.maxOutputBytes); err != nil {
			return editFailure(e, err)
		}
	}
	if err = w.Apply(plan); err != nil {
		return editFailure(e, err)
	}
	outputName := c.sourceName
	if outputName == "" {
		outputName = w.RootName()
	}
	if _, err = w.Data(outputName); err != nil {
		return invalid(e, "output source is not captured")
	}
	if c.format == "raw" && c.sourceName == "" {
		for _, change := range w.Changes() {
			if change.Source != outputName {
				return invalid(e, "raw root output would omit included-file changes; choose --source or a review format")
			}
		}
	}
	if c.output != "-" {
		for _, change := range w.Changes() {
			if change.Source != outputName {
				return invalid(e, "one-file copy would omit changes to another source; use --in-place or separate edits")
			}
		}
	}
	if c.format == "raw" {
		if err = w.WriteSource(ctx, &preview, outputName); err != nil {
			return editFailure(e, err)
		}
		if preview.Len() > c.maxOutputBytes {
			return editFailure(e, edit.ErrLimit)
		}
	}
	var guards []atomicfile.SourceGuard
	paths := map[string]string{}
	if !c.create {
		paths = editSourcePaths(inputFile, c.input, e, set)
		for _, name := range w.Names() {
			path, ok := paths[name]
			if !ok {
				continue
			}
			b, _ := setSourceData(set, name)
			guards = append(guards, atomicfile.SourceGuard{Path: path, Digest: editSourceDigest(b), Bytes: len(b)})
		}
	}
	if c.inPlace {
		for _, change := range w.Changes() {
			if paths[change.Source] == "" {
				return invalid(e, "in-place output requires named original files; use -o for stdin/new documents")
			}
		}
	}
	if !c.dryRun {
		if c.inPlace && e.CommitEdits == nil {
			return failure(e, "guarded source commit capability unavailable")
		}
		if c.output != "-" && e.WriteEditedOutput == nil {
			return failure(e, "guarded output-copy capability unavailable")
		}
	}
	for _, d := range plan.Diagnostics() {
		if _, err = fmt.Fprintf(e.Err, "edit %s: %s", d.Code, d.Message); err == nil {
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
	for _, d := range w.Sources().Diagnostics() {
		if _, err = fmt.Fprintf(e.Err, "%s:%d:%d: %s: %s\n", d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Message); err != nil {
			return failure(e, err.Error())
		}
	}
	if err = writeEditPreview(e.Out, preview.Bytes()); err != nil {
		return failure(e, err.Error())
	}
	if !c.dryRun {
		if c.output != "-" {
			b, _ := w.Data(outputName)
			if err = e.WriteEditedOutput(ctx, c.output, guards, b); err != nil {
				return failure(e, err.Error())
			}
		}
		if c.inPlace {
			err = w.Commit(ctx, func(ctx context.Context, batch edit.CommitBatch) error {
				edits := make([]atomicfile.SourceEdit, 0, len(batch.Changes))
				for _, change := range batch.Changes {
					edits = append(edits, atomicfile.SourceEdit{Path: paths[change.Source], Digest: change.BeforeDigest, Data: change.After})
				}
				return e.CommitEdits(ctx, edits, guards)
			})
			if err != nil {
				return failure(e, err.Error())
			}
		}
	}
	return ExitOK
}
func editFailure(e Environment, err error) int {
	if errors.Is(err, edit.ErrLimit) || errors.Is(err, query.ErrLimit) || errors.Is(err, source.ErrLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return failure(e, err.Error())
	}
	if errors.Is(err, edit.ErrOptions) || errors.Is(err, query.ErrSelector) {
		return usage(e, err.Error())
	}
	if errors.Is(err, edit.ErrSelection) || errors.Is(err, edit.ErrTarget) || errors.Is(err, edit.ErrStale) || errors.Is(err, edit.ErrOverlap) || errors.Is(err, edit.ErrValidation) || errors.Is(err, edit.ErrNeedsRewrite) || errors.Is(err, source.ErrStrict) {
		return invalid(e, err.Error())
	}
	return failure(e, err.Error())
}
func invalid(e Environment, message string) int { fmt.Fprintln(e.Err, message); return ExitInvalid }
func writeEditPreview(out io.Writer, b []byte) error {
	if out == nil || isNilEditWriter(out) {
		return fmt.Errorf("nil edit preview writer")
	}
	n, err := out.Write(b)
	if err != nil {
		return err
	}
	if n < 0 || n > len(b) {
		return fmt.Errorf("invalid preview writer count")
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}
func setSourceData(set *source.Set, name string) ([]byte, bool) {
	for _, f := range set.Files() {
		if f.Name() == name {
			return []byte(f.AST().Source()), true
		}
	}
	return nil, false
}
func editSourcePaths(file string, input sourceOptions, e Environment, set *source.Set) map[string]string {
	paths := map[string]string{}
	root := input.root
	rooted := e.OpenRoot != nil
	if root == "" && file != "" && file != "-" {
		root = filepath.Dir(file)
	}
	for _, f := range set.Files() {
		if f.ParentEdge() == 0 && (file == "" || file == "-") {
			continue
		}
		p := f.Name()
		if rooted {
			p = filepath.Join(root, filepath.FromSlash(p))
		}
		paths[f.Name()] = p
	}
	return paths
}

func editSourceDigest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func isNilEditWriter(out io.Writer) bool {
	v := reflect.ValueOf(out)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
