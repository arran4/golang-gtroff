# Create small man documents from Go

The `author` package is a small façade for new canonical man documents. It offers
constructors, mutable Page/Section Add/Append methods, typed literal blocks and
variadic functional options. It reuses the existing semantic model, canonical
man writer and bounded parser validation. It has no dependency on query, includes,
editing, scripting or a CLI command.

```go
page, err := author.NewManPage("mytool", "1", author.WithDate("2026-11-04"))
if err != nil { return err }
if _, err := page.AddSection("NAME", author.Paragraph("mytool - example")); err != nil {
    return err
}
options, err := page.AddSection("OPTIONS",
    author.Definition("--verbose", "Enable verbose output"))
if err != nil { return err }
if err := options.AddDefinition("--quiet", "Reduce output"); err != nil { return err }
return page.WriteRoff(writer)
```

Complete executable external-package examples are in `author/example_test.go`.
Output goes only to an explicit `io.Writer`; the package opens no files and reads
no ambient clock, date, configuration, macro package or network resource.

## Supported construction

- `NewManPage(title, section, ...Option)` requires nonempty literal single-line
  title/manual section. It returns errors rather than creating an invalid page.
- `WithDate`, `WithSource`, `WithManual` configure optional literal metadata.
  `SetTitle`, `SetSection`, `SetDate`, `SetSource`, `SetManual` provide typed setters.
  Dates/manual-section conventions are caller policy, not invented validation rules.
- `AddSection(heading, ...Block)` returns an owned Section. Repeated headings are
  valid. `Append`, `AddParagraph` and `AddDefinition` add section flow.
- `Paragraph(text)` creates flowing literal text; `Definition(term, description)`
  creates a bold term with a flowing description. Empty paragraphs/descriptions
  are valid; lint conventions remain separate.
- `Subsection(heading, ...Block)` begins subsection flow. Following section flow
  belongs to the latest subsection until a new heading, following man semantics.
  Subsections cannot contain nested subsection helpers. This is not a general
  document/layout tree or a collection of all groff requests.
- `Preformatted(text)` preserves literal line layout with canonical LF.
  `LiteralRoff(source)` is the same representation with an explicit intent: show
  roff source as data. `.so`, `.sy`, definitions and other requests remain escaped
  literal text. It never inserts active source or enables I/O/evaluation.

External application helpers can return `Block` or `[]Block` assembled from their
own data. The immutable block values keep their representation private; adding a
project-specific environment-variable helper does not require extending a central
macro switch or implementing a new interface. Subsection constructors snapshot
caller slices. Section methods copy supplied block values and validate the entire
append before mutation. Invalid terms, content, headings or exceeded construction
limits leave the existing page unchanged. Builder mutation is not concurrent-safe.

Metadata access returns a value copy. `Section.Heading()` returns the supplied
heading; canonical flow normalization may change its spacing in output. A zero
Page/Section/Block is not a constructed document or valid content. Options are
trusted Go functions and can use public setters/Add methods; no dynamically typed
option arguments, reflection-based dispatch or code imports are involved.

## Literal escaping and canonical output

All input text is literal data, including backslashes and quotes. The existing
`convert.WriteMan` writer escapes backslashes, quotes and control-line starts;
this façade does not add another quoting/parser implementation. `\fB`, `\"`,
leading periods and apostrophes in user text remain displayed data, not requests.
No active-raw-fragment escape hatch is provided in this initial subset.

Flowing paragraph, term and heading whitespace is canonicalized by the existing
writer. LF/CRLF in paragraph/description strings are soft flow boundaries.
Preformatted LF/CRLF is rendered with LF and preserves line layout; a final LF
may be canonicalized according to the existing writer's preformatted contract.
Metadata, headings and terms are single-line values. Invalid UTF-8, NUL, bare CR
and control characters other than tabs (and body LF) are rejected rather than
silently replaced. Line values have a 4096-byte ceiling. Literal strings are not
parsed as custom inline formatting; the initial term style is simply bold.

`WriteRoff` first builds a bounded semantic model, invokes the existing strict
canonical writer into memory, and validates the generated text with an isolated
built-in registry, strict `ParseAST` and strict `man.Build`. It evaluates no roff
and resolves no includes. Validation/encoding/budget errors leave the supplied
writer untouched. The final writer call can fail or short-write after emitting a
prefix; those errors, including underlying writer errors, propagate.

`AST()` returns the actual canonical generated snapshot, named `<author>`, with
physical byte spans/IDs into that new text. On bounded validation failure it may
return the inspectable partial generated AST plus an error. Original imported
source is not part of a Page; this is neither a source patching API nor a claim
of byte-for-byte input/output round trips. Use existing AST/model APIs to inspect
the generated page. Further Page mutations do not alter earlier AST snapshots.

## Bounds and errors

`WithLimits(Limits{MaxBlocks: ..., MaxBytes: ...})` sets shared page limits.
Nonpositive fields select defaults: 10,000 sections/blocks and 8 MiB literal-data/
canonical-output budget. Ceilings are 20,000 blocks and 32 MiB. Sections and
subsections each count as one block; definitions may create several semantic
nodes. Limits count appended data across all sections and can be reduced only
when existing content still fits. The writer has its own escaped-output check,
so construction can fit while escaping/header output exceeds the final budget.

Generated syntax/model validation additionally uses the existing 100,000-node
and bounded-depth/physical-line parser limits. Parser-specific failures retain
their cause with `ErrInvalid`; they are not reclassified by inspecting error text.
`ErrLimit` identifies façade construction/line/output limits and wraps existing
conversion output limits. `ErrInvalid` identifies invalid literal/zero/nil values
or failure to validate the generated subset. Check returned errors independently
of any inspectable AST. There is no promise of unlimited CPU/memory or callback
sandboxing; these are finite construction and serialization bounds.

Tests include eight readable TXTAR scenarios, external Go examples, writer/limit
errors, invalid literal data, quoted/Unicode/control-line text, CRLF, immutable
block/snapshot behavior, stored-roff display and deterministic generation.
Parse → canonical write → parse semantic equivalence is verified for this supported
subset with the existing conversion normalization API. It does not assert source
spelling equality or complete GNU groff semantics. Rendering/layout engines,
raw active fragments, editing existing documents, rich style builders and dozens
of macro-specific helpers remain separate capabilities.
