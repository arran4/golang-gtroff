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

## Arity and remaining #7 work

Ordinary generated parsers with zero argument fields now reject extra values
through both compatibility matchers. This includes explicit empty arguments:
`.nf ""` has one argument. `ErrUnexpectedArguments`/
`UnexpectedArgumentsError` retain count information through generated wrappers.
The catalogue route test executes this rejection for all 85 generated zero-field
routes in the current source. Comments do not count as arguments.

`ParseDocument` preserves an invalid request as an unknown control and records a
diagnostic in permissive mode. Strict mode returns the partial document and an
error. Neither mode discards its raw source. Special multiline document handlers
retain their separate header/body grammar.

This is part of [#7](https://github.com/arran4/golang-gtroff/issues/7).
**Nonzero legacy matchers still join excess values into an end field**. Full
catalogue fixed/optional/repeating/raw-rest binding declarations and serializers,
special request grammars and broader fidelity remain unfinished. In particular,
this change does not claim that arbitrary `.TH`/`.IP` extras are validated or
that legacy typed fields retain original argument boundaries. The generated
syntax report remains separate from these grammar contracts.

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
