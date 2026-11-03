# File parsing fixtures

`parser_file_test.go` checks the committed roff fixtures under `testdata`:
`simple.1`, `complex.7`, `minimal.1` and `environ_snippet.7`. The source tests
require successful strict parsing, no diagnostics, and the expected typed
macros/requests. Their helper retains blank lines and the final unterminated
line. A regression verifies that a newly unknown request fails the helper.
Line-by-line tests also fail on parse errors or nil tokens; errors are not just
logged. `.nf` and `.fi` are typed requests and have assertions in the complex
fixture and catalogue route tests.

`groff_man.txt` and `groff_ms.txt` are committed **formatted reference prose**.
`TestParseReferenceText` requires those files and parses their text, but does not
present that result as man/ms macro conformance. Untracked local `environ.txt`
is not part of the required suite, and no test/benchmark silently discovers it.
The small committed `environ_snippet.7` is the reproducible roff source case.

The [catalogue support report](docs/catalogue-support.md) enumerates all 518
records. Its conformance test executes each claimed generated/custom/inline
syntax route, independently of fixture token counts. Metadata counts do not
prove grammar completeness or evaluator/formatter behavior.

Readable TXTAR scenarios cover AST/source offsets and extraction, man semantics,
unknown/invalid controls, Unicode, EOF, man/ms contexts, multiline recovery,
include/registry boundaries and both conversion loops. See
[testdata/README.md](testdata/README.md). Normal tests do not update expectations;
the documented update flags are explicit review operations.

```sh
go test ./...
go test . -run 'TestParseFile|TestParseReferenceText|TestParseLineByLine|TestCatalogueSupportRoutes' -count=1
go test . -run '^$' -bench 'BenchmarkParse(Line|File|ReferenceText)$' -benchmem
```

Benchmarks use required checked-in inputs and fail on parse errors. Timings are
observations for the executing host, not correctness evidence or a reason to
weaken fidelity assertions. No GNU groff, installed man database or network is
required. Formatter differential equivalence is not claimed.
