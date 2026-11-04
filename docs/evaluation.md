# Opt-in evaluation and resource limits

`Evaluate(tokens, env)` interprets the integer evaluator profile described below.
`EvaluateContext(ctx, tokens, env)` adds cancellation. Parsing, AST construction,
text extraction, semantic conversion, rendering and serving do not enable it.
Keep the parsed `Document` or `AST` for source representation; evaluation returns
semantic output and does not promise a source round trip.

Use `NewEnvironment()` or a zero-value `Environment`. A nil environment selects
private default state. Reuse an environment to carry registers, strings,
macro definitions and aliases between calls. Do not mutate it or use it
concurrently during a call. Public setters copy macro body slices but do not enforce evaluation
budgets; evaluation validates caller-supplied state before interpreting tokens.
Caller-provided custom Go tokens and their metadata methods are trusted code.

See [the executable example](../evaluator_example_test.go) and
[TXTAR cases](../testdata/evaluation). These changes repair bounded execution and
error visibility under [#13](https://github.com/arran4/golang-gtroff/issues/13);
the supported core includes composable state, aliases and positional macro arguments.
This is a documented subset, not a full roff interpreter or typesetter.

## Processing and language profile

Tokens execute in input order with the current shared environment. Definitions
and register changes affect subsequent tokens. Selected single-line branches
and invoked macro body lines are parsed with an empty registry, using generated
default-package syntax. Parse dynamic calls permissively: strict structural
parsing does not consult runtime definitions. Global custom handlers and include
resolvers are not consulted. These reparses use the existing argument parser; raw argument quoting
and escape interpretation remain parser contracts, not a new roff input grammar.

| Input | Evaluation behavior |
| --- | --- |
| `.nr name expression [increment]` | Checked Go `int` arithmetic after interpolation. A leading `+`/`-` adds/subtracts the **entire following expression** from the current register. Omitted increments retain metadata; explicit zero clears it. Invalid expressions and overflow error. |
| `.rr name ...` | Removes each register and its increment metadata. |
| `.ds name value`, `.as name value` | Store/append literal parsed string values; append is bounded before concatenation. |
| `.de1`, `.dei`, `.dei1`, `.am1`, `.ami`, `.ami1` blocks | Preserved as complete variant tokens with a diagnostic; compatibility and indirect definition execution remain unsupported. Body requests do not execute. |
| `.de name` / `.am name` blocks | Copy/append raw body lines; append to an undefined macro creates it. Redefinition replaces the object shared by its aliases. |
| Unknown control matching a defined macro | Decode quoted/escaped arguments with `ArgumentParser`, interpolate each argument once, then reparse/evaluate body lines in a scoped call frame. Names colliding with generated syntax remain unsupported with diagnostics. |
| `.als new old`, `.rm name ...` | Alias a string/macro object or remove names. Redefinition/append through either name updates the shared object. Removing its original name preserves the object under remaining aliases. Undefined alias targets retain the request with a diagnostic. |
| `.if`, `.ie`, `.el` | Single-line branches. Integer expressions are true when **positive**, `n` is true and `t` is false. Leading `!` negates; numeric comparisons support `<`, `>`, `<=`, `>=`, `=`, `==`. String comparison uses `'left'right'` with interpolation; the historical missing-final-quote shorthand is retained. Each executed `.ie` pushes a result; `.el` consumes the most recent result across the current call, including nested branches/macros. An unpaired/repeated `.el` errors. Pairing does not carry across calls. |
| Text | One interpolation pass; other escapes remain literal. |
| Comments, empty controls | Pass through unchanged; no evaluation effect. |
| `.ig` block | Supported, effectless: the ignored body produces no output. |
| Generated man/ms formatting tokens | Interpolate plain string fields in a copy, preserving token type and argument boundaries. Formatting/layout semantics remain unevaluated with diagnostics. |
| Other tokens, including further roff requests, preprocessors, `.so`, `.sy`, device/filesystem requests and unknown controls | Pass through unchanged, with `Environment.Diagnostics`. They perform no evaluation action. |

Disabled branches are not reparsed, interpolated or executed. Selected multiline
conditional blocks (`\{`) return an explicit unsupported error. Scaling
units, loops, aggregate macro parameters, request shadowing and general
groff copy mode are not implemented. `.as` on a macro and `.am` on a string
report explicit unsupported errors. Strings and macros share a definition
namespace; simultaneous caller-supplied definitions of both kinds under one name
are rejected. Catalogue syntax coverage does not imply evaluation support.

Numeric expressions support decimal integers, unary `+`/`-`, parentheses,
`+ - * / %`, comparisons and positive-truth logical `&`/`:`. All binary operators
have the same precedence and evaluate **left to right**: `3+5*4` is `32`, whereas
`3+(5*4)` is `23`. Division truncates toward zero, and the remainder has the sign
of the dividend. Neither logical operator short-circuits. `!` is a conditional
prefix, not an inner arithmetic operator. The evaluator reconstructs numeric
parentheses split across legacy request fields, including `.nr X (3 + 5)*4 2`
and `.if (\n[X] == 32) .B yes`; it does not reparse/interpolate disabled branch
text while finding the predicate boundary. Other argument fidelity remains the
parser's contract. String comparisons containing spaces must occupy one quoted
parser argument; unquoted string-delimiter grammar is not reconstructed. This
profile uses overflow **errors**, including division of
the minimum integer by `-1`; it does not reproduce GNU troff's saturation policy.
Sources: [numeric grammar](https://www.gnu.org/software/groff/manual/groff.html.node/Numeric-Expressions.html),
[relative register assignment](https://www.gnu.org/software/groff/manual/groff.html.node/Setting-Registers.html).

Text interpolation recognizes `\*[name]`, `\*(na`, `\*n` and corresponding `\n`
register references, including Unicode names. Undefined strings remain as their
original escape; undefined registers become `0`. Inserted string values are
opaque, so a string referring to itself or another string is not expanded
recursively. Plain register references do not increment. `\n+` and `\n-`
update by the configured increment **before** inserting the new value, for all
three name forms. Arithmetic is checked; absent increments act as zero. Malformed
names report errors. Other escapes and invalid UTF-8 text bytes remain literal.

Inside a macro, `\$0` is the call name (including aliases), `\$1` through `\$9`,
`\$(01` and `\$[10]` insert positional values; absent arguments insert empty
text. `\n[.$]` is the scoped parameter count and is read-only. Nested calls
restore their caller's frame. Arguments are interpolated once at invocation;
inserted argument values stay opaque even when reused or passed to formatting
fields. Parameters cannot construct a request name dynamically. `\$*`, `\$@`
and other unsupported parameter forms error explicitly in a call frame. Outside
frames, parameter escapes remain literal. A doubled backslash before `n`, `*`
or `$` in an active frame delays that reference by one level, accommodating
common macro-body spelling. This is not general groff copy mode. Quoting and
invalid-byte decoding in macro arguments inherit `ArgumentParser` behavior.
See [GNU parameter syntax](https://www.gnu.org/software/groff/manual/groff.html.node/Parameters.html).

## Budgets and cancellation

Values <= 0 select defaults. Values above the ceiling select the ceiling; no
setting disables a budget. Byte limits are bytes, not runes.

| Environment field | Default | Ceiling | Accounting |
| --- | ---: | ---: | --- |
| `MaxDepth` | 50 | 128 | Active nested macro calls, body-line reparses, numeric parentheses and selected conditional branches; also bounds each alias chain. |
| `MaxSteps` | 50,000 | 1,000,000 | State validation, token dispatch, line parsing, interpolation segments, unary condition operators and other work checkpoints. |
| `MaxWorkBytes` | 32 MiB | 256 MiB | Strings scanned/copied for interpolation, reparsing, string assignment/append and state-name bookkeeping. Repeated processing is charged again. |
| `MaxOutputBytes` | 8 MiB | 128 MiB | Sum of emitted `TokenText` bytes, plus a separate bound of the same size on each temporary interpolation and the sum of each frame's argument values. |
| `MaxTokens` | 50,000 | 1,000,000 | All emitted tokens, including borrowed/preserved controls and comments. |
| `MaxStateBytes` | 8 MiB | 128 MiB | Names and stored values/body lines, plus 32 per binding and 16 per macro body line. Registers and increment entries each count separately; aliases count their own names and targets. |
| `MaxLineBytes` | 64 KiB | 1 MiB | Selected branches, macro-body/argument reparses and reconstructed numeric predicates. |

The byte budget for output does not serialize or copy preserved tokens. Their
existing fields are borrowed from the input; token count bounds the output
slice. A caller serializing them needs its own writer limit. State accounting
is a conservative size model, not a byte-exact bound on Go heap use. Existing
input/caller allocations are outside these budgets. Initial state validation
can itself exhaust a budget. Resource counters are deliberately conservative.

`StepCount` accumulates across calls on an environment, including initial state
validation. Reset it explicitly to start a new work allowance. Other counters
and conditional pairing start fresh per call. `Depth` is observational and is
reset to zero on exit. Cancellation uses `context.Context.Err()` at work
checkpoints, including within interpolation; it cannot interrupt arbitrary
caller-supplied Go methods. No evaluator request executes shell commands, reads
files, fetches resources or invokes include resolvers.

## Composable state

`Environment.State` selects an `EvaluationState` implementation. Nil uses the
environment's legacy maps, including `Aliases`; with custom state those maps are
not consulted or modified. `Environment` itself implements the interface, so a
separate environment can be the backing store, wrapped to observe or reject
operations. The [state tests](../evaluator_state_test.go) demonstrate decoration
without replacing the evaluator or connecting storage callbacks to parsing.

The interface has `Load`, `Store`, `Delete`, and `Walk`. Namespaced values hold
register/increment integers, string/alias text or copied macro bodies. `Walk`
must enumerate each binding exactly once and honor a false yield result; loaded
bodies are read-only and `Store` must copy caller-owned bodies. Initial state,
loads and writes are subject to the same budgets. Invalid fields, duplicate
bindings, definition/alias collisions, missing alias targets and alias cycles
are errors. Stored aliases target live canonical definitions directly; aliases
created by evaluation are normalized this way. When a canonical name is removed,
the least remaining alias becomes canonical, preserving deterministic output.
Public string/macro setters and string lookup follow these aliases.

State methods are explicit caller-supplied Go capabilities, just like custom
token metadata methods; they are trusted, not interruptible and must not mutate
unrelated state. Errors retain their identity through `EvaluationError`. Storage
is not transactional: a failure during a multi-binding definition/alias operation
may leave partial state requiring inspection or replacement. In particular, do
not assume a failed write rolled back earlier operations. Default storage uses
only memory; no shell, filesystem, device or network capability is provided by
roff requests.

## Errors, partial output and diagnostics

On failure, `Evaluate` and `EvaluateContext` return the already emitted prefix
and an error. State changes applied before the failure remain; evaluation is
not transactional. No parser error from a selected branch or macro body is
ignored. Original input tokens, source bytes and macro definition bodies are
not modified. Passthrough output tokens share identity with their input tokens;
formatting tokens with changed argument values are fresh copies.

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
