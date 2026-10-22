# Bounded block capture

`ParseDocument` captures `.de`, `.am`, `.de1`, `.dei`, `.dei1`, `.am1`,
`.ami`, `.ami1`, `.ig`, `.TS`/`.TE`, `.EQ`/`.EN`,
`.PS`/`.PE` (also `.PF`) and `.[`/`.]` as logical source blocks. Registrations
come from `Block` declarations in `cmd/gen/in.json`; capture and token factories
are handwritten. Parsing does not execute definitions, ending macros, includes
inside a block, preprocessors or external programs.

For every captured block, `DocumentToken.Block` exposes a `BlockSource` with an
exact header, physical body lines, outer terminator, declared rule and spans.
`Source()` reproduces the captured bytes, including CRLF, missing final LF,
quoted header spelling, terminator arguments, unknown requests and invalid UTF-8.
`BodySource()` excludes header/outer terminator. On terminal input/resource
failure, `Tail` separately retains buffered but unparsed bytes; `Source()` includes
it, while `BodySource()` does not classify it as body syntax. `Closed` records an
observed outer terminator; it does not imply the block was valid.

The older token fields still contain decoded header names and body line text
without LF/CRLF. Their `String()` output is canonical compatibility formatting,
not an exact source writer: it can omit terminator arguments or use `.PE` for an
input `.PF`. Keep `RawSource` or `Block.Source()` for exact input. See the
[executable custom-handler example](../block_example_test.go).

## Delimiters, errors and recovery

Definition/append headers require a name and allow one end name; ignore headers
allow one end name. These headers use checked argument binding, so quoted names
and terminators work, comments terminate arguments, and surplus arguments error.
Empty macro names and names containing whitespace/NUL are invalid. The default
terminator is `..`; an end name `END` selects `.END`. Invalid headers preserve
only their own line and consume no following body. In strict document mode,
already-buffered following bytes remain an opaque tail, as with other syntax
errors.

Definition and ignore capture is flat: the first matching end closes it. A body
line that looks like `.de` is data, not an executed definition. Preprocessor
capture recognizes nested pairs declared in the generated rules; nested
content/delimiters stay in the outer raw body and dispatch no callbacks. These
are lexical region rules, not implementations of tbl/eqn/pic/refer languages.
The existing dot/apostrophe normalization is retained for structural delimiters;
it is more permissive than GNU preprocessors' dot-only input convention.

A mismatched known closing delimiter is retained and diagnosed. If it closes an
outer pair, capture reports the unclosed inner pair and recovers at that outer
boundary. Otherwise capture continues to the expected outer terminator. An
unterminated block retains its entire consumed source without inventing an end.
Standalone built-in closing delimiters also produce errors. `BlockSyntaxError`
identifies the header/offending delimiter position, unwraps to `ErrInvalidBlock`
and retains a header/callback cause through strict document error wrappers.
Permissive document parsing keeps the invalid capture as an unknown token with
its source metadata and diagnostic; strict mode additionally returns an error.

Existing document byte, line and line-length limits also bound capture.
`WithMaxBlockDepth` bounds declared nested pairs (default/maximum 128).
Input/resource failures return partial source and an error in both modes.

## Definition variants and indirect boundaries

The six variant requests produce `TokenMacroDefinitionVariant` instead of
masquerading as ordinary `.de`/`.am` blocks. `Request` retains the variant name;
`Name` and `End` retain decoded header values; `Delimiter` records the actual
closing command. Body lines remain opaque. Existing generated
`TokenRequestDe1`/`TokenRequestDei`/etc. are still the line-only API and retain
existing fields. The ordinary macro block types keep their layouts.

GNU distinguishes compatibility-mode and indirect definition behavior; see
[Writing Macros](https://www.gnu.org/software/groff/manual/groff.html.node/Writing-Macros.html).
Structural parsing does not execute those behaviors. In particular, an indirect
`End` value is a **string reference**, so treating it as a literal end name can
close at the wrong line. With no explicit end reference, the default `..` boundary
is known and capture proceeds without resolution. Name references stay symbolic;
resolving the target macro name is not needed to find the body boundary.

Use `WithBlockNameResolver` with a `BlockNameResolver` or
`BlockNameResolverFunc` when the end reference is present. The callback receives
the exact header context and reference and returns the final name without its
control prefix. It is called once for an encountered indirect opener, never for
direct/default ends, line overrides or headers inside an opaque definition body.
An explicit empty result selects `..`; results over 64 KiB, invalid UTF-8,
ASCII whitespace or NUL are rejected. A result is treated as a literal name, not
interpolated again. See the [executable example](../definition_variant_example_test.go).

There is no default resolver and parsing does not execute preceding `.ds`/`.as`
requests or consult evaluator state. A missing, typed-nil, failing or invalid
resolver leaves the delimiter unresolved. The parser retains the entire bounded
remainder as one opaque block, including plausible `.REF`, `.STOP` and `..`
lines; it never guesses where top-level syntax resumes. Both document modes
record the diagnostic; strict mode additionally returns an error after capture.
An unresolved `BlockSource.Rule.UnresolvedEnd` and AST `UnresolvedEnd` attribute
identify the reference. `Closed` remains false. Reader/byte/line/node limits
still apply and retain partial source. `BlockDelimiterError` unwraps to
`ErrUnresolvedBlockDelimiter` and retains a resolver cause in document errors.
The shared block contract also allows an explicit `UnresolvedEnd` rule or a
`BlockDelimiterError` from `Begin` for other context-dependent grammars.

Registration snapshots and explicit includes retain the supplied resolver. It is
trusted Go code: protect shared state and bound its own work/allocation. This
option does not enable includes, preprocessor launches or evaluation. Its callback
is not stored in source snapshots. Keep the same explicit state when reparsing
canonical indirect syntax; `String()` cannot make an indirect boundary independent
of the state in which it is read.

Opt-in evaluation currently preserves complete variant tokens and emits an
unsupported-semantics diagnostic. It neither executes their body requests nor
silently treats their compatibility/indirection behavior as ordinary definitions.
Compatibility-mode execution, indirect name evaluation, copy mode and ending
macro execution remain separate evaluator work.

## Custom handlers

Register a `BlockHandler` with `Registry.RegisterBlock` and select that registry
with `WithRegistry`, or add one parse-local handler with `WithBlockHandler`.
The interface has three operations:

1. `CanHandle(cmd, pkg)` selects a control header. Like line handlers, it must be
   a pure predicate and receives the original dot/apostrophe command spelling.
2. `Begin(BlockHeader)` validates the header and returns `BlockRule`: alternative
   closing commands and optional nested opener/end pairs. Rules use dot-normalized
   command names, including the leading dot. Invalid/duplicate markers fail before
   a body is consumed. Closing matches take precedence over nested openers.
3. `ParseBlock(BlockSource)` receives one complete, valid, bounded capture and
   returns a token. It runs once for the outer block; nested/body lines do not
   dispatch line/block handlers. No reader or include resolver is exposed.

Custom line and block matches are checked together in document/AST context.
Exactly one custom match overrides built-in opening structure; overlapping
matches fail with `ErrAmbiguousHandler` before `Begin`/`Parse`. A line override
opts out of block capture for that header. Nil registrations/results fail with
`ErrInvalidHandler`; handler errors never fall back to generated parsing.
`ParseLine` remains line-only and does not invoke block handlers.

Registrations are snapshotted once at parse entry and inherited by explicit
includes. Later registrations affect later parses. Slice metadata passed to a
callback is copied; mutations cannot change captured source. Handler objects are
shared trusted Go code: protect state when used concurrently and bound your own
computation/allocation. Registration is explicit opt-in; this is not a sandbox
for untrusted plugins or an external preprocessor execution facility.

## AST inspection

`ParseAST` uses the same registrations, validated header rules and preprocessor
capture machine. `ASTNode.BlockSource()` returns an independent source/metadata
copy for captured block nodes. Physical body metadata is derived lazily from the
immutable source so deeply nested structural definitions do not retain repeated
body-line slices. A successful custom block token is available through the block
node's copied `Token()` snapshot; its header has a `TokenBlockHeader`. Custom
AST tokens must support JSON round trips, as with custom line tokens.

AST syntax recovery still builds a complete preserved tree and reports strict
errors after construction. Existing AST structural nesting of definitions with
distinct terminators is an inspection view; it does not execute copy-mode nesting
and is separate from flat document definition capture. Other AST display/list
blocks retain their existing structural rules. AST byte/node/line/depth controls
also apply, including line-length checks inside opaque payloads.

This implements capture/registration work under #5/#12/#14. It does not provide
preprocessor rendering, arbitrary parser callbacks consuming a reader, roff
copy-mode evaluation or execution of a macro named by an end delimiter.
Catalogue syntax coverage does not complete the broader #12/#13 execution scope.

Nineteen small TXTAR scenarios cover exact source, headers, all built-in families,
nesting, mismatches, strict recovery, custom capture and bounds. Separate tests
exercise callback/registration failures, snapshots, concurrency, include
inheritance, reader faults, unresolved boundaries and mutation isolation. A
separate evaluation scenario verifies that variant bodies do not execute. Fixtures
refresh only with:

```sh
go test . -run '^TestBlockScenarios$' -update-blocks
go test ./...
go run ./cmd/gen -check
```
