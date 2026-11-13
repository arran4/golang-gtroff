# Parser and catalogue benchmarks

Issue [#60](https://github.com/arran4/golang-gtroff/issues/60) tracks measurement
quality separately from syntax/catalogue conformance. Benchmarks run only when
explicitly selected; no scheduled workflow or absolute-duration test gate is added.
The module's pinned Go toolchain supports testing.B.Loop. No installed man pages,
untracked environ.txt, service or network fetch is needed by the workloads.

## Inventory and boundaries

The #64 baseline already contains ParseLine, ParseFile and ParseReferenceText.
These names and operations are retained; allocation reporting is made explicit.
The additional families fill the missing in-memory, package, AST/model, argument,
catalogue and generator coverage. All errors stop the benchmark.

| Family | Work and timed boundary | Fixture/context |
| --- | --- | --- |
| ParseLine | Round-robin typed line parsing, default package | Seven existing request/macro/comment/text lines |
| ParseFile | Open/read/close plus strict document parsing and token-slice projection | simple.1, complex.7, existing licensed environ_snippet.7 |
| ParseReferenceText | Same file pipeline over formatted reference **prose** | groff_man.txt and groff_ms.txt; not ms macro conformance |
| ParseDocument | In-memory strict token/source capture; registry snapshotting included | small/medium man, small ms, literal roff requests |
| ParseAST | In-memory strict syntax/inline/source tree construction | Same four inputs; no include loading/evaluation |
| ManBuild | Strict semantic construction over a pre-parsed immutable AST | Two man fixtures; parsing excluded |
| ParseArguments | Source-aware argument scanning | ASCII, quoting/escaped spaces/comment, Unicode |
| LexArguments | Compatibility lexemes and rune offsets | Quoted Unicode/escaped space/special name/comment |
| GeneratorArgumentGrammar | In-process generator signature grammar | Required, nested optional and ellipsis signatures |
| CatalogueLookup | Exact-package/collision, request, fallback and miss | Current generated catalogue; no new index |
| GeneratorCheck | Compiled read-only generator CLI: process startup, normalize/render/gofmt/check | Actual embedded in.json/templates and committed outputs; build excluded |

File workload definitions live in benchmarks_test.go; their ordinary
TestBenchmarkWorkloadContracts test checks strict diagnostics, source EOF, exact
captured bytes, model validity and the final medium-fixture definition. Parser
error/unknown/blank-line behavior remains separately tested in parser_file_test.go
and the existing TXTAR suites. Semantic output is not an exact-source roundtrip.
The ms/roff cases measure syntax recognition, not runtime macro evaluation.

The medium man page is an authored synthetic 100-definition ENVIRONMENT workload
(8,738 bytes); the ms report and roff request examples are 374 and 194 bytes.
The small man file reuses the existing 470-byte fixture. None is sampled from a
host installation. See [recorded hashes and example results](benchmarks/example-report.md).

GeneratorCheck builds the existing generator into an owned temporary directory
before timing and invokes only -check. It never refreshes generated output. A
missing/stale output fails the run. Parent B/op and allocs/op include subprocess
scaffolding, **not the child's allocations**; that is an explicit limitation of
this end-to-end measurement. The in-process signature benchmark reports its own
allocations. Per-phase generator heap attribution would require separate child
profiling; do not infer it from parent metrics. No handwritten files are added
under the generated cmd tree.

## Run a small smoke or repeated measurement

From the repository root:

```sh
# Discover actual benchmark names without running their workloads.
go test ./... -run '^$' -list '^Benchmark'

# One operation per case: checks setup/errors, not stable performance.
CGO_ENABLED=0 go test . -run '^$' -bench . -benchmem -benchtime=1x -count=1

# Repeat in-process parser/AST/model/argument/catalogue measurements.
CGO_ENABLED=0 go test . -run '^$' -bench 'Benchmark(Parse(Line|Document|AST|Arguments)|LexArguments|ManBuild|CatalogueLookup)' -benchmem -benchtime=500ms -count=10 -cpu=1 > parser.txt

# Generator check and signature parser, explicitly selected.
CGO_ENABLED=0 go test . -run '^$' -bench '^BenchmarkGenerator' -benchmem -benchtime=500ms -count=10 -cpu=1 > generator.txt
```

`-cpu=1` controls Go GOMAXPROCS, not OS affinity. The generator-check child receives
the same GOMAXPROCS as the parent. Repeated file runs usually exercise a warm OS
cache; use in-memory families when isolating parser CPU/allocation changes.
SetBytes normalizes MB/s by input length, including for ManBuild where the AST
is already parsed. Go B/op reports allocated bytes, not input bytes. Setup and
fixture loading are outside new
B.Loop timing. The legacy file-family measurements intentionally retain file I/O.
[Go's testing documentation](https://pkg.go.dev/testing#B.Loop) describes B.Loop's
measurement and compiler-retention behavior.

Record the exact commit/worktree diff, fixture hashes, command, `go version`,
GOOS/GOARCH, CGO_ENABLED, GOMAXPROCS/CPU model and relevant build flags. Keep the
machine idle and use the same toolchain, fixtures, names, flags and environment
for both sides. Do not compare a one-operation smoke with calibrated runs.
The [example report](benchmarks/example-report.md) commits ten samples per name
and separates observations from guarantees.

## Compare a candidate

Use two separate checkouts containing the **same benchmark definitions and
fixtures**, one at the desired baseline and one at the candidate. Do not edit the
baseline while a run is active. For example, run this command in each checkout,
changing only the output destination:

```sh
CGO_ENABLED=0 go test . -run '^$' -bench 'Benchmark(Parse(Line|Document|AST|Arguments)|LexArguments|ManBuild|CatalogueLookup)' -benchmem -benchtime=500ms -count=10 -cpu=1 > /tmp/before.txt
CGO_ENABLED=0 go test . -run '^$' -bench 'Benchmark(Parse(Line|Document|AST|Arguments)|LexArguments|ManBuild|CatalogueLookup)' -benchmem -benchtime=500ms -count=10 -cpu=1 > /tmp/after.txt
benchstat /tmp/before.txt /tmp/after.txt
```

Benchstat is an optional comparison tool, not a module/runtime dependency. Record
its installed version separately; this procedure installs nothing. Its official
[usage guide](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) documents matched
benchmark comparison, repeated samples and confidence reporting. Missing/renamed
cases or changed fixture hashes invalidate a direct comparison. Look at bytes and
allocations as well as time; retain raw samples and investigate noise before
proposing an optimization. Passing correctness tests remains a separate gate.

For source allocation/CPU attribution, select a narrow **in-process** family:

```sh
CGO_ENABLED=0 go test . -run '^$' -bench '^BenchmarkParseAST/man-medium$' -benchtime=2s -cpu=1 -cpuprofile=/tmp/gtroff-ast.cpu -memprofile=/tmp/gtroff-ast.mem -o /tmp/gtroff-bench.test
go tool pprof -top /tmp/gtroff-ast.cpu
go tool pprof -alloc_space -top /tmp/gtroff-ast.mem
```

These profiles describe the benchmark process; they cannot attribute the
GeneratorCheck child. Longer runs/profiles are opt-in. There is no timing or
allocation threshold tied to a Go version/compiler optimization.

## Scope and future feature benchmarks

This parser/catalogue work branches directly from #64. Query/extraction/editing
and optional Tengo PRs already add their own operation-specific benchmarks; they
are independent siblings/descendants and are not copied into this PR or claimed
as present on its base. Run their actual packages after integrating those PRs.
Analysis/lint/diff/renderer throughput and corpus-wide cold-start/include-graph
profiles are outside this parser/catalogue checkpoint. Existing issues #68–#74,
#29 and #30 own those consumer/integration scopes. When an observed allocation or
scaling defect warrants work, attach the matched samples/profile and exact source
lines to its owning issue, then propose a separate optimization PR. A small mixed
fixture's ratio alone cannot prove linear or quadratic asymptotic behavior.
