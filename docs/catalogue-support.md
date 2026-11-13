# Catalogue syntax support

[`generated/catalogue-support.json`](generated/catalogue-support.json) lists all
518 current canonical records by index, type, package and token. It is generated
from `cmd/gen/in.json` and the same definition/dispatch plan used by the Go
templates. `go generate ./...` refreshes it; normal tests do not rewrite it.
Review its diff when approving catalogue or classification changes.

`package` identifies the documented feature's family. An optional
`parser_package` identifies an existing executable binding when compatibility
requires a different context. For example, both `.QS` catalogue descriptions
belong to ms, while the historical `TokenManQS` and `TokenMsQS` constructors
remain distinct. Row 0 therefore carries `package: "ms"` and
`parser_package: "man"`. Route tests use the latter when present; the public
catalogue metadata retains the former. See [association evidence and input
schema](provenance/corrections.md). `Provenance` remains a reference-family hint,
not a verified passage citation.

| Parse status | Current records | Evidence |
| --- | ---: | --- |
| `supported_generated` | 330 | Generated line parser, concrete token type and metadata |
| `supported_custom` | 17 | Generated line token plus handwritten document/block route |
| `syntactically_preserved` | 171 | String/register reference kind, name and exact raw spelling |
| `unimplemented` | 0 | Missing definition/dispatch/custom route or invalid classification |
| `ambiguous` | 0 | Duplicate dispatch or incorrect package-specific parser |

`TestCatalogueSupportRoutes` executes every row, using explicit package context
and an isolated registry. The examples probe dispatch with required arguments;
they are not comprehensive grammar fixtures. For custom block/definition routes,
the test also parses the complete example and checks its concrete handwritten
token/body. Indirect variants use a default-end example; explicit/unresolved
indirect boundaries are verified separately in the block scenarios. A missing
handwritten implementation cannot satisfy that check by falling back to an ordinary generated token. Includes remain syntax-only in these tests.

Catalogue `declared_status` is retained separately: its 325 `generated`, 22
`custom_handler` and 171 `syntax_preserved` labels describe declarations. Six
definition variants now declare handwritten multiline capture; their generated
line-token APIs remain available.
Five custom declarations (`.so`, `.mso`, `.if`, `.ie`, `.el`) have generated syntax
tokens with separate optional execution. They count as generated syntax here;
they do not count as verified custom evaluation.

Every row's `evaluation_status` is **`not_assessed`**. This report proves neither
formatter equivalence nor complete grammar/evaluation support. Declared [argument binding](arguments.md) and [block capture](blocks.md) have
separate conformance tests; the syntax route count does not prove their complete
GNU grammar or evaluation semantics. Source snapshots belong to the
AST; canonical generated `String()` output is not a byte archive.

Generator validation rejects unknown kinds/statuses/packages, empty data,
inconsistent indices, malformed optional-argument delimiters and conflicting
generated names before replacing outputs. `unexplained_gaps` is computed from
invalid/missing/ambiguous planned routes. Tests deliberately remove definitions
or dispatches and corrupt a package variant to ensure the count changes.
Generated output drift and runtime route assertions form independent checks.
The #16 generation repair renders all outputs first, stages complete batches and
rolls back ordinary replacement failures. Renames remain atomic per file, with
no group visibility/crash guarantee. See [generation contracts](generation.md).

The all-entry probe found that `.Pn path` lost its required argument to an
optional punctuation slot. The small #7 prerequisite repair reserves arguments
for required slots in both legacy matchers and keeps explicit positional slots
when generated canonical formatting would otherwise make `.Pn` ambiguous.
Generated parsers now use the declared binding policies described in the
[argument contract](arguments.md), while legacy matcher joining remains a
compatibility API. Block registrations/delimiters come from catalogue `Block`
metadata; missing or duplicate block plans change the measured support gaps.

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
