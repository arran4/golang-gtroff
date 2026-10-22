# Argument source, decoded values and checked formatting

`ParseArguments(source)` returns an `ArgumentList` containing the original
argument string, decoded lexemes and byte spans. It returns a non-nil snapshot
on a malformed quote, so callers can inspect successful arguments and the exact
unparsed tail. It does not evaluate roff escapes, definitions or includes.
See the [executable example](../arguments_example_test.go).

```go
arguments, err := groff.ParseArguments(`  café\ mocha "世界" \" comment`)
// arguments.Source() retains the complete string, even if err is non-nil.
lexemes := arguments.Lexemes()
spans := arguments.ByteSpans()
// lexemes[0].Value == "café mocha"
// spans[0] == groff.ArgumentSpan{Start: 2, End: 14}       // bytes
// lexemes[0].Start == 2 && lexemes[0].End == 13          // runes
_ = err
_ = lexemes
_ = spans
```

`Source()` includes original whitespace, quotes, escapes, comments and malformed
or invalid UTF-8 bytes. `ArgumentLexeme.Raw` is the original byte slice for that
argument, not a reconstruction from runes. Invalid UTF-8 and NUL bytes are also
retained in decoded values; this scanner does not validate encoding. The AST's
separate invalid-UTF-8 diagnostic/opaque-source policy is unchanged.

`Lexemes()` and `ByteSpans()` return copies with matching indices. Spans are
half-open offsets relative to the supplied argument string, not document source
positions. `ArgumentLexeme.Start`/`End` retain the older **rune** index contract.
`Stop()` returns byte and rune offsets at the start of a comment or at EOF.
`Comment()` identifies the entire comment suffix; `Unparsed()` identifies the
incomplete argument after a syntax error. Gaps between spans remain in Source.
Changing returned slices cannot modify the snapshot.

`ArgumentSyntaxError` provides both byte and rune ranges and unwraps to
`ErrUnclosedQuote`. Successfully scanned arguments remain available through
`ParseArguments`; compatibility wrappers continue returning nil result slices on
syntax errors. `ArgumentParser` also retains its legacy error text and rune stop
index, with the richer error accessible through `errors.As`.

## Lexical profile

All three APIs use one scanner. It preserves the existing package profile:

- ASCII spaces and tabs separate arguments unless quoted or escaped.
- A double quote opens quoting only at the start of an argument. An empty quoted
  argument is retained. Doubled quotes within a quoted argument produce one
  literal quote. A closing quote ends the argument even if another character
  follows without whitespace: `"a"b` produces two arguments.
- A backslash followed by a space produces a literal space. Within quotes,
  backslash-quote produces a literal quote. Other roff escapes, including doubled
  backslashes, stay literal; interpolation belongs to opt-in evaluation.
- An unquoted backslash-quote starts a comment through the end of the provided
  string. It is a literal quote escape inside a quoted argument. Other comment
  or delimiter conventions are not inferred by this lexer.
- A lone/trailing backslash outside quotes is literal. An unclosed quote errors
  without losing the source or the successful prefix.

This profile is not shell syntax or the complete request-specific roff grammar.
The string input is already caller-owned; scanning is linear and allocates in
proportion to decoded values and requested metadata. The typed-value wrapper
avoids raw-lexeme/span allocation. Document and AST byte/line budgets remain
separate input controls.

## Checked canonical arguments

`CanonicalArguments(values)` returns one-space-separated encoded arguments or
`ErrArgumentEncoding`. `arguments.Canonical()` uses the same operation on the
snapshot's decoded values. Comments and original whitespace/quote choices are
intentionally omitted. Re-lexing successful output preserves the complete value
list and argument boundaries; it is not an exact source or whole-document writer.

The current lexical profile cannot represent every constructed string/list.
The formatter checks each candidate and the **joined list**, since a trailing
backslash followed by a separator can merge two arguments. It can use unquoted
escaped spaces for representable values that cannot close a quoted argument.
Physical CR/LF and ambiguous escape/boundary cases return
`ArgumentEncodingError`, whose `Index` identifies the first affected value.
For example, a single trailing backslash can be the final argument, but placing
another argument after it is rejected. Some literal backslash-space or
backslash-quote combinations are also unrepresentable by this profile.

On malformed input, `arguments.Canonical()` returns the original syntax cause;
it never silently serializes only the successful prefix. Use `Source()` for
exact retained input. The old `Quote` and generated `Token.String()` APIs retain
their existing formatting behavior and do not acquire a universal round-trip
promise. Representative constructed `.TH`, `.IP` and `.Pn` field round trips are
covered separately from raw-source preservation.

## Declared typed-field binding

Generated line parsers now use `BindArguments` with an `ArgumentGrammar` compiled
from `cmd/gen/in.json`. Omitted policy means fixed arity from the effective
catalogue signature. Fixed forms reject surplus arguments, including explicit
empty arguments: `.TH T 1 a b c extra`, `.IP tag 4 extra` and `.nf ""` fail.
`ErrMissingArguments`, `ErrUnexpectedArguments` and `ErrInvalidArguments` remain
accessible through generated error wrappers. Missing-argument diagnostics now
include actual/minimum counts; ordinary unclosed-quote diagnostics retain their
legacy text and typed cause.

Each generated token exposes `ArgumentGrammar()` as fresh metadata. It describes
field optionalness, binding direction, named literal flags, numeric-expression
slots and explicit aggregate policy. The generated syntax report adds
`argument_mode`, `minimum_arguments` and `maximum_arguments` (`-1` means no fixed
maximum). Counts describe grammar units, which can contain spaces, rather than
blindly counting whitespace-separated words. These are binding policies, not
claims that the full GNU language or evaluator is implemented.

| Policy | Typed projection |
| --- | --- |
| `fixed` | One argument per field, reserving required positions. `.Pn` explicitly retains its historical right binding. `.NT` recognizes literal `C`; `.VS` recognizes literal `4`. |
| `joined` | The declared final field joins remaining decoded arguments with spaces, as with man text/heading macros. |
| `variadic` | Declared minimum, prefix and repeated group size; only the named final field joins the remainder. `.hcode` and `.hpfcode` require complete pairs. |
| `raw` | Required prefix fields plus an uninterpreted tail, including its quotes and comments. |
| `string` | Prefix plus string contents, stripping one neutral opening quote and an unescaped comment suffix; trailing spaces remain data. |
| `register` | Name, one numeric expression, optional increment expression. |
| `conditional` | Predicate plus raw body; numeric parentheses and single-quote comparisons retain internal whitespace. `.while` requires a body. |
| `delimited` | Three fields separated and surrounded by one delimiter, retained together in the existing scalar field (`.tl`, ms page headers/footers). |

Numeric slots group balanced parentheses and skip escape/reference spellings,
including parentheses in bracketed reference names. They do not evaluate values,
validate units or prove arithmetic validity. For example, `.nr X (3 + 5)*4 2`
now binds `Value="(3 + 5)*4"` and `Incr="2"`; evaluation remains opt-in.

The string policy follows the request-specific neutral-quote convention rather
than ordinary argument quoting: `.ds S "  unfinished string` is valid and retains
the two leading spaces. See GNU's [string request grammar](https://www.gnu.org/software/groff/manual/groff.html.node/Strings.html).
String contents are retained before evaluation/copy-mode expansion; this binder
does not execute interpolation or the full copy-mode language.

`ParseDocument` preserves an invalid request as an unknown control and records a
diagnostic in permissive mode. Strict mode returns the partial document and an
error. Both retain its raw source. Special multiline document handlers retain
their separate header/body grammar; these declarations apply to generated line
parsers.

## Checked typed-token formatting and limits

`CanonicalToken(token)` and generated `token.CanonicalString()` format current
fields through their declared policy, then bind the result and check field
equality. They return `ErrArgumentEncoding` when data cannot be encoded without
changing fields. Raw bodies are emitted as syntax rather than quoted as one
ordinary argument. A string containing a comment introducer that would discard
part of the field is rejected. Physical line endings are rejected. Required
empty ordinary fields, positional optional gaps and absent literal flags receive
distinct handling. See the [executable example](../arguments_example_test.go).

Successful checked output promises equality of the existing typed fields, not
original argument boundaries or original source bytes. In particular, alternating
font macros, repeated lists, `.ta`, `.sizes` and `.NH` retain their scalar/joined
projections. Repetition counts do not fully validate tab-stop flags, size-list
terminators or hierarchical heading semantics. Rich conditional tests with
separate operands and arbitrary comparison delimiters remain future grammar
work. The older one-word comparison shorthand without its final single quote
remains accepted for compatibility and can be ambiguous if a later body contains
a single quote. Use the complete three-delimiter form.

The legacy `ForwardMatcher`, `BackwardMatcher`, `Quote` and
`String()` remain compatibility APIs. Nonzero legacy matchers still join an end
field, but generated parsers no longer use them. `String()` still uses legacy
formatting and has no universal round-trip guarantee; use checked formatting
when typed-field preservation matters, and `Source()`/document source snapshots
when exact source spelling matters.

This is a follow-up for [#7](https://github.com/arran4/golang-gtroff/issues/7),
not closure of its full grammar/fidelity scope. Seventeen binding TXTAR cases
check expected fields independently of serialization. Invalid syntax/arity is
checked separately in both document modes. Every generated dispatch example is
checked for canonical field equality; all 271 fixed catalogue routes reject
surplus fields. Generator fixtures reject malformed and conflicting policies.

Seven small argument TXTAR scenarios cover bytes, indices, quoting, comments,
invalid bytes and partial errors; a document scenario verifies arity recovery.
Raw/gap/tail reconstruction, legacy-wrapper agreement, checked formatting and
malformed cases have independent assertions and fuzz targets. Normal tests never
rewrite expectations. Explicit refresh commands are:

```sh
go test . -run '^TestArgumentSourceScenarios$' -update-arguments
go test . -run '^TestDocumentSourceScenarios$' -update-document
go test ./...
```

`input.args` fixture files encode one argument string plus the archive's final
LF, which the harness removes once; `input.hex` represents exact arbitrary bytes.
