# Document reader and source preservation

`ParseDocument(io.Reader, ...Option)` parses line tokens and handwritten logical
blocks. `ParseLine` remains the single-line API; `ParseAST` provides an independent
structural tree. The document reader does not evaluate macros or read includes
unless a resolver is explicitly supplied.

`Document.Source()` contains the exact bytes read from that document's reader.
`SourceName` identifies that stream and `SourceComplete` is true only after its
reader reported EOF. These are source facts, not a claim that the syntax is valid.
On success with observed EOF, source includes original LF/CRLF, tabs, blank
lines, comments, Unicode/invalid UTF-8, unknown controls and a missing final LF.

Each `DocumentToken.RawSource` contains its exact consumed source range, including
line endings and complete multiline delimiters/bodies. `Token.String()` remains
typed canonical formatting. The compatibility `RawLine` field retains ordinary
line text without LF/CRLF and canonical formatting for successful multiline
tokens; use the new source fields for byte preservation. See the executable
[`ExampleDocument_Source`](../document_source_example_test.go).

Spans are half-open byte ranges. Line/column are one-based and columns count
Unicode runes (tabs count as one); an ending LF moves the end position to the
next line at column one. Invalid UTF-8 bytes remain intact and each advances a
replacement-rune column. Physical line units and logical block tokens remain
distinct: a definition/ignore/preprocessor token can span several lines. Roff
execution, nested custom grammar and registry block-consumption interfaces are
separate #7/#12/#14 work.

## Errors and partial results

Strict syntax failure stops tokenization and returns a non-nil partial document
plus the syntax error. Permissive syntax errors are recovered visibly with
diagnostics; legacy unknown controls remain unknown tokens. Terminal input or
resource failures always return a partial document **and an error**, in either
mode. Already-buffered but unparsed bytes are retained as an opaque source tail;
they are not silently discarded or presented as validated typed syntax.
Unread bytes still in the caller's reader are outside this snapshot. Thus a
partial document may contain a buffered prefix beyond its last parsed statement,
and `SourceComplete` can be false even if a small input happened to fit in that
prefix. Errors must be checked before consuming tokens as valid input.

`DocumentInputError` carries a source position and unwraps the reader/resource
error. `errors.Is` retains an original reader error; `errors.As` can identify
`DocumentInputError` or `DocumentLimitError`. Diagnostics contain source
positions independently of error strings. Line-length failures identify the
offending line's start; other input failures identify the consumed position.
Readers returning data and an error together have both preserved. Terminal
errors/EOF are cached so the underlying reader is never retried afterward;
repeated zero-byte/no-error reads stop with `io.ErrNoProgress`.

`cmd/lex` uses this same reader and now prints available partial tokens and
diagnostics before returning failure. Its `ParseFile` compatibility helper also
returns available tokens alongside an input error. Tests inject streams/writers
instead of replacing process stdout or parser flags.

## Bounds and explicit includes

Defaults are 64 KiB per physical line and 1,000,000 lines per source.
`WithMaxLineLength`/`WithMaxLines` configure those legacy limits (nonpositive
values disable the respective line limit). The line boundary is checked against
actual data; a file ending exactly at the configured count succeeds.

`WithDocumentBytes` adds an aggregate reader-byte budget across root and explicit
included sources: default 8 MiB, maximum 128 MiB, nonpositive values use default.
One extra byte may detect exhaustion and is preserved. Buffering uses bounded
chunks (at most 4096 bytes of read-ahead for a line-limit check), so oversized
lines do not first accumulate an unbounded string. A document-byte bound still
applies when line limits are disabled. The argument scanner's accumulation is
linear rather than repeatedly concatenating growing strings.

Existing include expansion of `Document.Tokens` remains compatible. Root source
bytes are separate from expanded tokens: `Document.Includes` stores each
directive span and child document, including partial failed children. Repeated
names do not overwrite an earlier source snapshot. Included readers retain the
[explicit ownership/containment contract](library.md#explicit-legacy-includes),
and child input/resource errors propagate in permissive mode too. The root input
reader remains caller-owned. The newer AST/CLI/export/server workflows never
execute includes.

TXTAR snapshots and independent source/position checks cover canonical-vs-raw
blocks, CRLF, EOF, Unicode, malformed source, exact/over-limit boundaries and
opaque tails. Additional tests cover reader fault identity, data+EOF, no-progress
readers, byte-by-byte UTF-8 chunks, include-wide budgets and child cleanup.
