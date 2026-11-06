# Tengo document programs

The optional `script/` module embeds the approved Tengo engine in pure Go. It
shares the existing stateless selectors, source sets, extraction records and
source-backed editor. It does not evaluate roff macros or install a plugin runtime.
The default core module remains independent of Tengo.

```sh
CGO_ENABLED=0 go -C script build -o ../gtroff ./cli
./gtroff script script/examples/inspect.tengo script/examples/environment.1
./gtroff script --allow-edits --in-place --dry-run script/examples/transform.tengo script/examples/environment.1
./gtroff script --allow-edits --write-target new.1 script/examples/create.tengo
```

The optional binary is self-contained; it requires no external script process.
It uses the shared CLI's documented flags-before-input convention. Its ordinary
query/extract/edit/render commands remain available. A core build exposes generated
`script` help/inventory but returns an explanatory backend-unavailable error
without opening inputs. These are unreleased development APIs/builds.

`SCRIPT [DOCUMENT]` names explicit inputs; at most one may be `-` for stdin.
Omitting DOCUMENT gives no current document, rather than silently consuming stdin.
Named documents share bounded local include resolution, `--root` and
`--no-includes`. Script modules/file imports stay disabled even when a document
root is authorized. Loaded includes are references, not macro execution.

## Native language, stateless document operations

Tengo owns program/session variables, functions, collections and control flow.
Selectors remain stateless. The host object `doc` exposes:

| Method | Contract |
| --- | --- |
| `query(selector[,view])` | Immutable ordered match array; default semantic view, optional `"syntax"` |
| `extract(selector,format)` | Existing extraction report/export as bytes; formats json/jsonl/text/raw/grep/csv/tsv/html |
| `source([name])` | Exact bytes of one current physical file; default root, never include expansion |
| `sources()` | Sorted captured physical names, including retained dirty files |
| `quote(text)` | Quote one string for a selector literal using existing escaping rules |
| `edit(command)` | Apply an existing typed edit command shape in memory, only with an edit grant |
| `create(name[,title,section])` | Create a virtual man document, only with an edit grant; refuses to discard pending/new content |
| `preview(format)` | Review pending changes against the working baseline as `"diff"` or `"json"` bytes |
| `emit(stringOrBytes)` | Stage exact output bytes; adds no newline |
| `emit_json(data)` | Stage one deterministic JSON record and newline, with bounded cycle-aware data conversion |
| `write([source,]destination)` | Stage one explicit physical-file copy intent, only with a request grant; no filesystem access |

Matches expose `index`, `kind`, `name`, `term`, `text`, `raw`, `source`, `digest`,
`line`, `column`, `offset` and `end`. `raw` is a byte value; text is the selected
syntax/semantic representation. Source positions refer to the original roff file.
Tengo compiler/runtime locations refer separately to SCRIPT. Full structured
extraction records retain richer heading/context/diagnostic information and
separate term/description fields; use `extract` for those records.

```tengo
selected := doc.query("section[name=ENVIRONMENT] definition")
each := func(values, fn) { for value in values { fn(value) } }
each(selected, func(match) {
    if match.term == "TOOL_HOME" {
        doc.emit_json({term: match.term, file: match.source, line: match.line})
    }
})
doc.emit(doc.extract("definition[term=TOOL_HOME]", "jsonl"))
```

Host query arrays and JSON encoding use deterministic order. Native Tengo map
iteration order remains unspecified; do not use it as an ordered record stream.
Timing statistics are informational, not deterministic content. No stdlib/module,
network, shell, environment or arbitrary file-import capability is installed.

## In-memory transformations

`--allow-edits` grants memory changes; `--include-edits` additionally grants edits
to clearly selected captured child sources. It grants no filesystem writes.
Commands use the editor's `selector`, `operation`, `view`, `text`, `find`,
`heading`, `term`, `destination`, `rewrite` and `restrict` fields. Native field
types are checked directly; there is no JSON round-trip that could normalize
exact-source strings or accidentally turn byte values into base64 text.

```tengo
for match in doc.query("definition") {
    if match.term == "TOOL_HOME" {
        doc.edit({selector: "definition[term=" + doc.quote(match.term) + "]",
                  operation: "replace-description", text: "New directory."})
    }
}
doc.edit({selector: "source[source-id=1]", operation: "append-section",
          heading: "NOTES", text: "Added by a document program."})
doc.emit(doc.preview("diff"))
```

See [editor operations/policies](edit.md). Non-strict selectors broadcast to all
matches and retain mass-edit/repeated-source diagnostics. Strict requires one
match after restriction. `restrict` accepts one-based positive `nth`, inclusive
`from`/`to`, `firstN`, or boolean `all`; an explicit zero is invalid. Query arrays
are captured values, not live cursors. Requery after edits; old ordinals never
magically retarget an object. Every command evaluates its selector in the current
candidate snapshot. Conflicts, stale plans and invalid owners remain errors.

Raw operations use exact string bytes, including literal escaped bytes. Named
append/description operations accept escaped literal UTF-8 and are explicitly
canonical. A byte-valued argument where a string is required is rejected; use
native `string(bytes)` deliberately when source bytes are intended. New malformed
source fails rather than being normalized into valid replacement characters.
Existing malformed/unknown/opaque/comment/CRLF/invalid bytes can remain untouched.
Complex descriptions require explicit rewrite. Reparse uses captured files only.

## Staged output and explicit files

A program gets a private working snapshot. Until the complete program succeeds,
edits, emissions and write intents remain private. Compilation/runtime/capability/
resource failure returns no partially changed working object, output or requests.
On success, memory changes are available to the Go host; files remain untouched.

CLI `--write-target FILE` authorizes exactly that destination; the program must
also call `doc.write(FILE)` (or `doc.write(source, FILE)`). Merely granting the
capability does not write. A mismatched/duplicate destination is an error.
CLI `--in-place` explicitly commits changed named originals after successful
execution and output. It requires `--allow-edits`; new virtual documents and a
modified stdin root need a copy destination. These two output modes are exclusive.
`--dry-run` suppresses filesystem callbacks for either mode.

```tengo
doc.create("new.1", "NEW", "1")
doc.edit({selector: "source[source-id=1]", operation: "append-section",
          heading: "NAME", text: "new - demonstration"})
doc.write("new.1")
```

The copy intent captures the current physical bytes. Later edits/create make
pending intents stale; they cannot silently export older content. Root filenames,
byte ownership and original-source hashes remain separate. The virtual initial
header is an in-memory baseline; `source()` exports the complete new document.

The CLI verifies all captured document and named-script input hashes/aliases and uses the existing
staging/sync/rename/rollback engine. Input aliases, symlinks, stale disk data and
invalid targets fail. Each replacement is atomic when the host supports it; the
group is not crash-atomic and exclusive writer access is required. Failed restores
retain recovery paths. Stream serialization must succeed before filesystem writes.
A filesystem commit error can follow successful program execution; inspect the
reported recovery state. [Guarded writing/retry details](edit.md) apply.

The library `Run` never acquires a file writer or executes these intents. A Go
host must authorize paths and enforce its storage policy separately. Its request
contains explicit captured Sources and optional loading error. Handle that error
before execution. `Working.WriteSource` and `Working.Commit` provide the existing
explicit writer/guard boundary after success. Complete
[external-package examples](../script/example_test.go) exercise the API.

## Bounds, cancellation and WebAssembly

Defaults are 64 KiB script source, 20000 tokens, syntactic nesting 128, 4096
constant objects, 100000 statement/block checkpoints, 100000 VM object-accounting
units, 1000 host calls, 16 MiB staged output/requests, and a one-second compile/
execution deadline. CLI source loading defaults to 8 MiB/100000 nodes/256
occurrences/depth 16. Query/extraction/edit calls retain their independent bounded
work/resource policies. Go Options can adjust documented limits within ceilings.

The adapter parses native Tengo, inserts private calls before statement-list
entries and in empty blocks, then uses the native compiler/VM. This bounds empty
loops, recursive function bodies and callbacks without rewriting jump offsets or
inventing a language. `__gtroff_` identifiers are reserved for these private calls.
Counters are source-level checkpoints, not exact VM instructions. The deadline
clock is also checked synchronously and Go tasks yield cooperatively: a bare VM's
empty infinite loop otherwise starved timers on Go/js during actual verification.
Original user source locations remain in runtime errors.

This is not an OS sandbox or a hard heap/time quota. Stock Tengo object counting
does not charge every allocated byte; a controlled test creates a 1 MiB byte value
within ten accounting units. String/byte value maxima are process-global in the
engine; the library does not mutate them. A large native operation, recursion
through cyclic native values or a blocking host callback is not made safe merely
by a surrounding deadline. Do not expose this same-process interpreter as a
hostile-code security boundary. Capability denial prevents the exposed APIs from
performing unrestricted filesystem/network/shell/import operations.

Normal JSON serialization bounds depth/nodes/bytes and rejects cycles/functions.
JSON follows standard UTF-8 replacement for strings; byte values encode as base64.
Use byte emission/source output for exact original bytes. Runtime/timing stats
remain separate from emitted records. Exit codes are 0 success, 1 script/selection/
capability errors, 2 CLI usage, 3 input/output/resource/cancellation failure.

Persistent native variables/functions, transactional cells, original document
guards and explicit publication/retry are available through `NewSession` and
`script --repl`. [Session API, stream controls and limits](script-repl.md).
`Run` remains an independent one-program API; feeding it the last Sources does
not preserve a session or the original write baseline.

Pinned engine documentation and its MIT notice are linked in the
[optional module guide](../script/README.md). Root licensing/provenance/publication
gates remain unchanged.
