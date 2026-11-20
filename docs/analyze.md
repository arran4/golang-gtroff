# Explain document features

`gtroff analyze` inventories roff syntax, source relationships and processing
requirements without executing macros or rendering. Its primary result is a
classification with evidence, not a document-quality score.

```sh
./gtroff analyze examples/man-tool.1
./gtroff analyze --details document.1
./gtroff analyze --format json document.1
./gtroff analyze --no-includes --format jsonl document.1
./gtroff analyze --rules questions.json document.1
```

Options precede the optional file. Omit the file, or use `-`, for stdin. Named
files use the [shared bounded source policy](sources.md): local includes resolve
by default beneath the input directory or an explicit `--root`. `--no-includes`
retains references without loading. Stdin needs an explicit root to load anything.
No system macro path, network fetch or external formatter is used.

## Separate capability facts

Every finding identifies a named rule, feature/name, source occurrence, original
AST node/parent, original byte span, parsed package, possible catalogue packages,
source activity, explanation, optional target and processing tags. Schema version
1 and capability version 1 label the report's vocabulary.

| Stage | What the report establishes |
| --- | --- |
| Syntax `parsed` | The original snapshot has a parsed token/structure/inline form |
| Syntax `recognized` | The catalogue spelling is known, but this source is malformed or lacks a successful route |
| Syntax `unknown` | An active control has no known catalogue spelling |
| Syntax `opaque` | Stored or recovered source is retained without interpreting it as live syntax |
| Semantics `modeled` | Source contributes to or is accepted without diagnostics by the bounded optional man model, or a supported decoder form is identified in that context |
| Semantics `partially-modeled` | The source contributes but has unresolved interpretation details |
| Semantics `unsupported` | The man projection preserves unsupported literal content or an unsupported escape |
| Semantics `not-assessed` | No applicable active contributor/profile was assessed |
| Resolution | Actual source-edge state, independently of recognition and semantics |
| Rendering `not-assessed` | This analyzer did not render; parsed/modelled syntax is not a rendering guarantee |

Known catalogue names and their generated/custom declarations do not prove
successful parsing, evaluation or rendering. The analyzer snapshots existing
catalogue metadata and uses the actual AST and man-model contributors rather
than a second document parser. It never copies catalogue prose into explanations.
Inline support is scoped to the current bounded man decoder; unsupported fonts,
size changes, strings/registers and other escapes remain separately visible.
Source diagnostics provide details such as unsupported font requests.

The man assessment is explicit in `semanticProfile: "man"`; it is not a claim
that every source is a man document. ms, mdoc and mixed-profile semantics are
unassessed. Actual parsed metadata matters: distinctive ms tokens can be parsed
by the existing dispatch fallback even with the default man context, while their
ms semantics remain unassessed. `--package` selects parser context for ambiguous
spellings, not the document-profile result.

## Profiles and uncertainty

Profiles are lexical hypotheses with confidence, reasons and positioned evidence.
A man-style `.TH` header supports `man`; ms-style `.TL`/`.AU`/`.NH` support `ms`;
mdoc-style `.Dd`/`.Dt`/`.Nm`/`.Sh` support recognition of `mdoc` spelling only.
Multiple package-specific families produce `mixed`. Definition syntax without a
distinct package title produces `custom`; otherwise core controls can establish
`roff`. Plain text and ambiguous headings remain `unknown` with low confidence.
Repeated `.SH` alone does not establish man rather than ms.

Opaque/comment bodies do not provide active profile evidence. A `.TH` stored in
`.ig` or a macro body cannot change the surrounding profile. Conditional truth,
macro redefinition and runtime state are never evaluated; confidence describes
observed spellings, not a proof of execution or package compatibility. Each
source has its own profile; the report also summarizes observed source-set
families, preserving source identity on every evidence item.

## Feature inventory

Built-in named detectors cover:

- Physical controls, comments, text and blank lines, including unknown material.
- Parsed escape families, strings/register references and current man support.
- Macro/string/register definitions, aliases, removals and register formatting.
- Conditionals/loops and other stateful flow/trap/diversion/environment requests.
- Command/file/device-effect-looking requests, reported as syntax and never run.
- Includes, their literal targets and actual bounded resolution states.
- tbl/eqn/pic/refer captures and their additional interpretation requirements.
- Manual/URL/mail targets as data, without opening or validating destinations.
- Sections/subsections, nested block boundaries and opaque regions.

`BuiltinRules()` lists stable `builtin.*` IDs and descriptions. Definitions and
references retain names/targets and original positions; they are not resolved
through macro execution. Includes retain source edges separately, including
missing/denied/cyclic/disabled references and captured partial children. A child
with an incomplete load can be `partially-resolved` even though its directive
was parsed. Loading a file does not execute its contents.

`active` means outside parser-owned opaque captures and comments, not “executed”.
Inline material after a recognized comment delimiter is inactive. Conditionals
are not evaluated; their payloads are not recursively interpreted as a second
program. The inventory cannot prove absence of all possible runtime effects.
Stored control-looking syntax remains visible with inactive evidence, so an
ignored `.sy` is not confused with a command that ran.

Presence flags (`hasConditional`, `hasInclude`, `usesPreprocessor`, definitions,
references, side-effect syntax and opacity) summarize emitted builtin evidence
and can include stored source. `hasUnknown` refers to unknown active controls.
`requiresEvaluation` uses active builtin interpretation requirements; it is not
set by ignored/comment material or by custom annotations. `hasPartialSource`
records missing observed EOF. EOF does not by itself prove syntax validity or
success: a parser/resource error may still accompany captured complete bytes.

`requirements` combines processing tags from active findings, including user
annotations. `evaluation`, `source-resolution`, `preprocess:tbl` and
`advanced-interpretation` describe what applying a construct's native meaning
would require. Source inspection and literal fallback do not automatically need
those facilities. `explicit-side-effect-policy` identifies requests that would
need a deliberate policy in an interpreter, not permission for this tool to run
them. Custom tags remain attributable to their custom rule.

Counts have deliberately different meanings: feature/rule counts count findings
(and can overlap), control and escape counts count emitted builtin observations,
and package counts describe parsed-token metadata. Physical-line/byte/node/depth
metrics are observations, not scores. A recovery tail can be one AST line node
while representing several physical lines; physical line counts use captured
source bytes. Repeated included occurrences count again. On a limit, counts and
flags describe emitted finding prefixes; source preflight/profile observations
can already cover the captured source being analyzed.

## Go and declarative rules

```go
report, err := analyze.Run(ctx, sources, analyze.Options{})
if err != nil {
    // report contains bounded partial evidence; check report.Complete.
    return err
}
fmt.Println(report.Flags.HasConditional, report.Flags.RequiresEvaluation)
```

`Run` accepts an explicit immutable `source.Set`, not a pathname or filesystem.
The resulting report is caller-owned data and keeps source-local identities.
It never reacquires I/O capabilities. External examples in
[analyze/example_test.go](../analyze/example_test.go) demonstrate classification,
custom registries, rule parsing and writing to an explicit `io.Writer`.

A schema-version-1 rules file adds questions/annotations:

```json
{
  "schemaVersion": 1,
  "rules": [
    {
      "id": "project.has-table",
      "match": {"macro": ".TS"},
      "feature": "project-table",
      "reason": "Inspect the captured table source",
      "requires": ["manual-review"]
    }
  ]
}
```

Use `analyze.ParseRules(reader)` or `analyze.NewRegistry(rules...)`, then set
`Options.Registry`. `report.HasRule("project.has-table")` tests a named question;
ordinary Go code can inspect the typed flags/counts/findings. Registries snapshot
rule slices, requirement slices and optional boolean pointers; accessors return
copies. Built-in detectors always run. Custom rules cannot overwrite builtin
IDs or capability stages, and cannot change builtin observation flags.

Match conditions are literal conjunctions: syntax `kind`, dot-normalized `macro`,
parsed `package`, `inlineKind` or `opaque`. At least one condition is required.
A macro without explicit kind matches physical control lines; specify
`kind: "block"` to match a capture instead. Inline rules cannot also require
macro/package attributes that inline nodes do not have. Values do not execute
expressions or scripts. IDs/features/tags use bounded lowercase identifier
spellings; `builtin.*` is reserved. Requirements are declared tags, not imported
capabilities. There are no rule imports, source includes, external programs,
network calls or plugin installation in this file format.

Files are limited to 64 KiB and 128 custom rules. Unknown JSON fields, unsupported
versions, duplicate rule IDs, missing conditions/reasons, malformed match data
and trailing JSON are errors. `--rules` is an explicit file capability; no home,
working-directory or implicit configuration file is discovered. Named rule files
are closed before source loading, including failures. Invalid rules are usage
errors and do not read source input.

## Reports, errors and bounds

Text defaults to profile/reason, flags, requirements, aggregate counts, source
facts, include states and a diagnostic count. `--details` adds each original
`file:line:column` finding and parser/semantic diagnostics. JSON always exports
full evidence. JSONL emits a versioned summary record followed by versioned
finding records. Formats intentionally differ; JSON text follows Go's invalid
UTF-8 replacement rules, and this inventory is not a source archive. Keep the
original source set/AST for exact raw bytes, comments and opaque material.

CLI exit codes are 0 for completed enumeration, 1 for strict syntax/include
rejection, 2 for usage/rule errors and 3 for I/O/resource/cancellation failure.
Unsupported interpretation is classified, not treated as a lint policy violation.
`--strict` controls the shared source-loading diagnostics; analysis remains
separate from structural lint rules. `Complete` means enumeration finished over
the supplied snapshot. Per-source EOF, include states, `inputError`, `error` and
diagnostics retain separate completeness/failure facts.

Library defaults: 200,000 visited AST nodes, 100,000 findings and 67,108,864 work
units across source bytes, nodes, rules and explanations. Cancellation is checked
between nodes/phases and after finalization. The optional man-model pass has its
own bounded construction limits and is checked around that bounded work; host
readers/filesystems own blocked-I/O deadlines. Resource limits return errors and
partial reports, never successful silent truncation. CLI exposes
`--max-analysis-nodes`, `--max-findings` (ceilings 1,000,000) and `--max-work`
(ceiling 1,073,741,824), alongside shared source budgets. The budget is not a
promise of exact wall-clock time or encoded JSON size.

Tests cover profiles/uncertainty, positive and negative capability evidence,
unknown/malformed controls, definitions, conditionals, stored/comment material,
unresolved/resolved/disabled sources, partial input, CRLF/invalid UTF-8, recovery
metrics, rules/reader ownership, custom-fact isolation, deterministic counts,
resource/cancellation, JSON/JSONL/text and short writes. CLI TXTAR and a built
executable exercise shared include policy, explicit rules, strict errors and
limits. Opt-in `BenchmarkFeatureInventory` uses deterministic 10/1,000-definition
inputs with checked control counts; no absolute timing threshold is imposed.
