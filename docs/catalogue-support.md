# Catalogue syntax support

The [generated support report](generated/catalogue-support.json) lists 518 roff/man/ms records by token, package, kind and parser route.

| Parse status | Records | Meaning |
| --- | ---: | --- |
| supported_generated | 330 | Generated line parser and typed token |
| supported_custom | 17 | Generated token with custom document or block capture |
| syntactically_preserved | 171 | Preserved string/register reference |
| unimplemented | 0 | Missing syntax route |
| ambiguous | 0 | Ambiguous syntax route |

`package` identifies the documented feature family. `parser_package`, where present, identifies the compatibility parser namespace; for example, `.QS` belongs to ms but the distinct historical man and ms token constructors remain available. `Provenance` is a reference-family hint, not a verified passage citation.

Every record has `evaluation_status: "not_assessed"`. Syntax coverage is not a claim of complete GNU grammar, semantic interpretation, evaluation or rendering. The [argument](arguments.md), [block](blocks.md) and [evaluation](evaluation.md) guides describe those separate capabilities.

The report comes from `cmd/gen/in.json` and the same generation plan as the parser. Refresh with `go generate ./...`; `TestCatalogueSupportRoutes` exercises each syntax route. See [generation](generation.md).
