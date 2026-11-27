# Development

Use Go 1.26 or newer. The headless module contains the parser, AST, semantic model, CLI, rendering and HTTP library. The optional viewer lives in the separate `desktop` module.

## Tests and generation

```sh
go test ./...
go vet ./...
go run ./cmd/gen -check
go generate ./...
git diff --exit-code
```

Edit the catalogue or templates in `cmd/gen/` to change generated parser code. See [generation](generation.md) and [scenario fixtures](scenarios.md) for tooling and intentional golden-file updates.

## Documentation

The maintained guides live in `docs/*.md`. The [documentation site guide](documentation-site.md) explains how to preview them locally.
