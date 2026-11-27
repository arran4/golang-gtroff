# Source-aware AST

`ParseAST(io.Reader, ...Option)` creates a source snapshot for inspection and
selection. It shares generated line-token types with `ParseLine`, and inline
syntax with `ParseInline`. `ParseDocument`, `Document.Tokens`, the existing `Node`
metadata interface, and generated APIs remain available. This API does not
execute macros, expand references, fetch files, or resolve includes, even when
`WithIncludeResolver` is supplied.

## Structure and identity

The root contains physical lines, section/subsection groups, and block groups.
A line contains inline syntax spans. A line's `Attributes` expose its normalized
control name, token kind, package, and argument lexemes. `Token()` returns a
caller-owned copy of its concrete generated token for type switches. Opaque
lines and delimiter lines have no generated token. `Inline()` returns a copied
inline token. Group attributes expose headings and explicit block terminators.

Man `.SH`/`.SS` headings with arguments form groups; repeated headings are
separate nodes. `.TP`/`.IP` remain typed lines; the separately selected
[`man.Build` hierarchy](man.md#optional-hierarchy-and-source-navigation) supplies
definition-list/item/paragraph semantics. Ms macros use `WithPackage("ms")`,
without treating ms `.SH` as a man heading. Conventional explicit blocks include `.RS`/`.RE`,
`.nf`/`.fi`, `.EX`/`.EE`, ms displays/keeps/footnotes, definitions, ignores, and
preprocessors. Macro definitions, ignores, `.EX`, and preprocessor bodies stay
opaque. Within definitions, an explicit distinct terminator permits a nested
`.de`/`.am`/`.ig` syntax group; an opener using the enclosing terminator remains
literal body content. These are syntax boundaries, not roff execution semantics.
Headings or controls inside opaque bodies do not become live sections.

Nodes have a one-based `NodeID` in deterministic preorder. `AST.Node(id)` is an
O(1) lookup. IDs are stable for identical source and options, not across edits or
between documents. `Path()` is a slash-separated zero-based child index path;
the root is `/`. Parent and sibling lookups are O(1). Child/ancestor/descendant
accessors return copies, and ancestors start at the nearest parent. The tree
has no public mutation methods; copying accessor results or changing inspected
tokens cannot introduce cycles, reparent nodes, or change the source snapshot.

`Walk` is iterative preorder with `WalkContinue`, `WalkPrune`, and `WalkStop`.
`Find`, `FindKind`, `FindMacro`, `FindSection`, `FindText`, and `FindSpan` search a
selected subtree including its root, in document order. Searches are linear in
the selected subtree; walking and indexing never repeatedly scan all siblings.
Normalized searches fold whitespace and lowercase Unicode; they do not perform
Unicode NFC normalization. Content searches match line/inline contributions,
not duplicated container text; use `Find` with `ASTLine` for one hit per line.

## Inspection vocabulary

| Syntax kind | Meaning | Common source |
| --- | --- | --- |
| `document` | Entire captured snapshot | Includes a partial or opaque recovery tail |
| `section` | Man heading and following syntax | `.SH NAME` |
| `subsection` | Heading within a section, or at root | `.SS Details` |
| `block` | Explicit bounded source region | `.RS`/`.RE`, `.de`/`..`, `.TS`/`.TE` |
| `line` | Physical source line (or recovery remainder) | Text, request, comment, blank line, delimiter |
| `inline` | Inline syntax inside its owning line | Font/escape/reference syntax |

Node kinds identify structure; `Attributes().Name` identifies a control,
`Package` identifies its registered macro package, and `TokenKind` distinguishes
request, macro, text, comment and other parser routes. An unknown control remains
a named line with diagnostics; a known but unsupported control can still have a
typed token. Neither token recognition nor an opaque block means successful
semantic interpretation or rendering. `Opaque` marks source kept without live
interpretation; `EndMarker`, `UnresolvedEnd` and `Closed` describe boundary facts,
not execution. Use `BlockSource()` for a caller-owned capture with exact header,
body, tail and terminator positions.

`.TH` describes man metadata; `.SH`/`.SS` introduce headings; `.PP`/`.P`/`.LP`
separate paragraphs; `.TP`/`.TQ`/`.IP` introduce tagged definitions; `.B`/`.I`
and alternating font macros style text. `.RS`/`.RE` bound indentation. `.so`/`.mso`
are include instructions retained without loading. `.de`/`.am`/`.ig` and
preprocessor regions retain raw captures. Consult the [semantic vocabulary and
supported subset](man.md) before interpreting these controls as content.

The [external inspection example](../ast_inspection_example_test.go) starts from
the root, locates ENVIRONMENT, enumerates `.TP` openers, inspects a concrete
`TokenManB`, maps a span back to source bytes, and visits an opaque definition and
unknown macro. Use `Token()` with a type switch for typed fields; token copies
cannot mutate source. `NodeID` and `man.ID` are separate document-local identity
spaces. `Origin.ASTIDs` and `man.Document.SourceNodes` connect them without
reparenting syntax nodes. The syntax tree already groups explicit boundaries;
the semantic hierarchy additionally groups paragraphs and definitions.

The existing flat AST JSON table and semantic model version 1 are unchanged.
These pre-release APIs have no promised cross-version stability beyond their
documented contracts. Extend line tokens through the existing registry/custom
handlers and bounded captures; add task-oriented helpers separately instead of
inventing new kinds in the parser. Query selectors, feature classification and
policy lint remain separate workstreams.

## Source, extraction and JSON

`AST.Source()` and `ASTNode.Raw()` return original bytes, including unknown
controls, malformed syntax, tabs, CRLF, blank lines, invalid UTF-8, and a missing
final newline. Raw source is independent of the token's canonical `String()`.
Spans are half-open byte offsets into that source. Lines and groups include
line endings; inline spans cover the line body. Positions are one-based lines
and Unicode rune columns; tabs occupy one column and LF advances to column one
on the following line. Invalid UTF-8 bytes each advance one replacement-rune
column while remaining byte-exact in `Source`/`Raw`.

`Text()` performs syntax extraction: known control lines contribute unquoted
argument values, comments and block delimiters contribute nothing, ordinary
text and opaque/unknown/malformed lines contribute literal text. Existing line
endings are retained on contributions. Container extraction visits each physical
line exactly once, including section headings, without repeating inline content.
Escapes, register references, and string references are not expanded. For decoded
display text and paragraph/definition structure, use the separate
[semantic man model](man.md).

`json.Marshal(tree)` produces a deterministic flat preorder node table containing
IDs, parent/child links, attributes, spans, token fields, inline fields, and
diagnostics. It avoids recursive JSON nesting. JSON follows Go's normal invalid
UTF-8 replacement rules; use `Source` for byte-exact serialization.

## Errors and bounds

Permissive parsing returns a complete tree with diagnostics for malformed syntax
and warnings for unknown controls. `WithStrict(true)` also returns a non-nil
error for erroneous/unknown syntax, together with the preserved tree. It does
not stop at the first bad line. Incomplete inline references, unmatched explicit
block delimiters, and unterminated blocks are diagnosed. Unsupported controls
are preserved without being evaluated.

I/O errors and exhausted budgets always return an error, including in permissive
mode. Defaults: 8 MiB of source, 100,000 physical lines, 64 KiB per line, 128
nested blocks, and 100,000 nodes. Configure source/depth/node limits with
`WithASTBytes`, `WithASTDepth`, `WithASTNodes`; configure line limits with the
existing `WithMaxLines`/`WithMaxLineLength`. A byte limit may read one extra
sentinel byte. All bytes actually read remain accessible. A line/node budget
may collapse remaining source into one opaque tail line; inline budget exhaustion
retains the line without child inline nodes. One recovery node can exceed the
node budget. A line/depth failure preserves syntax as opaque input and returns
an error. Callers must check errors before treating a tree as validated.

### Reader failures and EOF facts

`SourceComplete()` records whether the underlying reader actually returned EOF.
It is independent of valid syntax and successful tree construction. A source
byte limit retains its extra detection byte but does not claim EOF unless the
reader reported it. A reader returning data and EOF together is handled without
losing those bytes. See the [executable partial-reader example](../ast_input_example_test.go).
The existing AST JSON shape is unchanged; this fact is available through the
library method.

`ASTInputError` carries the source name and the line/column/byte position after
the validated input prefix. It unwraps original reader errors. A source-byte
failure also exposes `ASTSourceLimitError{Limit}` through `errors.As`. If a byte
or tree budget fails alongside I/O, the original I/O cause remains accessible
through `errors.Is`; syntax diagnostics remain separate tree data. Input failures
return a navigable non-nil partial AST in both strict and permissive modes.

Nil and typed-nil readers return errors before their methods run. A shared
reader guard stops after 100 consecutive nonempty-buffer reads returning zero
bytes and no error, with `io.ErrNoProgress`; actual data resets that count.
It caches EOF/errors, so a terminal reader is never retried. Negative counts or
counts exceeding the supplied buffer produce `ErrInvalidReadCount` /
`ReaderCountError` before any slice operation. That call's buffer is not trusted;
only earlier validated bytes are exposed. The error retains an accompanying
reader cause. These checks do not interrupt a blocked `Read` or recover a panic
raised inside caller-supplied reader code; callers own I/O deadlines and reader
implementations.

See [the document reader](document.md), [custom handlers](library.md#custom-line-handlers) and [block capture](blocks.md) for related source-preservation contracts.
