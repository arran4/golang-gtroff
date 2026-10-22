# Opt-in evaluation and resource limits

`Evaluate(tokens, env)` interprets the legacy evaluator profile described below.
`EvaluateContext(ctx, tokens, env)` adds cancellation. Parsing, AST construction,
text extraction, semantic conversion, rendering and serving do not enable it.
Keep the parsed `Document` or `AST` for source representation; evaluation returns
semantic output and does not promise a source round trip.

Use `NewEnvironment()` or a zero-value `Environment`. A nil environment selects
private default state. Reuse an environment to carry registers, strings and
macro definitions between calls. Do not mutate it or use it concurrently during
a call. Public setters copy macro body slices but do not enforce evaluation
budgets; evaluation validates caller-supplied state before interpreting tokens.
Caller-provided custom Go tokens and their metadata methods are trusted code.

See [the executable example](../evaluator_example_test.go) and
[TXTAR cases](../testdata/evaluation). These changes repair bounded execution and
error visibility under [#13](https://github.com/arran4/golang-gtroff/issues/13);
they do not complete that issue's composable/full core evaluator.

## Processing and language profile

Tokens execute in input order with the current shared environment. Definitions
and register changes affect subsequent tokens. Selected single-line branches
and invoked macro body lines are parsed with an empty registry, using generated
default-package syntax. Global custom handlers and include resolvers are not
consulted. These reparses use the existing argument parser; raw argument quoting
and escape interpretation remain parser contracts, not a new roff input grammar.

| Input | Evaluation behavior |
| --- | --- |
| `.nr name value [increment]` | Decimal Go `int` values after interpolation. A leading `+` or `-` changes the current register by that amount. Invalid values and overflow are errors. Nonzero increment metadata is stored; omitted/zero increments retain earlier metadata. |
| `.rr name` | Removes the register and its increment metadata. |
| `.ds name value`, `.as name value` | Store/append literal parsed string values; append is bounded before concatenation. |
| `.de name` block | Copies raw body lines into the macro table; a new definition replaces the earlier definition. |
| Unknown control matching a defined macro, without arguments | Reparse/evaluate body lines in order using current state. Calls with arguments and names colliding with generated controls are preserved with diagnostics. |
| `.if`, `.ie`, `.el` | Single-line branches. Decimal integers are true when **positive**, `n` is true and `t` is false. Leading `!` negates; integer comparisons support `>`, `<`, `==`. String comparison uses `'left'right'` with interpolation; the historical missing-final-quote shorthand is retained. Each executed `.ie` pushes a result; `.el` consumes the most recent result across the current call, including nested branches/macros. An unpaired/repeated `.el` errors. Pairing does not carry across calls. |
| Text | One interpolation pass; other escapes remain literal. |
| Comments, empty controls | Pass through unchanged. |
| Other tokens, including `.am`, `.als`, `.rm`, formatting requests, preprocessors, `.so`, `.sy`, device/filesystem requests and unknown controls | Pass through unchanged, with `Environment.Diagnostics`. They perform no evaluation action. |

Disabled branches are not reparsed, interpolated or executed. Selected multiline
conditional blocks (`\{`) return an explicit unsupported error. Arithmetic
expressions such as `2+3`, scaling units, loops, macro argument substitution,
request shadowing, aliases, append/remove macro operations and general groff
copy mode are not implemented. They must not be inferred from generated token
or catalogue coverage. In particular, GNU roff arithmetic evaluates left to
right; extending this profile requires its grammar, not Go operator precedence
([GNU groff manual](https://www.gnu.org/software/groff/manual/groff.pdf)).

Text interpolation recognizes `\*[name]`, `\*(na`, `\*n` and corresponding `\n`
register references, including Unicode names. Undefined strings remain as their
original escape; undefined registers become `0`. Inserted string values are
opaque, so a string referring to itself or another string is not expanded
recursively. Plain register references do not increment. Increment/decrement
escapes (`\n+`, `\n-`) are explicitly unsupported errors. Malformed names report
errors. Other escapes and invalid UTF-8 bytes are preserved, without evaluation.
Typed formatting-token arguments are not interpolated.

## Budgets and cancellation

Values <= 0 select defaults. Values above the ceiling select the ceiling; no
setting disables a budget. Byte limits are bytes, not runes.

| Environment field | Default | Ceiling | Accounting |
| --- | ---: | ---: | --- |
| `MaxDepth` | 50 | 128 | Active nested macro calls, body-line reparses and selected conditional branches. |
| `MaxSteps` | 50,000 | 1,000,000 | State validation, token dispatch, line parsing, interpolation segments, unary condition operators and other work checkpoints. |
| `MaxWorkBytes` | 32 MiB | 256 MiB | Strings scanned/copied for interpolation, reparsing, string assignment/append and state-name bookkeeping. Repeated processing is charged again. |
| `MaxOutputBytes` | 8 MiB | 128 MiB | Sum of emitted `TokenText` bytes, plus a separate bound of the same size on each temporary interpolation. |
| `MaxTokens` | 50,000 | 1,000,000 | All emitted tokens, including borrowed/preserved controls and comments. |
| `MaxStateBytes` | 8 MiB | 128 MiB | Names and stored values/body lines, plus 32 per map entry and 16 per macro body line. Registers and increment entries each count separately. |
| `MaxLineBytes` | 64 KiB | 1 MiB | Each selected branch or macro-body line before reparsing. |

The byte budget for output does not serialize or copy preserved tokens. Their
existing fields are borrowed from the input; token count bounds the output
slice. A caller serializing them needs its own writer limit. State accounting
is a deterministic size model, not a byte-exact bound on Go heap use. Existing
input/caller allocations are outside these budgets. Initial state validation
can itself exhaust a budget. Resource counters are deliberately conservative.

`StepCount` accumulates across calls on an environment, including initial state
validation. Reset it explicitly to start a new work allowance. Other counters
and conditional pairing start fresh per call. `Depth` is observational and is
reset to zero on exit. Cancellation uses `context.Context.Err()` at work
checkpoints, including within interpolation; it cannot interrupt arbitrary
caller-supplied Go methods. No evaluator request executes shell commands, reads
files, fetches resources or invokes include resolvers.

## Errors, partial output and diagnostics

On failure, `Evaluate` and `EvaluateContext` return the already emitted prefix
and an error. State changes applied before the failure remain; evaluation is
not transactional. No parser error from a selected branch or macro body is
ignored. Original input tokens, source bytes and macro definition bodies are
not modified. Passthrough output tokens share identity with their input tokens.

`EvaluationError` wraps the original cause, an outer zero-based token index and
an expansion path. Initialization errors can have index `-1`; cancellation
before evaluation may return the context error directly. Use `errors.Is` for
cancellation/parser causes and `errors.As` for `EvaluationLimitError`. The path
identifies macro body lines; it does not claim original document source spans.
`Environment.Diagnostics` is reset on every call and records preserved tokens
with the same index/path convention. Inspect it even when the error is nil.

`env.ExpandTextContext(ctx, text)` returns bounded interpolated text or an error,
sharing the cumulative step allowance. The compatibility `env.ExpandText(text)`
returns the original string on failure and records an error diagnostic. Use the
context variant when failures must be handled directly.

Refresh TXTAR expectations only with explicit approval of the new output:

```sh
go test . -run '^TestEvaluationScenarios$' -update-evaluation
go test . -run '^(TestEvaluation|TestEvaluate)'
```
