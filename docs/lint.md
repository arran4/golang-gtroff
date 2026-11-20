# Structural documentation policy

`gtroff lint` applies conventions to explicit roff source snapshots. It reports
policy findings separately from parser diagnostics and the `analyze` capability
inventory. It does not execute macros, render pages, fix files, or inspect paths
that were not explicitly loaded. The API and JSON schema are version 1; this is
an initial policy vocabulary, not a promise of complete groff understanding.

```sh
gtroff lint page.1
gtroff lint --strict page.1
gtroff lint --required NAME,DESCRIPTION,ENVIRONMENT page.1
gtroff lint --format json --fail-on warning page.1 other.1
gtroff lint --closed-world page.1 other.1
gtroff lint --no-includes page.1
```

Flags precede positional filenames. No filenames means borrowed stdin; `-` is
accepted at most once. Named files use the shared bounded source loader, with
local includes relative to the referring file and confined to `--root` (default:
each root input's directory). Stdin needs an explicit root for includes. No
system macro search, network fetch, or arbitrary filesystem traversal is enabled.
The original immutable AST and exact captured bytes remain available through
`source.Set`. Lint does not serialize a replacement roff document.

## Rules and confidence

| ID | Default severity | Question |
| --- | --- | --- |
| `man.required-section` | error; SYNOPSIS warning | Are required section headings present? |
| `man.duplicate-section` | warning | Does a required section repeat? |
| `man.empty-section` | warning, heuristic | Does a section/subsection have modeled body content? |
| `man.name-pattern` | warning, heuristic | Does NAME follow `name \- description`? |
| `man.empty-definition` | warning, heuristic | Is a required term or a tagged description missing? |
| `man.duplicate-definition` | warning, heuristic | Does a simple term repeat in one structural scope? |
| `man.indent-depth` | warning, heuristic | Does relative indentation exceed the configured depth? |
| `source.unresolved-include` | warning | Could a requested local include be inspected? |
| `reference.local-target` | warning, semantic | Does an explicit `.MR name section` have a collection target? |
| `capability.unsupported` | warning, optional | Is active syntax unsupported by the optional man interpreter? |
| `profile.man-policy` | info, heuristic | Were man conventions skipped for another profile? |

NAME and DESCRIPTION are required by default. SYNOPSIS is a warning for manual
sections 1/8; strict policy adds 2/3. `requiredSections` replaces that convention;
an explicit empty array disables required-section questions. Subsections do not
satisfy required section headings. Required headings are compared after folding
case and whitespace. Duplicate headings are checked only for required sections.
Empty-body checks also inspect optional sections/subsections.

`--strict` enables capability warnings, sets the default failure threshold to
warning, and asks the source loader to reject all syntax/include diagnostics.
`--fail-on` can override the policy threshold independently. Custom severity
settings override a rule's emitted severity, including uncertainty downgrades.

Known ms/mdoc/mixed profiles skip man conventions with an informational finding.
Unknown/roff profiles are checked under the command's declared man conventions;
profile detection remains a lexical hypothesis. Incomplete input, unresolved or
disabled includes, unknown syntax, and constructs requiring evaluation downgrade
missing sections to heuristic warnings: inspected source does not prove their
runtime absence. Unsupported content and include directives prevent false empty
body/NAME warnings. These are conservative policies, not a compatibility score.

Definitions use the existing man model's item/tag structure. `.TP`/`.TQ` require
a term; an empty `.IP` tag is valid. Terms are compared case-sensitively after
whitespace normalization, within the nearest section, subsection or indent.
Included fragments inherit the scope at their referring directive. Multiline
alias tags and unsupported definitions are excluded from duplicate-term checks.
Intentional label-only entries or repeated labels can be suppressed.

Semantic interpretation remains **file-local**. Loading an include does not
propagate font, section, macro or register state as groff execution would.
Collection checks aggregate evidence; they do not create a merged runtime AST.
Parent-file body checks containing an include are conservatively skipped rather
than guessing whether a child supplied content.

Reference checks are off by default: normal manuals refer to pages outside the
supplied collection. `--closed-world` declares that all manual targets are in the
explicit collection. Only active `.MR name section` references with literal
complete targets are checked against observed title/section metadata, ignoring
case. A collection with input errors, partial/unknown/dynamic source, missing
metadata or incompatible profiles skips the missing-target check. URLs, inferred
prose links, incomplete `.MR` targets and opaque stored references are not checked.

Unsupported-capability warnings use analysis evidence, with one finding per
original syntax node. They describe the optional man interpreter's support, not
GNU groff portability. Include failures have their own rule. `not-assessed` is
not recast as `unsupported`; stored/opaque syntax is not treated as live execution.

## Configuration and suppression

Configuration is an explicitly opened JSON file, limited to 64 KiB. It cannot
load callbacks, import other files or run scripts. Unknown fields/IDs, invalid
severities and duplicate required headings are rejected before reading sources.

```json
{
  "schemaVersion": 1,
  "policy": {
    "requiredSections": ["NAME", "DESCRIPTION", "ENVIRONMENT"],
    "disable": ["man.name-pattern"],
    "enable": ["capability.unsupported"],
    "severity": {"man.duplicate-definition": "error"},
    "maxIndentDepth": 6,
    "closedWorld": false
  }
}
```

```sh
gtroff lint --rules policy.json page.1
```

`--enable`/`--disable` append comma-separated IDs to configuration lists; disable
wins. `--required` replaces the list when nonempty. Use configuration `[]` to
require no sections. `--closed-world` enables the configured collection policy.
Defaults use no global configuration search or ambient environment settings.

Only real standalone comment tokens outside opaque captures are directives:

```roff
.\" gtroff-lint-disable man.name-pattern
.SH NAME
An intentional project-specific presentation.
.\" gtroff-lint-enable man.name-pattern
.\" gtroff-lint-disable-next-line man.empty-section
.SH RESERVED
```

Use comma-separated rule IDs without spaces, or `all`. Disable starts on the
comment line and lasts until enable or source EOF; next-line applies to a finding
whose original start line is the following physical line. Required-section
findings are anchored at root offset zero, so an initial disable comment can
suppress them. Suppressions never cross source occurrences or include boundaries.
Inline comments and comments inside stored macro/preprocessor captures do not
activate suppression. Unknown/malformed directives produce separate configuration
diagnostics and fail input validity, including with `--fail-on never`.

Suppressed findings retain their spans/messages in a separate array. Disabling a
rule prevents it from running. Neither form hides parser diagnostics or I/O
errors. Source errors are not configurable lint rules.

## Go API and budgets

```go
set, loadErr := source.Load(ctx, fsys, "page.1", source.Options{})
report, err := lint.Run(ctx, []lint.Document{{
    Name: "page.1", Root: "project-docs", Sources: set, InputError: loadErr,
}}, lint.Options{})
// Handle err (operational failure), then report.Failed() (CI policy/input result).
```

`Document.Name` and `Root` are labels; `Run` never opens them. Source IDs are local
to each document's `source.Set`, so every finding includes a one-based document
index. Unreadable/absent snapshots remain document records without invented
missing-section findings. `InputError` preserves source-loader classifications.
Incomplete source captures and parser/optional-model error diagnostics fail input validity; warnings do so in
strict mode. Profiles, source metrics and diagnostics remain separate report data.

`NewRegistry` adds trusted Go callback rules with stable IDs, default severity,
category, explanation and `Check(ctx, *Context, Reporter) error`. `Rules()` exposes
the inventory. Context exposes immutable sources and copied section/definition
snapshots. `Reporter.Emit` validates span bounds and existing file/node identities and applies policy,
budgets and suppression. Callbacks must honor context and call `Reporter.Spend`
for their own substantial work. They are host code with host privileges; the API
is not a sandbox for untrusted scripts. See executable external-package examples
in `lint/example_test.go`.

Defaults are 256 root documents, 200,000 analysis AST nodes, 100,000 emitted
findings including suppressed, and 64 Mi charged work units. Supported ceilings
are 4096 roots, 1,000,000 nodes/findings and 1 Gi work units. Nonpositive API limits
select defaults; CLI limits must be positive. Analysis exposes nonserialized
`Report.WorkUnits` so lint can deduct actual charged analysis work from the shared
budget. Structural traversal, source bytes, text/predicate work, suppression and
emission are charged as well. These are defined work units, not CPU instructions;
source/model construction is independently bounded by input nodes/bytes.

CLI source bytes/nodes/occurrences are cumulative across all root loads, not reset
per file. Include depth is bounded per root. Root paths are explicit, not a glob
or directory crawler. Cancellation and resource failures stop work with a sorted
partial report. Input errors are retained while other explicit roots can still
be inspected; a resource/cancellation failure stops further loading.

## Results and CI

Findings contain schema version, rule ID, severity, category, source/node/span,
message, optional suggestion and related source locations. Suggestions are text;
no automatic edits occur. Order is document, source occurrence, byte offset, rule
ID, message; equal keys preserve emission order. Active and suppressed arrays each
have their own one-based ordinals after sorting. Source occurrence order is not
necessarily the interleaved include logical-line order.

JSON contains the full report, including the applied policy and an explicit
collection reference-check state (`not-requested`, `pending`, `checked`, or
`skipped-incomplete`). JSONL starts with a summary record retaining input
and configuration diagnostics, then active and suppressed finding records. Text
prints findings, suggestions, related locations, input diagnostics and suppression
evidence. JSON replaces invalid UTF-8 according to `encoding/json`; exact bytes
remain in the original source snapshot.

`Complete` means policy enumeration completed, not valid source or renderability.
`Run` returns an error for configuration/execution/limit/cancellation failure;
`Failed()` also checks incomplete/error reports, invalid inputs and suppression
configuration, then compares active findings to `FailOn`. `never` disables only
policy-threshold failure. Zero API threshold means error, or warning in strict.

CLI exits: 0 passes; 1 policy threshold or invalid input; 2 usage/configuration;
3 I/O/resource/cancellation failure. A report is written before a known operational
error when a snapshot/report exists; usage errors can occur before any report.
Writer errors and short writes are propagated. JSONL summary counts can describe
only an inspected prefix when complete is false. Tests cover these distinctions,
collection budgets, false positives and original source preservation.
