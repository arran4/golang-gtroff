# Semantic HTML/man conversion

Use `gtroff convert --from html --to roff -i article.html -o article.1`, or
`gtroff convert --from roff --to html -i article.1 -o article.html`. Both default
to stdin/stdout (`-`). HTML uses the same self-contained output and bundled Go
Regular/Bold/Italic/Mono fonts as `render`; `--fragment` and `--assets omit-fonts`
are available for HTML output. Use `render` for selection, custom fonts and
appearance options. No groff, browser, network or installed font discovery runs.

## Supported profile

| HTML | Man semantics / canonical source |
| --- | --- |
| First `h1` | Title, manual section `1`; emitted as `.TH` |
| `h2`, `h3` | Section/subsection, `.SH` / `.SS` |
| `p`, ordinary text, `br` | Paragraphs, explicit line breaks |
| `strong` / `b`, `em` / `i`, `code`, `small` | Bold, italic, mono, small styled runs; nesting combines styles |
| `dl`, `dt`, `dd` | Definition list/items, `.TP`; multiple body paragraphs retained |
| `ul`, `ol`, `li` | Definition items with literal bullet or numbered tags; `ol start` supported |
| `pre` / `code` | Whitespace-preserving `.nf` / `.fi` |
| `a href` | HTTP, HTTPS, mailto or same-document fragment; `.UR` / `.UE` |

The HTML parser is `golang.org/x/net/html`. Browser repairs apply before semantic
mapping. Omitted or mismatched closing tags produce a conservative diagnostic;
strict mode can therefore reject otherwise browser-acceptable optional end tags.
`html`, `head`, `body`, `article`, `section` and `div` act as containers. Plain
`span` is transparent. A head `title` is a fallback when no `h1` exists; absent
metadata stays absent. A generic `h1` supplies title text, not styled title runs.
HTML list presentation becomes definition-item semantics; original list type and
browser numbering/layout are not recoverable.

The exported `article data-man-version="1"` profile retains all five `.TH`
metadata fields, subsection kind, indentation, tag widths, vertical/paragraph
spacing and manual-reference annotations. Recognized document headers and
permalink labels are display elements. Conversion notes become import diagnostics.
The importer recognizes the renderer's **exact** typed stylesheet template,
including bounded inline font data; it never loads or interprets those fonts.
Arbitrary stylesheet changes remain unsupported. Malicious nested content cannot
hide inside a forged header, permalink or conversion-notes block.

Tables, equations, scripts, images, iframes, external styles, unknown tags and
arbitrary CSS produce warnings and visible literal markup. Unsupported inline
markup stays literal in its paragraph. Event handlers and other unsupported
attributes are diagnosed and retained only in the original input snapshot.
Unsafe link targets are omitted while labels remain visible. Nothing executes or
fetches resources. Permissive fallback canonical roff retains literal text but
cannot retain an executable unsupported macro or HTML annotation. Its next import
therefore has plain/preformatted semantics, not the former unsupported marker.

A selection may contain inherited visual context that cannot recover its missing
ancestors; importing such a fragment warns. Import its containing section for
structural indentation. Links/references in preformatted blocks and annotations
inside headings/tags beyond their inline font styles warn when canonical roff
cannot retain them. Arbitrary nested headings/layout are outside this profile.

## API

```go
// Imports: "github.com/arran4/golang-gtroff/convert"
input, err := convert.ImportHTML(strings.NewReader(htmlSource), convert.Options{})
if err != nil {
    return err
}
// input.Source is the original HTML. Document.Source() is reserved for roff.
result, err := convert.WriteMan(writer, input.Document, convert.Options{Strict: true})
// Handle err and result.Diagnostics; warnings are observable even in permissive mode.
```

`HTMLToMan(reader, writer, options)` and `ManToHTML(reader, writer, options)`
provide the complete conversions. `ImportMan` keeps original roff in both
`Input.Source` and `Document.Source()`. `WriteMan` accepts a validated semantic
model without mutating it. `Options.HTML` supplies outbound `renderhtml.Options`,
including explicit font bytes; no API opens paths. The CLI opens only `-i` and
writes only `-o`. Named output uses the same source-alias protection and atomic
regular-file replacement as [HTML rendering](html.md).

Source, canonical roff and `Normalize(document)` are three separate views:

- Source bytes remain independent of DOM normalization and canonical formatting.
- Canonical roff centralizes argument quoting, backslash/quote escaping and
  neutral text-line prefixes. `.danger`, apostrophe controls and backslashes in
  input text cannot create requests. Canonical text uses controlled font changes
  and `\c` to preserve styled adjacency; it is intentionally verbose.
- `Normalize` retains metadata, block hierarchy, tags, visible text, styles,
  links/references, unsupported content/flags and spacing. It removes IDs,
  provenance and interpreter state. ASCII flow whitespace folds; NBSP and
  preformatted whitespace remain. Heading bold and preformatted mono are
  inherent. Hard breaks have no glyph style/link annotation; their count remains. Equivalent plain/`n`/`m`/`em` indents normalize to em; a zero-width
  indent is transparent. Nested item lists use such a neutral roff container.
  This comparison view is distinct from `man.Document.Canonical()`'s state-aware
  model JSON. It does not discard unsupported nodes to claim equivalence.

Strict conversion returns `ErrStrict` (or the outbound HTML renderer's strict
error) and leaves the writer untouched. Import functions return an inspectable
model alongside strict failures. Output construction/budget failures also leave
writers untouched; actual writer failures may leave partial streams. CLI strict,
resource and I/O failures preserve existing destination files. Exit codes follow
`render`: 0 success/warnings, 1 syntax/strict failure, 2 usage, 3 I/O/resource.

## Bounds and provenance

Default input is 8 MiB, output 32 MiB, 100000 nodes/tokens and 128 levels.
Positive limits can reduce these budgets; input can rise to 32 MiB. Output,
node and depth hard ceilings cannot be raised. HTML is bounded during tokenization
and after DOM repair; semantic nodes and writer depth are bounded too. Roff input
uses the input-byte budget as its per-line limit, so a long canonical paragraph
does not inherit the AST inspector's independent 64 KiB line default. AST and
semantic budgets still apply; HTML and roff node counts need not match.

HTML origins cover the input snapshot, since x/net/html does not provide source
positions. HTML diagnostics use offset `-1`, line/column `0` and the requested
source name. They do not pretend to identify exact HTML bytes. Roff origins keep
the parser's physical spans. Invalid UTF-8 is diagnosed and DOM repair is lossy;
the HTML snapshot remains available separately.

## Tests and examples

`testdata/convert/*.txtar` contains separate supported, nested, Unicode,
escaping, malformed, hostile and unsupported scenarios. Each has canonical roff,
HTML, normalized semantics and diagnostics. Supported cases compare the actual
`roff → model → HTML → model → roff → model` and `HTML → roff → HTML` loops.
Every case checks a stable second canonicalization; negative cases also require
explicit retained-content assertions. Normal tests do not modify expectations.

```sh
go test . -run '^TestConvertScenarios$' -update-convert
go test ./convert ./internal/cli ./cmd/gtroff
```

The complete checked-in `examples/man-tool.html`, including all embedded font
bytes, passes through the importer and canonical writer in tests. The small
[HTML input](../examples/convert.html) and [canonical man output](../examples/convert.1)
are also checked against the real CLI. These loops promise the documented
semantic subset, not original source spelling or pixel-identical typography.
