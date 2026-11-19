# Bounded source sets

`source.Load(ctx, fsys, name, options)` loads a named document from an explicit
`fs.FS`. `source.Read(ctx, reader, name, options)` parses a borrowed reader and
loads includes only when `options.FS` is supplied. Neither calls external tools,
fetches network resources nor evaluates macros. `groff.ParseAST` continues to
parse one source without any implicit filesystem access.

```sh
./gtroff sources document.1
./gtroff sources --format text --root docs docs/document.1
./gtroff sources --no-includes document.1
./gtroff sources --root docs - < fragment.roff
```

These are real generated commands. `sources` exposes the shared input layer for
the implemented [query command](query.md) and subsequent analyze, lint and extract
commands. The source-inspection command itself does not perform those operations. Existing `ast`, `validate`, rendering and conversion behavior stays
syntax-only with respect to include loading. The legacy opt-in
`ParseDocument` resolver remains supported separately.

## Root and path policy

For the CLI, a named file defaults to its containing directory as the allowed
root. `--root DIR` selects a larger explicit root, and the input file must lie
inside it. Each literal `.so` or `.mso` target resolves relative to the referring
file's directory **within that root**, including `../` that stays inside the
root. This is a bounded source-inspection policy, not GNU groff's working-directory
or macro-search-path emulation. `.mso` has no implicit system macro directories.
Quoted path arguments use the existing argument lexer. Apostrophe control prefixes
are recognized through dot-normalized AST names. Malformed or multiple arguments,
indirect roff escapes, invalid UTF-8 paths, NUL, backslashes, drive/URL spellings
and outside-root paths are refused. No variable, tilde or shell expansion occurs.

The process adapter uses the existing `manlib.OpenDirectory` root capability,
backed by `os.Root.FS()`, to prevent symlink escape. OS file identity also detects
active symlink/hard-link aliases as cycles. Mounts and hard links within a trusted
root are still trusted: select roots containing data you intend to inspect.
Regular files are required after opening. Applications must supply their own
I/O deadlines for blocking filesystem operations or readers; Go context checks
do not interrupt arbitrary blocked host calls.

Library callers can use `fstest.MapFS`, an embedded FS or another explicit
capability. An injected FS is a trust boundary and must enforce its own symlink
containment; plain `os.DirFS` does not provide that guarantee. To select an OS
root explicitly, `source.OpenDirectory(dir)` returns a caller-owned `*os.Root`;
pass `root.FS()` to `Load` and close the root after loading. Every source file
opened by `Load` is closed, including failed/cyclic attempts. `Read` never closes
its borrowed root reader. The [complete Go examples](../source/example_test.go)
show both injected virtual files and unresolved reader input.

Stdin has no source root unless `--root` is explicit. Without one, includes
remain unresolved references and no host path is consulted. `--no-includes`
disables loading while retaining recognized directives, even if a root exists.
An embedded CLI caller without an injected `OpenRoot` capability falls back to
an explicit named stream and reports unresolved includes; an explicit `--root`
without that capability is an error. No process-global filesystem is substituted.
The library uses only built-in parsing handlers by default; `Options.Registry`
can explicitly inject a configured registry. Handler objects remain caller-owned
and must not be reconfigured during a load.

## Three separate representations

- `Set.Files()` returns immutable source occurrences, each with a file-local AST,
  exact source bytes, source name, parent include edge and nesting depth.
- `Set.Edges()` retains each recognized active include's original node ID/span,
  literal argument, child occurrence (if captured) and resolution state.
- `Set.Lines()` provides a logical traversal of physical source: keep an include
  directive, visit its child lines, then continue the referring source.

A repeated include has a separate occurrence ID and consumes budgets again.
There is no content cache that hides a failed attempt or assumes a file stayed
unchanged between reads. Source names are root-relative; AST node IDs and byte
offsets always belong to their original file. Use `set.Node(line)` and
`set.File(line.Source).AST()` to navigate. No synthetic concatenated AST or
relocated byte offsets are invented. A source set is not an executed document;
heading/paragraph interpretation across include boundaries remains a separate
consumer projection. Within each file, the existing optional `man.Build`
hierarchy remains available.

Opaque definition, ignore, preprocessor and parser-owned capture bodies are
not include instructions. Conditional bodies are not executed. Recognition,
loading, semantic interpretation and rendering are distinct facts. All original
syntax remains accessible, including unknown macros, comments, CRLF, malformed
material and invalid UTF-8. Slice accessors return independent slices; pointed-to
files and ASTs are immutable through the public API.

## Errors, states and limits

Edges have deterministic states: `resolved`, `disabled`, `unresolved`, `invalid`,
`missing`, `denied`, `cycle`, `depth-limit`, `resource-limit` or `incomplete`.
`To` identifies a captured child even if that child encountered an input/resource
failure. A resolved edge proves only that its file was read; descendants may
still contain missing references or syntax diagnostics. Broken references never
erase their referring node. A fatal input/resource/cancellation error stops any
further include opens, while retaining the captured syntax and line references.

Permissive mode reports syntax and include diagnostics. Strict mode returns a
partial useful set plus `ErrStrict` for **any** diagnostic, including deliberately
disabled/unresolved includes. `IncludeError` carries the original edge and unwraps
underlying errors such as `fs.ErrNotExist`. Root-open, reader, close, resource and
cancellation failures always return errors. `ErrLimit` identifies set byte/file/
depth exhaustion; parser-specific node/line bounds retain their existing errors.
Check both the returned error and each AST's `SourceComplete()`; neither a graph
nor a strict flag guarantees an observed EOF or semantic support.

Defaults across all occurrences are 8 MiB captured bytes, 100,000 AST nodes,
256 sources and 16 include levels (depth is capped at 128). Repeated and failed
reads count toward the byte budget. The parser can retain one extra detection
byte and one recovery node; no later file is opened after exhaustion. Existing
per-AST line-length/depth protections still apply. Cancellation is checked around
reads, before opens and between physical lines; bounded AST CPU parsing checks
it before and after construction. User-supplied blocked readers, FS methods and
custom handlers remain responsible for their own deadlines.

CLI limits must be positive: at most 64 MiB, 1,000,000 nodes (minimum 6), 4,096
source occurrences and 128 levels. The library accepts explicit budgets without
those CLI ceilings except for the depth safety cap. `--max-bytes`, `--max-nodes`,
`--max-files` and `--max-depth` select the cumulative budgets.

## Reports and verification

JSON reports use outer schema version 1 and a `sourceSet` with its own version 1,
containing original per-file source/ASTs, edges, logical line references and
source-attributed diagnostics. JSON replaces invalid UTF-8 according to Go's
`encoding/json`; it is not an archive or a deserializable source snapshot. Use
`AST.Source()` for exact original bytes. Text reports summarize files/edges and
diagnostics with original `file:line:column` locations. Both modes write stdout.
CLI exit codes are 0 for permissive reports, 1 for strict diagnostics, 2 for usage
and 3 for I/O/resource/cancellation failure; partial JSON includes an error.
Diagnostics sort by source occurrence, byte offset and stable insertion order.

Readable `source/testdata/*.txtar` scenarios assert graph/line order and physical
source reconstruction. `testdata/cli/sources-*.txtar` covers named-file defaults,
`--no-includes`, stdin/root, broken/cyclic/escaping includes, strict output and
limits. Unit tests separately check symlink aliases/escape, shared failed-read
budgets, cancellation, reader and close errors, exact bytes, immutable slices
and explicit registries. A subprocess test builds and executes the generated CLI
for named/default, disabled and stdin/root policies and strict exit behavior.

`gtroff extract` uses this shared source policy. Records retain original-file
provenance and include relationships; exact parent source is not expanded into
child bytes. Explicit file output protects every loaded original source. See
[extraction](extract.md).
