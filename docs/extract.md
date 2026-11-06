# Extract selected documentation

Query locates objects. Extraction constructs records and exports content in a
representation suited to the next task. It uses the same stateless selector IR,
source set and optional man model as `query`; there is no second document parser.
The initial API/JSON schema is version 1, with no general format-conversion or
stable-API promise.

```sh
gtroff extract 'section[name="ENVIRONMENT"] definition' page.1
gtroff extract --format csv --fields term,description,file,line \
  'section[name="ENVIRONMENT"] definition' page.1
gtroff extract --format raw 'definition[term="TOOL_HOME"]' page.1
gtroff extract --format text --normalize 'section[name="DESCRIPTION"]' page.1
gtroff extract --format grep 'definition' page.1
gtroff extract --format html -o environment.html \
  'section[name="ENVIRONMENT"]' page.1
gtroff extract --view syntax --format raw 'macro[macro=".B"]' page.1
```

Flags precede the selector and optional input filename. Omit FILE or use `-` for
borrowed stdin. A named root resolves literal local includes through the shared
source loader, relative to each referring file and confined to `--root` (default:
input directory). `--no-includes` retains a single-source snapshot and references;
stdin needs an explicit root for includes. No system macro search, network fetch
or roff execution occurs. One invocation has one root source set, which can contain
many explicit include occurrences. Enumerating unrelated collection roots remains
a host/shell workflow rather than an implicit directory crawl.

## Records and source ownership

A record retains:

- one-based extraction index and original query index/ID;
- syntax/semantic view, actual node kind and original source occurrence/digest/EOF;
- original file-local span, AST contributor IDs and optional semantic ID;
- heading context, optional man metadata and upstream formatting state;
- separate definition term/runs and description, selected text and normalized text;
- exact raw source, copied semantic subtree blocks, includes and diagnostics;
- unsupported-content evidence and optional rendered HTML.

Repeated definitions remain separate records, including equal terms. Definition
`Text` includes the term and body; `Description` omits only the root item's term.
Nested semantic blocks remain in the body. Term-only selections retain styled
term runs and omit the description. Generic sections, paragraphs, blocks, syntax
lines and source wrappers can also be selected; fields irrelevant to that object
are absent. `Semantic` describes whether the record uses the optional man model;
a source wrapper in semantic query view still exposes syntax text.

Heading context uses original source/semantic identities. Included fragments gain
context from the original directive owner's heading ancestry. A child's own
headings are appended rather than pretending the runtime heading state changed.
Metadata, interpretation and text stay file-local. Included sources do not expand
inside a parent's raw/text/HTML span; select descendant definitions to obtain
child records. Included-source relationships remain visible in `Includes` and
original source data. Loaded source, modeled semantics and executed groff are
separate concepts.

Blocks keep original local IDs and references to parents outside the selection.
They are useful subtree records, not an independently validated `man.Document` or
a canonical replacement document. Runs retain original physical spans. Record
slices, nested runs, metadata and formatting state are caller-owned copies; source
ASTs remain immutable. Query IDs/indices are labels tied to one query snapshot,
not persistent editing cursors.

Unsupported nodes/runs are kept as literal source/blocks and create positioned
`unsupported-content` diagnostics. Existing source, optional-model and renderer
diagnostics remain visible, deduplicated by source/position/severity/message.
Uninterpreted syntax is not declared semantically supported merely because it
can be extracted. No variable classification or Great-Library-of-Environ YAML
schema is embedded in this generic library.

## Output contracts

| Format | Representation |
| --- | --- |
| `json` | Full versioned report, records, diagnostics and completion/input facts |
| `jsonl` (default CLI) | Versioned summary envelope, then versioned record envelopes |
| `raw` | Exact selected per-file byte spans, with no added delimiter/newline |
| `text` | Syntax text or decoded optional-man text, newline between records as needed |
| `grep` | Original file:line:column and normalized selected text |
| `csv`, `tsv` | Quoted field records, including separate term and description |
| `html` | Independent selected semantic fragments using the existing renderer |

Record order follows query enumeration. Overlapping selections deliberately repeat
source/content; extraction does not merge their ranges or silently deduplicate.
Raw spans preserve CRLF and invalid UTF-8. JSON follows `encoding/json` replacement
rules and is not a byte-exact source archive. `Text` preserves case and decoded
content; `NormalizedText` folds whitespace **without** the query predicate's case
folding. `--normalize` selects that representation for text output only.

CSV/TSV default columns are `index,file,line,kind,term,description,text,unsupported`.
`--fields` accepts: `index`, `query-index`, `id`, `source-id`, `file`, `digest`,
`line`, `column`, `offset`, `end-offset`, `kind`, `name`, `headings`, `term`,
`description`, `text`, `normalized-text`, `raw`, `unsupported`. Description fields
can contain newlines and are quoted by `encoding/csv`. Headings use a readable
` / ` path; JSON retains their individual identities. Fields on other formats,
unknown columns, and normalize on non-text output are usage errors.

HTML is a supported-man preview, not exact roff. It uses `renderhtml.RenderSelection`
with fragment mode, omitted fonts and anchors disabled, preserving selection order
and repeated overlaps. Virtual terms become paragraphs of their existing styled
runs so unselected descriptions are not added. Syntax and source-wrapper selections
cannot produce semantic HTML. No extra converters, fonts, browser or external
formatter are introduced. Renderer diagnostics remain in extraction reports.

## Go API and contributed exporters

```go
selector := query.Select(query.Section).
    Where(query.Name, query.Equal, "ENVIRONMENT").
    Descendant(query.Select(query.Definition))
report, err := extract.Select(ctx, set, selector, extract.Options{})
```

`Select` calls the existing query executor and constructs records with a shared
work budget. `FromQuery(ctx, set, result, options)` accepts an existing result or
ordered subset of its matches; original query indices need not be contiguous.
It checks occurrence metadata/digests and source-backed IDs/spans, then derives
content again from the supplied immutable source. Editing `Match.Text`, `Term`
or `Raw` does not change extraction. It rejects stale/mismatched provenance with
`ErrSnapshot`. It does not authenticate the caller's selection or re-run an absent
selector, and query ID/index labels are not a security credential. Parser registry/
package context must match the source snapshot used for the query.

`FromQuery` uses its own construction budget; `Select` deducts actual query work
using additive, nonserialized `query.Result.WorkUnits`. Query results retain their
existing JSON contract. Set `Options.HTML=true` to prepare HTML during construction;
this explicitly charges render work/data. Other outputs do not invoke rendering.

```go
err = report.Write(ctx, writer, extract.CSV, extract.WriteOptions{
    Fields: []string{"term", "description", "file", "line"},
})
```

`Report.Export` accepts an explicit trusted `Exporter` interface or `ExporterFunc`
for contributed serializers. It supplies a cancellation-aware bounded writer and
retains writer/limit errors even if a callback ignores them. A callback must honor
context for its own computation; it is host code, not an untrusted-script sandbox.
There is no global plugin registry, configuration search or code importer.
Working external-package examples are in `extract/example_test.go`.

## Failure, budgets and explicit files

`Complete` means record construction finished over the supplied match sequence.
`QueryComplete` means selector enumeration finished; `Source.Complete` means the
original reader observed EOF. Each is independent of successful interpretation.
`QueryError`, `InputError`, extraction `Error` and source-linked diagnostics retain
those different failure facts. A fully constructed prefix of an incomplete query
can have Complete=true and QueryComplete=false. `FromQuery` returns `ErrIncomplete`
for incomplete/error query results; `Select` also returns the original typed query
error. Check the returned error even when records exist.

Strict extraction rejects any source/model/extraction/renderer diagnostics, input
error labels or selected incomplete captures, retaining useful records/report.
CLI strict mode also asks the source loader to reject diagnostics. JSON/JSONL on
stdout retain strict reports; other strict output is withheld. Non-JSON stdout
formats put diagnostics on stderr. Permissive stdout can contain a valid prefix
on input, limit, cancellation or writer failure; consumers must check exit status.
A writer can fail after bytes were emitted. Raw/text output alone cannot carry
all global facts; use structured reports when that evidence must travel with data.

Default construction budgets: 200,000 source/model nodes, 10,000 records, 16 MiB
estimated materialized record data and 64 Mi charged work. Limits count repeated
selected content; large subtrees are checked incrementally. Supported ceilings:
1,000,000 nodes, 100,000 records, 64 MiB record data and 1 Gi work. Nonpositive API
construction limits select defaults; CLI limits must be positive. The output writer
has its own actual byte limit (32 MiB default, 64 MiB ceiling). These units bound
phases; they are not CPU instructions or a wall-clock deadline for custom host code.
Source input defaults/limits are the shared loader's separate byte/node/file/depth
budgets. `--max-*` flags expose those policies. HTML validation scans a whole source
model per fragment and is explicitly charged; large HTML exports can reach work
limits sooner than plain records. No native platform runtime guarantee is inferred
from a cross-build.

`-o FILE` is an explicit host file-output capability. Serialization is staged and
bounded; construction, strict, input and serialization failures leave the named
destination untouched. Successful output uses same-directory temporary file,
sync/close and regular-file replacement, retaining existing permissions (new file:
0600). Symlink/special destinations and aliases of **every loaded source** are
rejected, including included files and their hardlinks. Source paths must remain
available for identity checks. Atomic rename semantics come from the host filesystem;
there is no delete-and-retry fallback or guarantee against concurrent hostile
filesystem replacement. No in-place edit, release or implicit multi-output write.

CLI exit codes: 0 complete, 1 strict diagnostics, 2 usage/selector errors,
3 input/output/resource/cancellation failures. Named output is never written on a
failed operation. If an input cannot be opened and no snapshot exists, the CLI
reports the failure without inventing records.

Tests exercise all formats, exact CRLF/invalid bytes, repeated terms, included
heading context, opaque/unknown/malformed source, unsupported semantics, strict
partial reports, filtered/stale query results, callback/writer failures, budgets,
cancellation and original-file protection. Nineteen CLI TXTAR scenarios include
JSONL summaries and quoted CSV descriptions; executable tests verify the actual
generated command and atomic boundary. The existing licensed environ(7) snippet
checks real source/text provenance; synthetic ENVIRONMENT examples demonstrate the
generic term/description workflow without claiming a full external corpus crawl.
