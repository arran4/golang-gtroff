# Semantic Linux man-page model

The `man` package turns a source-aware AST into explicit content for future
renderers, servers and extraction tools. It is separate from the syntax AST and
has no layout, HTML, image or GUI dependency. It never expands macros, evaluates
requests, follows references, opens include targets or accesses a man database.

```go
tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
if err != nil {
    // Resource or reader failure: the tree contains only the collected snapshot.
    return err
}
document, err := man.Build(tree, man.Options{Strict: true})
if err != nil {
    // The document and diagnostics remain available for inspection.
    return err
}
options, err := document.SelectSection("OPTIONS", false)
if err != nil {
    return err
}
fmt.Print(options.Text(man.TextOptions{}))
```

See the executable `man.ExampleBuild`, [representative source](../examples/man-tool.1)
and its [tested semantic JSON](../examples/man-tool.semantic.json). The expected
JSON uses the scenario source label `input.1` for deterministic provenance.

## Optional hierarchy and source navigation

Choose `groff.ParseAST` for original physical syntax and `man.Build(tree, options)`
for an opinionated man hierarchy. No new parser mode or filesystem access is
required. Both views remain available together: `document.Syntax()` returns the
**same AST pointer** supplied to `Build`. This accessor is nil after decoding a
semantic document from JSON, which does not carry the private source snapshot.

For example, `.SH ENVIRONMENT` owns following content until the next `.SH`;
`.SS` introduces a child subsection until the next `.SS` or `.SH`. Repeated
headings remain separate nodes. A subsection before any section belongs to the
document. Content before a heading also belongs to the document. Empty or broken
headings remain diagnostic source; a successfully recovered next-line heading
belongs only to the semantic interpretation. `.TP` and `.IP` become items within
a definition list, with styled term runs in `Node.Tag` and description nodes in
`Node.Children`. Indented bodies nest through `indent` nodes. These are bounded
man conventions, not execution of user macro definitions.

`Origin.ASTIDs` identifies semantic contributors, not every byte in a semantic
span. `document.SourceNodes(id)` returns **all intersecting physical AST lines**
in source order, including comments, blank lines, unknown controls, opaque bodies
and terminators. Each returned node retains its original identity, attributes,
diagnostics, children and byte span. The returned slice is caller-owned; its
nodes remain immutable. This provides a source view of a paragraph, definition
or section even when a comment has no semantic node of its own. Section spans
include trailing comments and separators. The root covers the entire captured
source. Container spans overlap their descendants, so concatenate only physical
lines once when reconstructing bytes.

The source lookup validates the caller-owned model, then uses binary searches
in a physical-line index. Its total cost includes model validation plus the
number of returned lines. An invalid ID/model or a JSON model without a syntax
snapshot returns an error; an empty span returns no lines. Changes to semantic
fields cannot edit the original AST. Check both parse/build errors: access to a
partial hierarchy never proves the original read succeeded or reached EOF.

The [external Go example](../man/source_example_test.go) extracts ENVIRONMENT
terms and then visits comments and an unresolved `.so` through the same source
identities. Low-level parsing and semantic building recognize includes as data;
neither follows them. The separate [source-set API and `sources` command](sources.md) provide explicit
bounded include resolution without changing these low-level contracts.

## Content and interpretation

Version 1 contains title/manual-section/date/source/manual metadata, metadata
origin, and an explicit node table: document, section, subsection, paragraph,
definition list, item, indentation, preformatted text, vertical space and
unsupported content. Nodes have stable one-based semantic IDs, parent/child
links, effective state, source spans and contributing AST IDs. Semantic IDs are
deterministic for a snapshot; they are distinct from AST IDs.

`.TH`, `.SH`, `.SS`, `.PP`/`.P`/`.LP`, `.TP`/`.TQ`/`.IP`, `.RS`/`.RE`,
`.nf`/`.fi`/`.EX`/`.EE`, `.br`, `.sp`, `.ft`, `.PD`, `.B`/`.I`/`.R` and the
alternating font macros form the common subset. `.SM`/`.SB` retain small-style
intent. `.SY`/`.OP`/`.YS` provide a simple synopsis paragraph. `.UR`/`.UE`,
`.MT`/`.ME` and `.MR` preserve link/reference targets as data, without fetching
anything. References do not imply a renderer's permission to activate a URL.

Physical text lines flow into paragraphs with a separating space. Blank lines
and paragraph controls terminate flow. Alternating macro arguments concatenate
without inserted spaces, while `.B`/`.I` arguments retain separation. Empty
`.B`/`.I` apply to the next displayed text line. Tagged items have separate styled
tags and body children. Indentation is structural and its raw argument is kept;
numeric roff expressions are not evaluated. Paragraph macros reset the normal font and tag width as appropriate.
Numeric `.RE` levels warn and close
one indentation level, rather than pretending to implement roff's level rules.

Preformatted text retains spaces, tabs and line breaks. `.EX` selects mono
text and `.EE` restores the font used before the example. Known style controls
inside it still create styled runs; unsupported controls become literal fallback
runs with diagnostics. This semantic interpretation of `.EX` bodies is separate
from the AST's opaque syntax grouping. Sections reset normal flow/font/indent
context. A heading without arguments can take text from the next nonempty plain
or style-macro line. The corresponding legacy AST missing-argument diagnostic is
suppressed only for that successfully recovered heading, or an empty alternating
font macro documented as a no-op. Original AST/source remain unchanged. Other
parser errors are retained. Nested next-line style traps warn rather than
pretending to fully implement input-trap evaluation.

Runs carry bold, italic, mono and small flags, links, manual references, hard
breaks, unsupported markers and physical-line provenance. Inline font changes
persist across physical text lines; `\fP` restores the previous font. Common
escaped punctuation, spaces, `\&`, `\c` continuation and the existing special
character decoder are supported. Unsupported fonts, size escapes, strings,
registers and unknown glyphs remain literal syntax with diagnostics. Definitions,
ignored regions and preprocessors are preserved as inspectable unsupported
blocks. They are not expanded or silently discarded. The model follows parser-owned
`BlockSource` boundaries for every captured block, including definition variants
(`.de1`, `.dei`, `.dei1`, `.am1`, `.ami`, `.ami1`) and custom block handlers.
Stored `.TH`, `.SH`, font, link, fill and indentation controls remain literal
bytes; they cannot change page metadata, sections or following content. Nested,
unterminated and unresolved indirect captures keep their AST recovery boundary
and diagnostics. A capture inside preformatted content or a pending tag remains
in that semantic context. Model construction never reruns a block handler or
interprets its formatted token as page content. Ordinary `.EX`, `.nf` and `.RS`
structures keep their documented semantics unless a parser block handler
explicitly overrides them.

See the [captured-definition example](../man/captured_example_test.go), four
focused TXTAR cases in `testdata/man/`, and the cross-feature
[`captured-definition` scenario](../testdata/scenarios/captured-definition.txtar).
That scenario checks literal retention through HTML, PNG encoding, canonical
roff and HTTP. Canonical roff emits escaped preformatted text for unsupported
source; its original unsupported syntax kind is outside semantic loop equality.

## Selection, text and source

`SelectSection(heading, normalized)` returns all repeated matching sections and
subsections in source order. Normalized matching folds whitespace and case.
`SelectAST(id)` resolves a contributing heading/group/line AST ID to its smallest
semantic block. Supported `.RS`, `.nf` and `.EX` AST block IDs select their
indent/preformatted group, just as the opener line does. A captured region
records its outer/nested block IDs and physical header/body/terminator line IDs
on the literal semantic owner. Selecting any of those IDs returns the complete
unsupported block, preformatted context or definition item containing it.
These contributing IDs survive semantic JSON round trips, even though original
raw source does not. Inline IDs without a block-origin entry are not fabricated;
CLI `--node` first maps an inline ID to its physical line. Comments outside
captures and malformed structures without a semantic group remain selection
errors. See the [executable AST selection example](../man/ast_selection_example_test.go).
`SelectNode(id)` and inclusive `SelectRange(first, last)` select semantic blocks
and their descendants. A range preserves whole paragraphs rather than cutting
through their source-line runs. Overlapping roots produce each selected node
once. A missing selection returns an error.

A selection contains metadata and one upstream `State` per root, including font,
previous font, fill mode, indentation stack, tag indent, link and paragraph
distance. Renderers can therefore interpret a selected paragraph without
replaying arbitrary earlier source. Parent IDs outside a selection continue to
refer to the full document; selections do not invent replacement identities.

`FindTerm(term, normalized)` finds content-bearing semantic nodes, and
`SelectTerm` extracts matching blocks with their context in one selection. `Text(id,
options)` or `selection.Text(options)` extract derived display text. Normalized
text folds whitespace; ordinary extraction keeps paragraph, tag and preformatted
boundaries. Unsupported text remains visible. `Raw: true` takes precedence and
returns original source slices; overlapping selected spans are merged without
pulling in unselected gaps. Section source spans include trailing controls,
comments and blank lines up to the next heading. `Source()` always retains the
original parsed bytes, independently of canonical or extracted content.

Source columns and offsets follow the AST contract. Run origins normally cover a
physical line, rather than claiming character-exact provenance after decoding;
a recovered heading origin includes its header and following text line.

## Round-trips, errors and bounds

The exported model fields are caller-owned mutable data. `Validate()` checks
version, kinds, identity, parent/child consistency, reachability, cycles, source
ranges and bounded depth. Call it after accepting a JSON model. Selectors and
`Canonical()` validate before traversing. Exported JSON has no private original
source; after a JSON round-trip semantic extraction still works, while `Source`
and raw extraction are empty. Keep the original AST/source separately when raw
bytes are needed.

`Canonical()` returns deterministic semantic JSON with source identity and
diagnostics removed and adjacent equivalent runs coalesced. It is a comparison
projection, not a roff writer. Model JSON round-trips and equivalent physical
line wrapping are tested against this projection; no byte-exact regeneration
claim is made. Coalescing and tree traversal are iterative and linear in content
size, apart from diagnostic sorting and bounded-depth state copies.

Permissive build returns useful fallback content plus diagnostics. Strict build
returns the model and an error if any semantic or unrecovered AST diagnostic
remains. `ErrLimit` identifies semantic budget exhaustion in either mode. The
builder defaults to 8 MiB source, 100,000 nodes and 128 nested indents, clamped
where appropriate to structural safety limits. It can preserve one unsupported
tail when a budget is exhausted. The source snapshot remains available. Check
both the upstream `ParseAST` error and the semantic build error before claiming
that a document is complete.

Parser work under #7/#10/#11/#13 remains separate. The bundled `groff_man.txt` was consulted for paragraph font resets, example
font restoration, tag handling and alternating macro semantics. This phase adds no interpreter
and does not fix the parent parser's global registry, includes, evaluator or
catalogue generation. The implementation is stacked on #33 → #32 → #19.

## Scenarios and examples

Embedded `testdata/man/*.txtar` files contain `input.1`, optional `options.json`
and `error.txt`, plus `expected-man.json`, `expected-text.txt` and
`expected-canonical.json`. A scenario may select a section, normalize text, or
require strict failure. `astBlock` selects one named AST block; `astLine` selects
one exact physical line body; `raw` requests source extraction. Section/block/line
selectors are mutually exclusive. Semantic JSON is generated by real code, checked for
ancestry/provenance/determinism and round-tripped through the canonical model.
The representative source and expected JSON in `examples/` are checked against
the same scenario. Normal runs never rewrite them:

```sh
go test . -run '^TestManScenarios$' -update-man
go test ./...
go test ./man -run '^$' -fuzz '^FuzzManModel$' -fuzztime=15s
```

Refresh is explicit and package-local. Review golden diffs before committing.
TXTAR text expectations have a final LF; separate tests assert exact raw section
bytes, continuation behavior and source boundaries. HTML export is the next
phase (#23); no render backend is included here.
