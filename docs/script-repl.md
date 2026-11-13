# Persistent Tengo sessions

The optional Tengo executable supports persistent native programs on stdin:

```sh
CGO_ENABLED=0 go -C script build -o ../gtroff ./cli
./gtroff script --repl script/examples/environment.1
./gtroff script --repl --prompt=false script/examples/environment.1 < script/examples/session.tengo
```

The generated core command exposes these flags/help without importing Tengo. A
core build reports the missing backend before opening inputs. The optional build
uses the same source loader, query/extraction/editor APIs and guarded output as
one-program mode. No external interpreter, shell, terminal library or agent
service is needed. `--repl` accepts one optional **named** document; stdin belongs
to cells. Named documents use bounded includes under their directory or explicit
`--root`; `--no-includes` retains references without loading them.

## Cells and streams

Native Tengo variables, functions, closure cells and the working document persist.
Use assignment (`count = 2`) for an existing variable; native declaration rules
still apply. Selectors remain stateless. A saved query array is captured data:
its indices, source digests and spans refer to that snapshot. Editing later does
not turn the array into live nodes or persistent cursors. Requery to select the
current document.

```tengo
count := 0
next := func() {
    count++
    return count
}
doc.emit_json([next(), len(doc.query("definition"))])
```

The native parser detects incomplete expressions, functions, brackets, raw
strings and block comments. Other syntax errors are reported as failed cells;
input then resumes at the next cell. An incomplete cell at EOF fails without
executing its prefix. Empty lines outside a cell are ignored. CRLF and raw input
bytes are not normalized by the stream reader. As in native Tengo, an expression
followed by a newline may already form a complete statement; put continuation
syntax on the appropriate line.

Prompts (`gtroff> ` and `...> `), `:help` and errors use stderr. Disable prompts
with `--prompt=false` for scripts and agent transcripts. Stdout contains explicit
`doc.emit`/`emit_json` output, or a requested `:pending` byte-diff review; ordinary
expressions are not automatically printed. Text emissions need not end in a
newline. A borrowed blocking input/output stream owns its I/O cancellation;
context checks do not make an arbitrary blocked Reader/Writer interruptible.

| Control | Effect |
| --- | --- |
| `:help` | Show controls on stderr |
| `:cancel` | Discard incomplete multiline input |
| `:pending` | Review the pending document diff after publication failure |
| `:retry` | Retry exactly the pending output/write batch |
| `:discard` | Accept memory state, discarding pending emissions/write intents |
| `:commit` | Explicit original-file commit; requires `--in-place` |
| `:quit` | Exit without additional writing |

Controls apply at cell boundaries; `:cancel` is additionally reserved on a line
by itself during multiline input. These controls manage the session and its I/O;
all document programs use the native Tengo grammar. `:pending` is a quoted
byte-range review, not a patch(1) program. To review normally, call
`doc.emit(doc.preview("diff"))` in a cell.

Exit status is the highest encountered class: 0 success, 1 script/selection/
capability failure, 2 invalid invocation, 3 I/O/resource/cancellation failure.
A recovered error remains reflected in the final status. Explicit quit and EOF
do not commit memory changes or silently retry a failed write.

## Cell transactions and explicit publication

Before running a cell, the engine forks the working editor **with its original
baseline**, reconstructs isolated compiler bindings and copies the retained
native object graph. Cycles, shared objects/maps, overlapping array/byte slices,
immutable wrappers and closure cells preserve their relationships. It uses a
bounded iterative graph walk rather than native recursive Object.Copy. Native
copy operations *inside a program* retain Tengo's existing behavior and limits.
Host functions always refer to this session's explicit document capabilities.
Compiled instruction/source maps are immutable; constant indices stay stable
across cells instead of being deduplicated/reindexed beneath retained functions.

A compilation/runtime/resource failure publishes no new globals, document edits,
emissions or intents. A successful cell publishes its in-memory state and then
holds output/intents pending until the host accepts them. Publication happens
after success; stdout or a filesystem callback can still fail. In that case the
same pending result remains inspectable, and further cells are refused until
`:retry` or `:discard`. Do not rerun an append/edit cell to recover its output.
Retrying a stream can repeat a prefix already written. Atomic filesystem recovery
and retained backup paths come from the existing guarded writer; diagnostics name
those paths when restoration fails. This does not make a group crash-atomic.

`--allow-edits` permits memory mutation. `--include-edits` additionally permits
selected included owners. Strict unique-match and permissive broadcast/restriction
policies are the editor's existing policies. Untouched bytes, comments, unknown
requests and malformed captured material remain exact; intentionally canonical
operations retain their explicit rewrite requirements.

`--write-target FILE` authorizes only a matching explicit `doc.write(FILE)` intent
(or `doc.write(source, FILE)` for another captured owner). It protects original
source hashes and aliases. Copy output never acknowledges an original-file edit.
In REPL mode `--in-place` authorizes **`:commit`**; ordinary cells do not commit
merely because the flag was supplied. Each successful original commit adopts the
new baseline/guards; later edits compare against it. Queries, ordinary cells and
failed commits keep the previous guards. `--dry-run` suppresses filesystem calls
and leaves the dirty baseline intact. New/stored-stdin documents need a copy
destination; REPL documents themselves cannot read stdin.

## Go session API

Import `github.com/arran4/golang-gtroff/script` from the optional module:

```go
session, err := script.NewSession(ctx, script.Request{Sources: captured},
    script.SessionOptions{Cell: script.Options{AllowEdits: true}})
if err != nil { return err }
result, err := session.Eval(ctx, script.Cell{Name: "inspect", Code: program})
if err != nil { return err }
// Inspect result first. Publish uses a fresh isolated Result on each attempt.
err = session.Publish(ctx, func(ctx context.Context, result *script.Result) error {
    // Use an explicitly authorized stream/filesystem callback here.
    return writeToAuthorizedStream(result.Output)
})
```

`Pending(ctx)` returns an isolated copy. A failed publisher retains the original
pending result; retry Publish, rather than Eval. Use an explicit no-op publisher
to accept memory state while discarding staged output. If the publisher commits
`result.Working` through the editor's explicit Commit callback and then succeeds,
the session adopts that committed baseline. Publisher failure does not acknowledge
its fork even if it was mutated; the host owns recovery from external side
effects. Publishers are trusted capabilities and must not reenter the session.
Methods require exclusive caller access. Sessions own only memory and need no
Close operation. Construction context cancellation prevents future Eval/Ready;
explicit Pending/Publish calls use their supplied contexts for inspection/recovery.

`Ready(ctx, cell)` probes native syntax without executing or defining variables.
`NewSession` accepts no Request.Code; InputError must be clear before execution.
The root `scriptapi` package provides engine-free Session/SessionFactory ports for
embedded CLIs. External Go examples demonstrate persistence and publication;
`edit.Working.Fork`/`Guards` expose the baseline-preserving building blocks.

## Resource contracts

Per-cell bounds are inherited from [script options](script.md): script bytes,
tokens, nesting, constants, checkpoints, allocation accounting, host calls,
output and duration. Session defaults add:

| Budget | Default | Maximum |
| --- | ---: | ---: |
| Attempted cells | 1,000 | 10,000 |
| Program bytes | 4 MiB | 64 MiB |
| Checkpoints | 1,000,000 | 100,000,000 |
| Host calls | 10,000 | 1,000,000 |
| Emission plus write-intent bytes | 32 MiB | 128 MiB |
| Retained state accounting objects/edges | 100,000 | 1,000,000 |
| Retained state accounting bytes | 16 MiB | 64 MiB |

CLI `--max-cells` and `--max-session-bytes` configure the first two. CLI input
bytes also count controls and blank lines. Repeated multiline probes have a
separate work cap of 16 times the session program-byte limit. Failed attempted
cells consume budgets. Publication retries do not re-execute or replenish them.
Retained constants count cumulatively; defaults allow 4,096, with a session ceiling
of 65,535 due to the pinned native constant operands. Native globals are capped at
1,024 slots, including compiler-private slots retained for old functions.

The pinned engine's one-byte call/local/closure operands require additional
shape limits: at most 255 explicit call arguments/parameters, 256 local slots and
255 distinct identifiers across each function and its nested functions. That
identifier rule conservatively bounds forwarded closure captures. Unsupported
shapes fail before execution in both Run and sessions. Source checkpoint
instrumentation, synchronous deadline checks and cooperative scheduling remain
active for functions retained from earlier cells, including empty loops on Go/js.

State accounting includes container capacity and references conservatively; it
can reject a graph below an actual heap-byte estimate. It validates candidates
before publication and never traverses cycles recursively. These controls are not
an OS sandbox or a hard process heap/time quota. Large native operations,
recursive built-ins over cyclic values and blocking callbacks retain the limits
documented for one-program execution. The runtime does not change process-global
Tengo settings.

Fifteen small TXTAR transcripts capture real CLI stdout/stderr using Go quoted
bytes (preserving missing final newlines), with explicit `-update-repl` refresh.
Native executable and Go/js tests separately cover graph rollback, guards,
publication retry and loop cancellation. Cross-compilation alone is not runtime
execution evidence.
