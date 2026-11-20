# Registry snapshot allocation comparison

Issue #97 follows the allocation profile in #60/#96. The source change removes
only temporary Registry wrapper objects in line/block handler selection and
custom dispatch. Registration slices are still copied together under the read
lock; caller callbacks run after unlocking. Parser entry still owns its registry
snapshot. No public API, generated dispatch, catalogue, error or source contract
changes.

Measured 2026-11-13, Go 1.26.10, Linux amd64, Intel i5-12600KF, CGO0,
GOMAXPROCS1. Baseline is be4e2f8ed1baba939062659c51971d762164646d (#96);
candidate is the production change in this PR. Benchmark definitions and fixture
bytes are identical. Both native test binaries were built before timing; each
ran from its corresponding worktree. No OS affinity/governor control was applied.

Ten pairs were run, alternating which binary ran first on each round. Each
invocation used -test.run=^$, -test.bench=^Benchmark(ParseLine|ParseDocument|ParseAST)$,
-test.benchmem, -test.benchtime=200ms, -test.count=1 and -test.cpu=1. Standard
`go test` can reproduce these flags as described in [the guide](../benchmarks.md).
[Before](registry-before.txt) and [after](registry-after.txt) retain all raw results;
the nth occurrence of each name belongs to the nth pair.

The wall-clock levels shifted sharply midway through **both** series. This
measurement did not capture the cause. Aggregate timing medians alone can give a
misleading comparison across that shift. Paired ratios below are the median of
after/before within the ten corresponding rounds; they are observations, not a
statistical significance test or a general speed guarantee. Allocation reductions
are the primary result. There is no timing/allocation threshold in tests or CI.

| Benchmark | Before B/op | After B/op | Before allocs/op | After allocs/op | Paired timing delta |
| --- | ---: | ---: | ---: | ---: | ---: |
| BenchmarkParseLine | 187 | 130 | 6 | 6 | -7.2% |
| BenchmarkParseDocument/man-small | 21,096 | 17,336 | 264 | 217 | -10.5% |
| BenchmarkParseDocument/man-medium | 241,048 | 206,488 | 2,377 | 1,945 | -7.1% |
| BenchmarkParseDocument/ms-small | 17,984 | 15,744 | 134 | 106 | -10.6% |
| BenchmarkParseDocument/roff-small | 8,776 | 7,896 | 65 | 54 | -9.6% |
| BenchmarkParseAST/man-small | 52,826 | 49,227 | 667 | 622 | -5.9% |
| BenchmarkParseAST/man-medium | 487,142 | 452,580 | 5,622 | 5,190 | -6.0% |
| BenchmarkParseAST/ms-small | 30,315 | 28,395 | 320 | 296 | -3.0% |
| BenchmarkParseAST/roff-small | 17,841 | 17,121 | 182 | 173 | -2.8% |

The 8,738-byte man AST workload saves about 34.6 KB and 432 allocations per parse
(about 7.1% of allocated bytes). Its document workload saves the same number of
allocations. The mixed line benchmark averages seven inputs, so the integer
allocs/op output can hide a fractional per-operation reduction even while B/op
falls. Counters depend on toolchain and fixture; they are not API guarantees.

For the medium document timing, the aggregate median was ~212 us before and
~234 us after, while the within-pair median ratio improved. This discrepancy is
why no simple aggregate speedup claim is made. Retain the raw samples and repeat
under a controlled environment when latency is the decision criterion.

Focused tests cover reentrant registration from CanHandle/Parse, line/block
selection using one snapshot, next-call ambiguity without callback execution,
nil registries and generated fallback. Existing TXTAR, entry-snapshot and
concurrent-registration suites remain the functional contracts. A five-second
wait only bounds a broken callback-lock deadlock; it is not a performance gate.
