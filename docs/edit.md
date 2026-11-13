# In-memory source editing

The experimental `edit` package and `gtroff edit` operate on captured roff sources.
They reuse the source loader, stateless query IR, man model and canonical man
writer. They neither execute roff nor serialize a whole AST back into source.
The original snapshot, mutable working bytes and optional canonical fragments are
separate representations.

```sh
# Preview a physical description change; no files are written.
gtroff edit --select 'section[name="ENVIRONMENT"] definition[term="TOOL_HOME"]' --operation replace-description --text 'New directory.' tool.1
# Explicit copy; every loaded original is hash-checked and protected from aliases.
gtroff edit --select 'definition[term="TOOL_HOME"]' --operation replace-description --text 'New directory.' -o updated.1 tool.1
# First inspect an include edit, then deliberately commit it.
gtroff edit --include-edits --in-place --dry-run --select 'definition[term="TOOL_HOME"]' --operation replace-description --text 'New directory.' tool.1
# Create a new document, explicitly writing one destination.
gtroff edit --create --title TOOL --section 1 --operation append-section --heading NAME --text 'tool - demonstration' -o new.1
```

Flags precede the optional input. Named inputs resolve local includes beneath
`--root` (default: input directory); `--no-includes` retains directives without
loading them. Stdin has no external source context unless a root is explicit.
`--create [NAME]` reads nothing and starts an empty document, optionally with a
canonical `.TH` from `--title` and `--section`. Its default source name is
`untitled.1`. Regular functionality remains pure Go.

## Selectors, commands and plans

CSS-like selectors and typed Go selectors locate syntax or semantic objects.
They carry no editing/session state. Line-addressed work can use the query's
`line`, `offset` and source predicates or the Go exact-span API. Structural
commands remain readable when files contain repeated sections and includes.
Tengo can own procedural state and call this same transformation API separately.
There is no new ed interpreter, loop language or command evaluation.

The default selector is `source[source-id=1]`, selecting the root occurrence only.
`--view syntax` selects physical syntax rather than the default man projection.
Each command queries the current candidate. A command file executes sequentially,
so later selectors see earlier edits. The complete batch succeeds or leaves the
live working document unchanged.

| Operation | Content and boundary |
| --- | --- |
| `replace-source` | Replace the exact selected physical byte interval with `--text` |
| `find-replace-source` | Replace all nonoverlapping literal `--find` byte occurrences inside each selected interval |
| `delete` | Delete the selected physical interval |
| `insert-before`, `insert-after` | Insert exact `--text` at the selected interval boundary |
| `move-before`, `move-after` | Move selected exact source to a unique `--destination` selector boundary; may cross explicitly allowed included sources |
| `append-paragraph` | Append an escaped canonical paragraph from literal `--text` |
| `append-section` | Append a canonical section from literal `--heading`, with optional paragraph `--text` |
| `append-definition` | Append a canonical definition from literal `--term` and description `--text` |
| `replace-description` | Replace a semantic definition's body with escaped literal `--text`, preserving its request/tag and exterior comments |

Append operations accept a source, document, section or subsection. They append
at its physical end. New sections enter the file's section flow; they do not
create an illegal nested `.SH`. A `.PP` closes a preceding definition list;
appending a paragraph to a definition itself is rejected. Insertions into a
captured/unclosed block are rejected for named append operations. Exact source
operations deliberately allow editing opaque material and are validated again.

Source operations are byte operations, including roff escapes and controls, not
glyph substitutions in interpreted text. Named operations accept literal UTF-8,
reject control characters, and reuse existing escaping/writer rules. Canonical
flowing paragraphs may fold whitespace or add font resets. Added fragments use
the nearest physical newline convention (LF/CRLF); existing bytes are untouched.
A missing final newline receives a separator when a named fragment is appended.

Simple plain description bodies can be replaced without reconstructing tags or
surrounding comments. Styled/escaped bodies, internal comments, nested structures
and unsupported material require explicit `--rewrite`; otherwise
`ErrNeedsRewrite` is returned. With that flag the bounded body interval becomes
canonical literal content, a diagnostic describes the rewrite and the full removed
bytes remain in the review. A missing `.TP` tag is an invalid target.

## Multiple matches and ownership

Query indices are one-based, ordered and scoped to one snapshot. Non-strict edits
select all matches by default and report broadcast diagnostics for multiple
logical matches. Restrict with one of `--nth N`, inclusive `--range A:B`,
`--first N` or `--all`. A first-N bound exceeding the result count selects the
available prefix. Missing selectors/find text yield non-strict warnings/no-ops;
out-of-range Nth/range selections are errors. CLI explicit zero is invalid.
Go `Restriction` uses zero to mean unset, including in JSON command files.

Strict mode requires exactly one result after restriction; even an explicit
multi-result `--all` or range fails. It also rejects existing/new parser/include
diagnostics. Invalid owners, conflicts, stale versions and invalid write targets
remain errors in every mode.

Every match retains original source ownership. Included-file modifications require
`--include-edits` / `Options.IncludeEdits`, including move destinations. A mixed
root/include selection fails together rather than skipping disallowed targets.
Repeated include occurrences refer to the same physical name: identical patches
from the same physical node apply once with a diagnostic. Distinct conflicting or
parent/child intervals fail with `ErrOverlap`. Separate source names that alias a
host file are rejected at the explicit write boundary.

## Go API and exact patches

[`edit/example_test.go`](../edit/example_test.go) contains complete executable
examples for planning, exact patches, creating content and committing.

```go
// set was already loaded with source.Read/Load; handle its input error first.
working, err := edit.New(set, edit.Options{})
if err != nil { return err }
selector := query.Select(query.Definition).Where(query.TermText, query.Equal, "TOOL_HOME")
plan, err := working.Plan(ctx, edit.Command{
    Select: selector, Operation: edit.ReplaceDescription, Text: "New directory.",
})
if err != nil { return err }
if err := plan.WritePreview(ctx, reviewWriter, "diff", 32<<20); err != nil { return err }
if err := working.Apply(plan); err != nil { return err } // memory only
return working.WriteSource(ctx, explicitWriter, working.RootName())
```

`New` requires complete captured EOF snapshots and refuses recovery-budget tails.
The host must handle loading errors separately; a partial/read-failed snapshot is
never an overwrite baseline. Non-strict syntax errors can remain unchanged,
including malformed original lines. Reparse rejects newly introduced error
diagnostics and new unresolved includes. Existing error locations are mapped over
exact patches; changing their source requires fixing them. Strict rejects all
diagnostics. No fresh disk reads occur in planning: reparse can access only the
originally captured files. Missing-reference diagnostics may reflect that limited
captured resolver rather than an original host resolver's message.

Pass the original source package/registry/limits in `Options.Source` for reparsing.
Its FS is ignored and released; its Strict field is ignored in favor of
`Options.Strict`. An initial explicitly disabled include policy is retained.
Captured files remain available even after a directive stops referring to them,
so dirty files and their original hashes are not lost.

`PlanPatches` accepts exact source name, mandatory whole-source SHA-256, start/end
byte offsets and replacement bytes. Plans privately retain owner, version and
candidate snapshots. `Apply` checks the live owner/version/source bytes and swaps
all in-memory state together. Accessors copy mutable slices. Even a no-op Apply
advances version; indices and old plans cannot be replayed after an edit or
successful Commit. Working documents require externally serialized mutation.

`Query` and `Sources` expose the current immutable snapshot; use `extract.Select`
with `working.Sources()` for exports. Stored query indices never retarget future
edits automatically. `Changes` compares the last captured/committed baseline to
current exact bytes. `Commit` invokes an explicit trusted callback with copied
changes and guards for **all** original sources. Failure retains pending memory
changes for inspection/retry; success acknowledges the new baseline. A callback
must enforce its own disk hash, alias, atomicity and recovery policy.

## Declarative command files

`--commands FILE` is exclusive with single-command selector/operation flags. It
accepts at most 64 KiB, schema version 1 and 256 commands. Unknown fields, trailing
JSON and invalid selectors are rejected before source loading. No file content
becomes executable code.

```json
{"schemaVersion":1,"commands":[
  {"selector":"source[source-id=1]","operation":"append-section","heading":"ENVIRONMENT"},
  {"selector":"section[name=ENVIRONMENT]","operation":"append-definition","term":"TOOL_HOME","text":"Directory for tool data."},
  {"selector":"definition[term=TOOL_HOME]","operation":"replace-description","text":"Updated directory."}
]}
```

Each entry supports `view`, `restrict` (`nth`, `from`/`to`, `firstN`, `all`),
`text`, `find`, `heading`, `term`, `destination`, and explicit `rewrite`. JSON
text contains Unicode strings; Go exact patches can contain arbitrary bytes.

## Output, writes and limits

Default `--format diff` emits a deterministic quoted byte-interval review with
before/after file hashes. It is not a unified patch program. `--format json`
contains schemaVersion 1, working version, exact baseline/final bytes, ordered
patches, edit and source diagnostics. Bytes encode as base64; invalid UTF-8 remains
recoverable. Patches identify a one-based command `step`. In a sequential plan they refer to their step's digest
and offsets, not a single initial coordinate system. Reports cannot be loaded as
executable plans. `--format raw` emits one current physical file; it never expands
includes. Explicit `--source NAME` chooses a file. Implicit root raw output fails
if it would omit included changes. A one-file `-o` copy always fails if other
changed sources would be omitted; use explicit in-place output for a multi-file
commit. Preview output goes to stdout and diagnostics to stderr.

No source file is written by default. `--dry-run` suppresses output-file callbacks
even when `-o` or `--in-place` is supplied. Review serialization and stream writes
must succeed before any explicit filesystem write begins. `-o FILE` writes a
transformed source copy, rejects aliases of every captured original, and checks
all original source hashes first. It does not overwrite the originals.
`--in-place` updates existing changed named sources only. Stdin/new documents
require a new destination for their own source. Explicitly rooted stdin may edit
captured child files when ownership is enabled.

The process adapter reuses the repository's staging/sync/rename/rollback engine.
All regular-file guards, duplicates, hard-link aliases, symlinks and stale hashes
are checked before replacements. Inputs are rechecked after staging. Ordinary
rename failures restore earlier files; failed restores preserve backups and name
recovery staging in the error. Inspect those files before cleanup. Exact already
written proposed bytes are accepted on retry, without accepting unrelated disk
changes. Successful retries do not delete old recovery artifacts automatically. Retry
means resubmitting the same captured commit batch; rerunning append commands
against already changed files creates a new transformation.
Existing Go-reported file modes are retained; explicit new copies request mode
0600. POSIX hosts enforce those permission bits. On Windows, Go's mode bits
represent the readonly attribute, not ACL access rights; destination-directory
ACL inheritance controls access. The writer does not configure Windows ACLs.
Per-file rename is
atomic when supported by the host filesystem. The group is not crash-atomic, and
exclusive writer access is required; hashes/preflight are not an atomic filesystem
compare-and-swap against hostile concurrent changes.

Defaults: 8 MiB cumulative CLI input/working bytes, 100000 source AST nodes,
256 source occurrences, depth 16, 10000 patches/matches, 256 Mi work units and
32 MiB serialized output. The Go working-byte default is 16 MiB. Work charges
query processing, selected payload and source-byte/reparse processing; limits
bound individual load/query/model phases too. Cancellation is checked between
bounded phases; a host's blocked I/O/callback still owns its deadlines. Node and
source limits remain separate from successful parsing, interpretation or rendering.

Exit codes: 0 success (possibly warnings/no-op), 1 selection/ownership/validation/
rewrite/stale/overlap failure, 2 usage/selector/command-schema error, 3 input,
output (including stale disk guards), resource or cancellation failure.
Validation/preflight/staging failures
leave originals unchanged; commit/rollback failures report recovery details.
This provides selective transformations of captured roff, not full groff execution
or general semantic equivalence guarantees.
