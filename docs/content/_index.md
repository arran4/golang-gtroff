---
title: "gtroff"
type: docs
---

# gtroff

> **Work in progress.** A pure-Go toolkit for a documented subset of roff/man; not a complete GNU groff replacement or a stable binary release.

Explore and validate roff syntax, inspect a source-aware AST, extract semantic man-page content and produce HTML or convert a documented HTML/man subset.

## Start here

- [Getting started](docs/reference/getting-started/) — build the CLI and process a first man page.
- [Practical CLI workflows](docs/reference/workflows/) — commands, JSON, selections and automation.
- [Go API examples](docs/reference/go-api/) — embed parsing and semantic inspection.
- [Compatibility](docs/reference/compatibility/) — boundaries and experimental features.
- [Development and tests](docs/reference/development/) — generation, TXTAR and PR conventions.
- [Full reference](docs/reference/) — AST, CLI, HTML, conversion and semantic man contracts.

## Try it

```sh
go build -o /tmp/gtroff ./cmd/gtroff
/tmp/gtroff validate --strict testdata/simple.1
/tmp/gtroff ast --format text testdata/simple.1
/tmp/gtroff render --format html -o /tmp/man.html examples/man-tool.1
```

**Documentation tracks the merged source.** Image, server, desktop and release work may exist on feature branches but are not claimed to be in the default branch until integrated. See the [source](https://github.com/arran4/golang-gtroff), [PRs](https://github.com/arran4/golang-gtroff/pulls) and [issues](https://github.com/arran4/golang-gtroff/issues).
