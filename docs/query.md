# Stateless structural queries

`gtroff query` selects source-aware roff syntax or an optional man hierarchy.
The `query` Go package compiles selectors and typed builders into one immutable
representation and evaluates it without changing documents, opening files or
retaining procedural state. Tengo sessions and editing are separate workstreams.

```sh
./gtroff query 'section[name="ENVIRONMENT"] definition' document.1
./gtroff query --format csv --fields term,file,line 'definition' document.1
./gtroff query --view syntax --format raw 'macro[macro=".B"][arg.0="TOOL_HOME"]' document.1
./gtroff query --no-includes --format grep 'section[name="DESCRIPTION"] paragraph' document.1
./gtroff query --view syntax 'inline[inline-kind="StringRef"]' document.1
```

Options precede the selector and optional file. Omit the file, or use `-`, for
stdin. Quote the selector as one shell argument; use `--` for a selector or path
that would otherwise look like a flag. Invalid selectors/options are rejected
before input is opened. A complete query with no matches succeeds.

## Go API

```go
selector := query.Select(query.Section).
    Where(query.Name, query.Equal, "ENVIRONMENT").
    Descendant(query.Select(query.Definition))
result, err := query.Execute(ctx, sources, selector, query.Options{})
```

`query.ParseSelector` returns the same representation. `Where` adds a predicate
to the last step; `Child` and `Descendant` append another selector. Builder
operations copy their private slices, so reusing a base selector cannot change
another query. `Validate` reports invalid builders before execution. Complete
external examples in [query/example_test.go](../query/example_test.go) show input
loading, builder/selector execution and explicit CSV output through an `io.Writer`.
Use `source.Load` or `source.Read` to supply the [explicit source set](sources.md).

## Grammar

```text
selector = step { whitespace step | [whitespace] ">" [whitespace] step }
step     = kind { "[" field [operator value] "]" }
operator = "=" | "!=" | "*=" | "^=" | "$=" | "<" | "<=" | ">" | ">="
```

Whitespace means descendant, `>` means direct child, and `*` matches any kind.
Predicates on a step are ANDed. Values are double-quoted Go string literals,
single-quoted strings, or bare ASCII identifiers/unsigned decimal numbers.
Double-quoted strings support Go escapes such as `\"`, `\\`, `\n`, `\u96ea` and
`\xff` (the latter can select preserved invalid source bytes). Single-quoted
strings support `\'`, `\\`, `\n`, `\r` and `\t`; literal newlines are rejected.
Bare identifiers contain letters, digits, `_`, `-` and `.`. Quote source paths
containing slashes or values containing spaces. Whitespace around operators and
inside brackets is allowed; a predicate immediately follows its kind.

`=` and `!=` compare exact values. `*=`, `^=` and `$=` test substring, prefix and
suffix. Location/occurrence/depth fields use unsigned numeric comparisons, also
for equality; text operators on numeric fields are errors. Ordering operators on
text fields are errors. `normalized-text` folds whitespace and Unicode case on
**both** operands; it does not perform Unicode normalization. `[field]` requires
a nonempty value, except `[opaque]`, which requires true. Opaque equality accepts
only true/false. Missing string properties have the empty value.

There are no groups, pseudo-classes, nested bracket expressions, assignments,
variables, SQL, regular expressions or callbacks in this initial language.
Unsupported grammar is an error, never a weaker interpretation. Syntax errors
carry zero-based **selector-byte** offsets and unwrap `query.ErrSelector`;
these offsets do not refer to the roff input. Selectors are limited to 16 KiB
of valid UTF-8, 64 structural steps and 128 predicates. Builders share step/
predicate/value bounds. Fields/kinds/operators form explicit extension points
in the shared representation; arbitrary script evaluation is not a predicate.

## Views and vocabulary

`--view syntax` uses original AST kinds: `document`, `section`, `subsection`,
`block`, `line` and `inline`. `macro`, `request`, `comment` and `unknown` are
convenient filters for physical lines by token category; their result kind is
still `line`. Inline syntax uses `inline-kind` values such as `FontChange`,
`StringRef`, `RegisterRef` and `SpecialChar`, with the parsed reference name in
`name`. Syntax text leaves escapes unexpanded. Comments and delimiters are still
selectable even if their text contribution is empty.

The default `--view semantic` uses the bounded **man** interpretation: `document`,
`section`, `subsection`, `paragraph`, `definition-list`, `definition`, `term`,
`indent`, `preformatted`, `space` and `unsupported`. `definition` labels existing
`man.Item` nodes (`item` is a selector alias); `term` is a separate projection of
that item's styled tag. It retains the item's semantic ID and its own identity
suffix and physical tag-line span/AST IDs. Descriptions remain child paragraphs
or other semantic blocks. A section commonly owns a definition-list which owns
its definitions, so `section definition` works while `section > definition`
ordinarily does not. A definition directly owns its projected term.

Repeated headings remain separate and results retain source order. Subsections
nest within sections, with orphan content following the existing man model.
Opaque macro definitions/preprocessors remain unsupported literal content;
stored `.SH` or `.TP` controls do not become live headings or definitions. The
syntax view can inspect the original physical material independently. Semantic
view is an opinionated man convention, not an ms interpreter or successful groff
execution. Select syntax view when inspecting other packages without that view.

Both views expose a `source` wrapper for each source occurrence. It contains the
file-local document root. Included source wrappers are children of the referring
line (syntax) or its smallest semantic owner, usually the preserved unsupported
include instruction. This allows `section[name="ENVIRONMENT"] definition` to
reach an included fragment. All underlying AST links and IDs stay unchanged.
The query graph is a separate traversal; it never reassigns AST parents.

Semantic grouping, font/heading state and text extraction remain **local to each
physical file**. An included heading does not change the parent file's following
content or retroactively reinterpret it. A container's `text`/`raw` belongs to
its own file; selecting its descendants can separately select child-source
content. A `source` wrapper's text is the original syntax extraction for its
whole file in either view. This is source inspection, not runtime macro/include
execution or an invented multi-file canonical AST.

## Attributes

| Field | Meaning |
| --- | --- |
| `name` | Source filename, heading, term, inline reference name, or control spelling |
| `macro` | Dot-normalized control name on a line or source-backed block opener |
| `package`, `token-kind` | Parsed token metadata; semantic nodes use man conventions |
| `inline-kind` | Parsed inline category in syntax view |
| `arg.N` | Zero-based parsed argument; Go builder uses `query.Arg(N)` |
| `text`, `normalized-text` | View-specific content, with explicit normalization |
| `raw` | Exact bytes of the original per-file span |
| `term` | Definition/item tag text, also on projected term nodes |
| `source`, `source-id` | Root-relative source name and source occurrence ID |
| `id` | Stable per-snapshot query identity (view/source/node, with term suffix) |
| `line`, `column`, `offset`, `end-offset` | Original span start or exclusive end byte |
| `include-depth` | Nesting depth of the owning source occurrence |
| `opaque` | Preserved uninterpreted syntax on the node, not a summary of all descendants |

Argument indices are limited to 0..1023. Source positions use the AST's original
half-open byte offsets, one-based lines and rune columns. Container spans overlap
their contents. Term positions identify complete physical source lines, not
character-exact provenance after semantic decoding. Header attributes can be
inspected on source-backed groups without treating the group as another macro
line. Unknown and unsupported source remain visible with diagnostics.

## Results, includes and output

Every match contains schema version 1, a one-based `index`, a view/source/node
identity, kind, original span and contributing AST IDs, optional semantic ID,
source occurrence/parent edge/depth, exact-source SHA-256 digest, EOF fact, name,
term, text, normalized text, raw source and relevant diagnostics. Ordinals and
identities are deterministic for the same captured source set/options; they are
not persistent addresses across edits. A digest identifies captured bytes, not a
claim that a partial reader reached EOF. Later editors can use identity/digest
without conflating them with a mutable cursor in the query language.

Named-file input uses the shared source policy: bounded literal includes by
default, `--root` for a larger allowed root, `--no-includes` for one file. Stdin
needs an explicit root to load anything. Repeated includes have distinct source
occurrences. Missing/broken includes retain their references and diagnostics;
queries do not silently fetch system or remote resources. Recognition, loading
and semantic support remain separate facts; a resolved `.so` can still produce
a man-model unsupported-instruction diagnostic.

| Output | Contract |
| --- | --- |
| `json` | Full result report, global diagnostics, completeness and error fields |
| `jsonl` | One independently versioned match per line; global diagnostics go to stderr |
| `text` | Each selected view-text contribution, newline terminated |
| `raw` | Concatenated exact selected source spans, without added separators |
| `grep` | Original `file:line:column:` plus one case/whitespace-normalized text line |
| `csv`, `tsv` | Quoted location/content records, header and explicit field order |

CSV/TSV defaults to `index,file,line,column,kind,term,text`. `--fields` can select
`index,id,source-id,file,line,column,offset,end-offset,kind,name,term,text,normalized-text,raw`.
Other formats reject this option. Overlapping selections deliberately repeat
content in text/raw formats; these formats are selections, not reconstruction
of a whole document. Full JSON includes global diagnostics; other formats put
them on stderr even if the selection is empty. Check exit status for partial
JSONL/text/raw/record streams. JSON replaces invalid UTF-8 as usual; `raw` and
library `Match.Raw` retain bytes exactly.

Exit codes: 0 complete query (possibly no matches), 1 strict syntax/include
rejection, 2 selector/usage error, 3 I/O/resource/cancellation failure. Strictness
is a source-loading policy; semantic interpretation findings remain separately
labelled query diagnostics. `Result.Complete` means enumeration over the supplied
snapshot finished, not that all source reads/includes/semantics succeeded.
`Source.Complete`, `inputError`, `error` and diagnostics preserve those facts.
On query failure returned matches are a valid prefix and `Complete` is false.
No write to source files occurs.

## Bounds and verification

Library defaults: 200,000 nodes for each AST preflight and the projected graph, 64 million node/predicate/text-byte
work units, 10,000 matches and 16 MiB of materialized match data. The latter is an
accounting bound on match fields, not exact encoded JSON size or caller-supplied
source storage. Source loading has its own cumulative byte/node/file/depth bounds.
Query construction, hashing, traversal, text predicates and matching consume
work. Per-file man construction remains bounded by its existing limits and is
checked for cancellation before and after that bounded work; arbitrary blocked
input I/O remains a host responsibility. CLI `--max-results`, `--max-work` and
`--max-result-bytes` expose output/execution limits (ceilings 100,000 matches,
1,073,741,824 work units and 64 MiB respectively). A limit is an error, never a
successful silent truncation.

Structural matching propagates selector-step bitsets through one preorder walk,
with O(projected nodes × selector steps) structural work and no exponential
ancestor backtracking. Text work is separately charged, including overlapping
container extraction. Dense ancestor chains have deterministic work-budget tests.

Tests include equivalent Go/parser queries, quoted/invalid selectors, include
provenance, repeated headings, subsections, opaque material, malformed/partial
source, CRLF, invalid UTF-8 byte predicates, cancellation, resource limits,
serialization/short writes, ten CLI TXTAR scenarios and an actual generated-binary
smoke. The existing licensed `testdata/environ_snippet.7` checks real environ(7)
paragraph/reference selection; the representative ENVIRONMENT fixtures demonstrate
term/description extraction without environment-specific library logic. No full
Great-Library-of-Environ crawl is claimed from these fixtures.

Run opt-in reproducible benchmarks with:

```sh
go test ./query -run '^$' -bench . -benchmem
```

They construct deterministic 10/1,000-definition documents, check result counts,
and measure parsing plus complete stateless execution (including projection).
Compare on the same Go/toolchain/CPU; no absolute timing CI threshold is imposed.
