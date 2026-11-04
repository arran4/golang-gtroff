# Development and verification

This is an actively developed Go library with stacked PRs and generated parser code.

## Repository layout

- Module root: parser, AST, generated token catalogue and document/evaluator APIs.
- `cmd/gtroff/`: headless executable; `cmd/lex/`: compatibility lexer.
- `cmd/gen/`: catalogue schema, templates and generator.
- `internal/cli/`: testable CLI command logic.
- `man/`, `renderhtml/` and `convert/`: semantics and outputs.
- `testdata/`: readable and TXTAR scenario fixtures.
- `docs/*.md`: maintained documentation, staged for Hugo at build time.

## Verify before review

Use Go 1.26 or later and check the repository's actual module toolchain:

```sh
go test ./...
go vet ./...
go generate ./...
git diff --exit-code
go test . -run '^TestASTScenarios$' -count=1
go test . -run '^TestCLIScenarios$' -count=1
go test . -run '^TestHTMLScenarios$' -count=1
go test . -run '^TestConvertScenarios$' -count=1
```

Generated code must not be patched manually; change templates or definitions instead. Do not run golden-file update flags just to silence a difference. Inspect scenarios, errors, output and expected data before accepting a baseline change.

## Build documentation

```sh
git submodule update --init --recursive
python3 docs/scripts/stage-docs.py
hugo --source docs --minify --baseURL 'https://arran4.github.io/golang-gtroff/'
```

Hugo presentation copies under `docs/content/docs/reference/` are ignored; edit the original `docs/*.md`. See [documentation-site](documentation-site.md).

## Stacked review and CI

A draft PR is not ready. A PR targeting a feature branch is not automatically ready for `main`; inspect base, parent ancestry, conflicts and actual integration diff. The historical merged state of a feature PR is not proof its changes reached `main`.

GitHub Actions quota failures can occur before a job runs. Record those as **missing hosted verification**, not source code test failures or successful tests. Keep release gates separate from public documentation publishing.
