# Headless CLI

Build `go build -o gtroff ./cmd/gtroff` or install with
`go install github.com/arran4/golang-gtroff/cmd/gtroff@latest`. The new command has
no groff, browser, font, terminal, GUI or CGO requirement. The existing `cmd/lex`
command is preserved as a compatibility token-dump utility.

```sh
./gtroff help
./gtroff version
./gtroff validate --strict testdata/simple.1
./gtroff validate --format json testdata/simple.1
./gtroff ast --select section:OPTIONS --format text testdata/simple.1
./gtroff ast --select section:OPTIONS testdata/simple.1 > options.json
printf '.B fragment\n' | ./gtroff validate --format json -
```

Flags precede FILE. FILE is optional and defaults to stdin; `-` explicitly means
stdin. Use `--` before a filename that begins with a dash. Exactly one input is
accepted. Named input is opened explicitly and closed before output; stdin,
stdout and stderr remain owned by the caller. No include targets or other files
are opened from document content, and no macro or external command is executed.

The package defaults to `man`; `--package ms` and `--package roff` are also
accepted. `validate` defaults to text and `ast` to JSON; both accept
`--format text|json`. Help is available as `help COMMAND` or `COMMAND --help`.
Version provenance defaults to `dev`, `none`, `unknown`, and release builds can
supply `main.version`, `main.commit`, `main.date` through `-ldflags -X`.

## Validation and exit status

Validation is syntax inspection of the current parser's supported grammar. It
does not check the complete man-page semantic model, execute preprocessor
languages, or establish that GNU groff will typeset a page successfully. A
fragment or empty input can be valid syntax. An include directive produces an
explicit warning that its target was not loaded. Opaque bodies and unsupported
features remain source; they do not become evaluated content.

| Exit | Meaning |
| --- | --- |
| 0 | Syntax accepted, possibly with warnings in permissive mode |
| 1 | Syntax diagnostics contain an error, or strict mode rejects warnings |
| 2 | Invalid command/flags/input count/selector, or selection matches nothing |
| 3 | Input open/read/close failure, resource exhaustion, or output failure |

Malformed syntax fails in permissive mode as well as strict mode. Unknown
controls produce warnings and partial support in permissive mode; `--strict`
rejects all warnings, including unknown controls and unloaded includes. The
original severity remains visible. These statuses express this tool's parser
and policy decisions, not a judgment about all valid GNU roff programs.

JSON reports `assessment: "syntax-only"` and `status` as `valid`, `partial`,
`invalid` or `incomplete`. The last indicates an input/resource failure, with an
`error` field. A preserved AST is still available for malformed or partially read
input. Diagnostics always refer to the whole input, even for a selected subtree.

## Output and selection

`validate --format json` writes a versioned result containing source name,
assessment, status and ordered diagnostics. `ast --format json` adds the real
AST's selected node table, root IDs and the complete original source snapshot.
Both use `schemaVersion: 1`. Nodes retain original IDs, parent/child links, paths,
kinds, spans, raw source, attributes, generated token fields and inline fields.
Source offsets index original bytes; JSON uses UTF-8. Invalid UTF-8 is diagnosed,
and `sourceBase64`/`rawBase64` fields preserve those bytes alongside JSON's
replacement-character text representation.

Selectors are exact and case-sensitive:

- `node:42`: one source node and its descendants.
- `path:/0/1`: the node at a zero-based child path; `path:/` selects the root.
- `section:OPTIONS`: all matching sections/subsections, including repeated names.

Overlapping selected subtrees produce each node row once in document order.
Original parent IDs outside the selection are listed in `externalParentIds`;
the selected roots remain listed in `rootIds`. Empty selection is an error,
never an implicit fallback to the full document.

Text AST output is a preorder row per node with ID, path, kind, control/heading,
source position, half-open byte range and quoted raw source. Text validation
writes a one-line assessment to stdout. Text diagnostics go to stderr. JSON
syntax diagnostics stay in the JSON envelope; usage/open/export errors that
prevent an envelope go to stderr. Stdout contains no progress messages.

The `validate` and `ast` commands are stdout-only. The separate HTML `render`
command supports atomic `-o FILE`; see [HTML contracts](html.md). The `image generate`
subcommand provides a discoverable interface for exporting paginated raster images,
reusing the existing PNG/JPEG raster engine. The `view FILE` command launches the
optional `gtroff-view` Shiny viewer when available. The `commands --format json`
command emits a versioned, machine-readable inventory of commands, flags,
descriptions, and capabilities for agents and integrations. Shell
redirection is owned by the caller and has no atomicity promise. JSON/text export is buffered with a 32 MiB limit; an encoding
or export-budget failure leaves stdout empty. An underlying stdout write error
can leave a partial stream and exits 3. Input uses the AST's documented source,
line, node and depth bounds. Callers must check the exit code before consuming
an export as validated input.

## Implementation and verification

Command dispatch is generated at build time using `go-subcommand` (`gosubc` v0.0.32),
pinned in `internal/tools/tools.go` and executed via `go generate ./...`.
`cmd/gtroff/main.go` wires process arguments, build values, streams and an
explicit file opener into the generated command root, while preserving the
injected `cli.Run` compatibility API for embedded test scenarios. Tests inject
readers/writers and an in-memory filesystem rather than replacing process
globals. A separate built-executable test verifies real files, stdin, output,
exit codes, build metadata, selections, raster rendering and no includes.
There are no GUI dependencies in the headless `gtroff` binary.

The `md2png` generated `go-subcommand` main was inspected for command/version and
exit plumbing. This repository has no existing command generator. A small
standard-library dispatcher avoids a command-generator dependency; no generated
main was edited or copied. The module now requires Go 1.26 for patched HTML/font
dependencies. HTML render is documented separately; image/view/server commands
remain subsequent implementations.

Embedded `testdata/cli/*.txtar` scenarios contain `args.json`, optional
`stdin.1` or `input/NAME`, and exact `stdout.txt`, `stderr.txt`, `exit.txt`
expectations. CRLF is stored as actual bytes. JSON is actual command output,
parsed and cross-checked against a fresh source AST. Normal tests do not rewrite
expectations. Refresh explicitly, then review the diff:

```sh
go test . -run '^TestCLIScenarios$' -update-cli
go test ./...
go test ./internal/cli -run '^$' -fuzz '^FuzzCLIInspection$' -fuzztime=10s
```

This phase depends on AST PR #32, itself stacked on unmerged parser PR #19 at
`75d60db`. Parser issues #3–#17 remain separate contracts; see
[the AST dependency notes](ast.md#parser-dependencies). No includes or evaluator
capabilities are exposed. The semantic man model in #22 is the next phase.

## Bounded source inspection

`gtroff sources [--no-includes] [--root DIR] [FILE]` inspects safely bounded local
includes and original-file provenance, with JSON or text output. Stdin needs an
explicit root. See the [source-set guide](sources.md) for path policy, resource
limits, strict diagnostics, source exactness and exit codes. `gtroff ast` remains
a single-file syntax inspection command.

## Structural queries

`gtroff query [OPTIONS] SELECTOR [FILE]` uses CSS-like structural selectors or
equivalent typed Go builders over syntax or an optional man hierarchy. It shares
the bounded source loader and `--no-includes` policy, and exposes JSON/JSONL, text,
raw source, grep and CSV/TSV records. [Grammar, source attribution and limits](query.md).

## Source editing

`gtroff edit` previews source-backed in-memory transformations. It shares
query selectors and bounded local source loading, supports one-based match
restrictions and explicitly owned include edits, and requires `-o` or `--in-place`
for filesystem writes. `--dry-run` prevents those writes. See the
[operation, source fidelity and recovery contracts](edit.md).

## Optional Tengo execution

The generated `script` command accepts explicit SCRIPT and optional DOCUMENT
inputs. The core binary is engine-free; build the self-contained optional variant
from `script/cli` to execute programs. Read-only queries default on, while memory
edits and guarded filesystem writes require separate grants.
[Script syntax, build, source policy and output contracts](script.md).
