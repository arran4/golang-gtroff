# Compare documentation revisions

`gtroff diff BEFORE AFTER` compares two explicit source snapshots and retains old
and new source locations. The default assessment is `man-content-projection`;
`--view source` assesses `exact-source-bytes`. These are separate views, not a
claim that identical rendered output, source bytes and interpreted meaning are
interchangeable. No stable API or complete groff compatibility is promised.

```sh
gtroff diff old/tool.1 new/tool.1
gtroff diff --format json old/tool.1 new/tool.1
gtroff diff --format jsonl old/tool.1 new/tool.1
gtroff diff --view source --no-includes old/tool.1 new/tool.1
```

Flags precede both input names. Stdin `-` is allowed on one side. Named files
resolve bounded local includes through the shared source loader, relative to each
referring file and confined to `--root` (default: each root file's directory).
Stdin needs an explicit root for includes. `--no-includes` deliberately compares
only captured root sources and their references. There is no directory crawl,
network lookup, roff execution, external comparison program or file writing.

The initial Go API is `docdiff.Compare(ctx, before, after, options)` with two
`docdiff.Input{Name, Sources, Error}` values. Names are labels; Sources are explicit
immutable `source.Set` snapshots. Input.Error retains any source-loader error.
The library acquires no I/O capability. See `docdiff/example_test.go` for complete
external-package examples.

## Compared content and matching

A single existing query/extraction Document projection per revision supplies
file-local man metadata, nodes, decoded runs, relationships and source origins.
Diff indexes those records; it does not implement another parser or selector IR.

Compared objects include metadata, sections, subsections, definitions, references,
preamble prose, uninterpreted regions and, in source view, captured source
occurrences. Section/subsection/definition prose belongs to the nearest such
owner; nested objects have their own findings rather than cascading every child
edit into all enclosing sections. Definition terms are identity-bearing, compared
case-sensitively. Renaming a term/heading generally produces removed/added objects;
there is no arbitrary fuzzy rename guess.

Semantic content ignores font styles, relative indentation, explicit spacing
requests, paragraph splits and flowing whitespace differences. It preserves case,
preformatted line/space data, literal metadata values and explicit reference
targets. Equivalent font macros/escapes, quoting and CRLF wrapping can therefore
compare equal while source view reports changed bytes. Preformatted indentation
can be meaningful program/document data and remains significant.

References come from existing decoded link/manual runs. Font-split and wrapped
runs are coalesced. A reference label plus its type identifies the relationship;
its target is compared content. An unchanged label with a new URL can be changed;
a changed manual name/label may instead be removed/added. Legacy styled prose
that resembles a reference is not promoted into an inferred link. This is the
existing model's relationship inventory, not a full reference resolver.

Root filenames are labels: `old.1` and `new.1` can compare equal. Included occurrence
keys use directive ancestry, normalized literal target, request and repeated-target
ordinal, retaining actual filenames/IDs in locations. Included objects also retain
the referring semantic owner's scope. Each source still has file-local man state;
loading includes does not simulate global groff execution.

Matching first uses object kind, structural/source scope and identity. A unique
identity can then match across scopes and be reported as moved. Repeated labels
can be aligned by unchanged source digest/native identity/span or unique content
payloads; remaining unique candidates align after those eliminations. These rules
do not pair arbitrary duplicates by result ordinal. Unresolved many-to-many or
many-to-one groups are explicit Ambiguities with both candidate lists. Identical
source snapshots do not become ambiguous solely because terms repeat.

Relative sibling reorder uses longest increasing subsequences. Inserting/removing
an earlier sibling does not mark every later object moved. A swap can have one
minimal deterministic moved finding. A content edit plus a move is a changed
finding with moved=true and contentChanged=true. Section moves do not automatically
create identical move findings for every child whose local scope stays intact.
Source view compares each captured occurrence's bytes and relationships; it is
not a line-hunk/patch engine or a filesystem-wide file-rename detector.

## Unsupported and unavailable evidence

Unsupported blocks and inline source stay explicit. Their actual byte spans are
retained as opaque objects and diagnostics; their changes use confidence
`uninterpreted-source`. Known projected objects use `projected-content`; source
view uses `source-bytes`. An unsupported inline value makes its enclosing content
uncertain as well. Unknown source is not silently discarded or declared evaluated.
Metadata uses the existing model's actual title-line origin, including when
comments precede `.TH`. Source IDs/AST IDs/semantic IDs remain local to their
revision; they are not persistent edit cursors.

Missing/unresolved/denied/cyclic/resource-limited includes, truncated captures,
reader/close failures or absent snapshots refuse comparison with an incomplete
report and no invented removals. In semantic view, syntax errors also refuse
comparison; source view can compare fully captured malformed bytes independently
of their parser diagnostics. Explicitly disabled includes are a root-only policy,
with diagnostics retained; incompatible disabled/expanded inputs are rejected.
Use `--no-includes` explicitly when root-only inspection is intended.

Opaque/programmatic syntax can affect real groff output beyond observed content.
An equal projected report is not proof of runtime/rendering equivalence. The
assessment and diagnostics make that limit explicit. The comparison does not
infer macro execution, unsupported-package semantics or external resources.

## Results and output

Report schema version 1 contains before/after inputs, source digests/EOF/edges,
input diagnostics, compared-object counts, changes, ambiguity candidates and
comparison diagnostics. A Change carries kind, old/new Object locations and
payload digests, contentChanged/moved flags, confidence and reason. Object.Value
is a deterministic content payload; it is not original roff except for opaque
and source objects. Source keys/scopes are relation keys, not another query grammar.

Changes sort by source key, original before offset when available (otherwise
after offset), kind, identity and change kind. One-based indices are assigned even
on partial comparison errors. Input and comparison completion are separate facts;
counts can describe an enumerated prefix after a resource failure.

`Complete` means comparison enumeration finished. `Status` is equal, different,
ambiguous, strict-invalid or incomplete. `Different` means the returned changes
contain observed differences, including an emitted prefix on failure. Ambiguity
is separate from asserting changes. `--strict` rejects any source/model/opaque or
ambiguity diagnostics while retaining the inspectable report. Source-loader strict
errors remain classified alongside their snapshots. Strict-invalid can still
have Complete=true because enumeration completed before policy rejection.

`Report.Write(ctx, writer, format, maxBytes)` has text, JSON and JSONL exporters.
JSON contains the full report. JSONL starts with a summary envelope and counts,
then change and ambiguity envelopes, all schemaVersion 1. Text includes both
locations, payloads, candidate locations and diagnostics. Filenames are quoted
in text, preserving spaces/newline evidence without inventing a grep contract.
JSON follows UTF-8 replacement rules; exact byte identities use the retained
source snapshots/digests, and source Object.Value remains exact in Go memory.
Neither JSON nor semantic payloads are a byte-exact source archive.

CLI exit codes are 0 equal in the chosen assessment; 1 differences, ambiguity or
strict diagnostics; 2 usage; 3 input/output/resource/cancellation failure. Inspect
the report status to distinguish them. Serialization/writer errors can leave a
prefix; callers must check both errors and status. No automatic patches or writes
occur. Output byte limits are checked against actual writes.

## Bounds and verification

Defaults: 200,000 projection/preflight nodes per revision, 10,000 objects per
revision, 10,000 changes plus ambiguity groups, 32 MiB comparison data, 16 MiB
materialized result data and 64 Mi shared work units. Ceilings are 1,000,000 nodes,
100,000 objects/changes, 64 MiB data/results and 1 Gi work. Nonpositive API limits
select defaults; CLI limits must be positive. The writer separately defaults to
32 MiB actual output, capped at 64 MiB. Limits are finite phase/data bounds, not
CPU instruction or peak-memory accounting promises.

Work is shared across input inspection, both query/extraction projections,
content indexing, matching and result emission. Additive nonserialized
`extract.Report.WorkUnits` supports composition without changing extraction report
JSON. `extract.Record.MetadataOrigin` exposes the already-existing model origin
instead of rediscovering title positions from source syntax. These are narrow
consumer-facing additions, not new parsers.

CLI source bytes/nodes/source occurrences are cumulative across BOTH root loads.
When a fatal first read or shared budget stops loading, the second root is not
opened and no comparison result is fabricated. The source loader retains its
minimum admission allowance for another root. Context cancellation stops work;
caller-supplied blocked readers and Go callbacks remain responsible for deadlines.

Tests include formatting/source contrasts, add/remove/change/move/reorder,
insertion without false moves, unique duplicate alignment, ambiguity, real changed
licensed environ(7) prose, included provenance, actual title-line positions,
unsupported blocks/inline syntax, malformed source, strict policy, partial ordinals,
limits, cancellation and writer failures. Twenty-two CLI TXTAR scenarios and
built generated-executable tests exercise stdout formats and exit behavior.
Deterministic 10/1000-definition benchmarks and identical-snapshot fuzzing supplement
those focused tests; no absolute timing gate or universal-corpus claim is made.
