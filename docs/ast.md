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
separate nodes. `.TP`/`.IP` remain typed lines, so paragraph semantics can be
added independently in #22. Ms macros use `WithPackage("ms")`, without treating
ms `.SH` as a man heading. Conventional explicit blocks include `.RS`/`.RE`,
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
Escapes, register references, and string references are not expanded. This is
not the semantic man model planned in #22.

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

## Parser dependencies

This implementation is stacked on unmerged PR #19 at `75d60db`. It does not
claim that issues #3–#17 are complete. The narrowly scoped parser repairs here
address #7's tab-separated arguments and #11/#12/#14's nil-token panic and lost
consumed source on unterminated blocks. Terminal block I/O/budget failures now
return their preserved partial document instead of continuing a failed read.

The separate #12 include-safety repair now confines the explicitly enabled legacy
resolver with `os.Root`, closes owned readers, preserves nested diagnostics and
bounds recursion for custom resolvers. See the [ownership and limit contracts](library.md#explicit-legacy-includes).

The separate #5 registry repair connects `WithRegistry` and `WithCustomHandler`
to document/AST parsing, with registration snapshots and explicit errors for
ambiguous or invalid handler results. [Custom handler contracts](library.md#custom-line-handlers)
describe precedence, concurrency and retained structural grammar.

Other parent-parser work remains separate: catalogue-wide custom implementation
validation (#5), full argument fidelity (#7), document streaming/fidelity and
consistent legacy spans (#11), bounded evaluation (#13), richer multiline/preprocessor grammar
(#12/#14), honest catalogue coverage (#15), and multi-output generation atomicity
(#16). AST source preservation is independent of legacy `DocumentToken.RawLine`, which still uses
canonical formatting for successfully parsed multiline blocks. Do not use that
legacy field as a byte-exact source archive. Multi-line heading interpretation
and fuller man paragraph semantics belong to #22; raw lines remain selectable.

## Scenario verification

Embedded `testdata/ast/*.txtar` files each contain `input.1`, `expected-ast.json`,
and `expected-extract.txt`. Optional `options.json` selects package, strictness,
first matching section, or removes the input's final LF. Optional `error.txt`
requires an error substring. The JSON golden uses one row per node. The text
golden includes a final LF because txtar represents text files; separate unit
tests assert exact extraction ending bytes. The CRLF case stores real CRLF.

Normal tests never rewrite fixtures. Explicit updates are package-local:

```sh
go test . -run '^TestASTScenarios$' -update-ast
go test ./...
go test . -run '^$' -fuzz '^FuzzParseAST$' -fuzztime=15s
```

The harness checks source/line cardinality, tree integrity, source positions and
determinism as well as goldens. API tests independently assert expected search,
extraction, Unicode positions, typed inspection isolation, traversal controls,
no includes/evaluation, limits, I/O failures, and realistic existing man fixtures.
