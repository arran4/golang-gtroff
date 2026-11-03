# Catalogue syntax support

[`generated/catalogue-support.json`](generated/catalogue-support.json) lists all
518 current canonical records by index, type, package and token. It is generated
from `cmd/gen/in.json` and the same definition/dispatch plan used by the Go
templates. `go generate ./...` refreshes it; normal tests do not rewrite it.
Review its diff when approving catalogue or classification changes.

| Parse status | Current records | Evidence |
| --- | ---: | --- |
| `supported_generated` | 336 | Generated line parser, concrete token type and metadata |
| `supported_custom` | 11 | Generated line token plus handwritten document/block route |
| `syntactically_preserved` | 171 | String/register reference kind, name and exact raw spelling |
| `unimplemented` | 0 | Missing definition/dispatch/custom route or invalid classification |
| `ambiguous` | 0 | Duplicate dispatch or incorrect package-specific parser |

`TestCatalogueSupportRoutes` executes every row, using explicit package context
and an isolated registry. The examples probe dispatch with required arguments;
they are not comprehensive grammar fixtures. For custom block/definition routes,
the test also parses the complete example and checks its concrete handwritten
token. A missing handwritten implementation cannot satisfy that check by falling
back to an ordinary generated token. Includes remain syntax-only in these tests.

Historical `declared_status` is retained separately: its 331 `generated`, 16
`custom_handler` and 171 `syntax_preserved` labels describe catalogue declarations.
Five custom declarations (`.so`, `.mso`, `.if`, `.ie`, `.el`) have generated syntax
tokens with separate optional execution. They count as generated syntax here;
they do not count as verified custom evaluation.

Every row's `evaluation_status` is **`not_assessed`**. This report proves neither
formatter equivalence nor complete grammar/evaluation support. Full `.Pn`/`.TH`
argument grammar, source-preserving serialization, richer blocks and bounded
evaluation remain separate tracked contracts. Source snapshots belong to the
AST; canonical generated `String()` output is not a byte archive.

Generator validation rejects unknown kinds/statuses/packages, empty data,
inconsistent indices, malformed optional-argument delimiters and conflicting
generated names before replacing outputs. `unexplained_gaps` is computed from
invalid/missing/ambiguous planned routes. Tests deliberately remove definitions
or dispatches and corrupt a package variant to ensure the count changes.
Generated output drift and runtime route assertions form independent checks.
Output replacement remains per file; collection-wide transaction work is #16.

The all-entry probe found that `.Pn path` lost its required argument to an
optional punctuation slot. The small #7 prerequisite repair reserves arguments
for required slots in both legacy matchers and keeps explicit positional slots
when generated canonical formatting would otherwise make `.Pn` ambiguous.
Legacy rest-argument joining remains unchanged; this is not the complete #7
grammar redesign.

Focused verification:

```sh
go test ./cmd/gen -count=1
go test . -run 'TestCatalogue|TestMatcherReserves|TestPnRequired|TestFileFixture' -count=1
go test ./cmd/gen -run '^$' -fuzz '^FuzzNormalizeCatalogue$' -fuzztime=5s
go generate ./...
git diff --exit-code
```

See [file fixture boundaries](../FILE_PARSING_TESTS.md) and the
[full roadmap verification record](roadmap-verification.md).
