# Example local benchmark report

Captured 2026-11-06, Go 1.26.10, Linux amd64, Intel i5-12600KF, CGO disabled.
The source baseline is #64 at a5359412d0363ade8b7dd611e365a79412397b98 plus the
benchmark/fixture additions in this PR. Production parser/generator code is
unchanged. This report is historical evidence, not a timing/allocation CI gate.

```sh
CGO_ENABLED=0 go test . -run '^$' -bench . -benchmem -benchtime=100ms -count=10 -cpu=1
```

GOMAXPROCS is 1 in both the benchmark parent and generator-check child; this does
not pin an OS core. No CPU affinity or power-governor control was applied. File
benchmarks use the local filesystem/cache. Native generator compilation and its
initial check run before timing. Each reported name has ten samples. Values below
are medians and observed ranges, not confidence intervals or an A/B significance
claim. [Raw Go benchmark output](baseline-linux-amd64.txt) retains every sample.

| Benchmark | Median ns/op | Observed ns/op range | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: | ---: |
| BenchmarkParseLine | 257.25 | 256.9–262.6 | 187 | 6 |
| BenchmarkParseFile/simple | 18,016 | 17,910–18,462 | 22,232 | 271 |
| BenchmarkParseFile/complex | 34,550 | 34,253–35,806 | 40,016 | 533 |
| BenchmarkParseFile/environ_snippet | 44,215.5 | 43,921–44,421 | 68,816 | 575 |
| BenchmarkParseReferenceText/groff_man.txt | 201,862 | 198,954–205,674 | 505,000 | 1,335 |
| BenchmarkParseReferenceText/groff_ms.txt | 298,833 | 294,701–306,001 | 780,329 | 1,868 |
| BenchmarkParseDocument/man-small | 15,000.5 | 14,849–15,082 | 21,096 | 264 |
| BenchmarkParseDocument/man-medium | 141,895.5 | 141,057–142,269 | 241,048 | 2,377 |
| BenchmarkParseDocument/ms-small | 9,292.5 | 9,172–9,432 | 17,984 | 134 |
| BenchmarkParseDocument/roff-small | 4,315.5 | 4,265–4,337 | 8,776 | 65 |
| BenchmarkParseAST/man-small | 45,660 | 45,134–45,871 | 52,819 | 667 |
| BenchmarkParseAST/man-medium | 451,878.5 | 449,609–473,054 | 487,142 | 5,622 |
| BenchmarkParseAST/ms-small | 24,538 | 24,487–24,738 | 30,313 | 320 |
| BenchmarkParseAST/roff-small | 12,580 | 12,540–12,826 | 17,841 | 182 |
| BenchmarkManBuild/man-small | 29,857 | 29,768–30,162 | 55,544 | 405 |
| BenchmarkManBuild/man-medium | 266,985 | 265,728–268,994 | 382,306 | 3,069 |
| BenchmarkParseArguments/ascii | 702.95 | 698.6–707.2 | 1,288 | 14 |
| BenchmarkParseArguments/quoted-escaped | 604.75 | 602–612.2 | 744 | 13 |
| BenchmarkParseArguments/unicode | 526.75 | 525–528.6 | 728 | 12 |
| BenchmarkLexArguments | 449.1 | 446.2–459.2 | 496 | 11 |
| BenchmarkGeneratorArgumentGrammar/required | 1,371 | 1,351–1,392 | 600 | 69 |
| BenchmarkGeneratorArgumentGrammar/nested-optional | 3,592.5 | 3,578–3,612 | 1,880 | 175 |
| BenchmarkGeneratorArgumentGrammar/repeating | 1,773 | 1,764–1,823 | 776 | 76 |
| BenchmarkCatalogueLookup/man | 4.506 | 4.492–4.573 | 0 | 0 |
| BenchmarkCatalogueLookup/ms-collision | 194.95 | 189.4–200.8 | 0 | 0 |
| BenchmarkCatalogueLookup/request | 382.05 | 367.7–397.1 | 0 | 0 |
| BenchmarkCatalogueLookup/fallback | 952.3 | 926–996.9 | 0 | 0 |
| BenchmarkCatalogueLookup/missing | 291.9 | 284.4–296.4 | 0 | 0 |
| BenchmarkGeneratorCheck | 93,474,147 | 91,517,400–96,506,820 | 12,728 | 46 |

GeneratorCheck's B/op and allocs/op measure only the parent subprocess scaffolding.
They do **not** describe the generator child's heap. Its ns/op includes startup,
normalization, template execution, Go formatting and generated-file comparison.
GeneratorArgumentGrammar separately measures the in-process signature parser.

The first single-operation smoke run is deliberately not used as this report:
startup/reflection caches can dominate it. Fixtures with different construct mixes
cannot establish asymptotic complexity from their size ratios. The catalogue
measurements cover exact-package, collision, fallback and missing paths; they do
not justify changing the public mutable catalogue contract to an unchecked index.

## Fixture identities

These lengths/digests identify the exact source bytes used for the file/AST/model
measurements. Reference prose is committed formatted text, not man/ms macro syntax.

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| testdata/simple.1 | 470 | f8f7aed85568fdd0b43fd0387aba5019867405c8c805b3d4b9fb25c99763dd31 |
| testdata/complex.7 | 1,068 | 593a28a58b90b476028342e74f8e1a6bd7c5dbd2784ebd5edf1135a2103f7dfa |
| testdata/environ_snippet.7 | 3,568 | 775dbb1c197658aa0a10195b3fe58f2940780d29bddc5bd23143d2e86f047414 |
| testdata/bench/environment.1 | 8,738 | 39efd96e2fe177bccdf40787b0e7f54482b71af72a40a63eac371543bc46e76f |
| testdata/bench/report.ms | 374 | 3d1f3e3d4293729b959e33c4faf963b5824be31f1069dd52b6d49b0af12eb7c3 |
| testdata/bench/requests.roff | 194 | 7429bed44483b9618f37bdf7fdfa1d1d01ec5c7189a3af0db7c25d935d67896b |
| groff_man.txt | 35,586 | 78f7d90548895a31d68255c64b4acd14de94b2315987a48179726aeadb7150f1 |
| groff_ms.txt | 51,873 | 45b272e30f8b220b549bc1702d58617e05b9649b9362e71ba2349c30d99f61aa |

Use the [runner/comparison guide](../benchmarks.md) for a new measurement.

## Follow-up profile (2026-11-13)

The documented two-second ParseAST/man-medium CPU/allocation profile was also
executed with Go 1.26.10 and GOMAXPROCS=1. Profiling changes timing; its ~826 us/op
must not be compared as an A/B regression against the unprofiled November 6 run.
The sampled allocation profile attributed about 54% of alloc_space to the AST
node-creation closure and about 7% to Registry.snapshot. These are attribution
observations, not fixed heap guarantees. Registry snapshots are copied again by
line-handler dispatch even after the parser captures its entry snapshot;
registry.go:165–166, 200–202 and 235–252 identify the inspected paths on the #64
base. A separately scoped optimization can examine temporary wrapper allocations
while preserving copied registrations and releasing locks before callbacks.
No production optimization or parsing contract change is included here.
