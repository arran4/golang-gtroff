# Practical command-line workflows

The examples describe the currently merged **headless** CLI. Build the executable from the repository root:

```sh
go build -o /tmp/gtroff ./cmd/gtroff
```

## Validate source in automation

```sh
/tmp/gtroff validate --strict testdata/simple.1
/tmp/gtroff validate --format json testdata/simple.1
printf '.SH NAME\nexample\n' | /tmp/gtroff validate --format json -
```

Documented exit codes: **0** accepted syntax, **1** invalid syntax, **2** usage/selection error, **3** input/output/resource failure. Distinguish process exit code from a diagnostic's human-readable description. Strict mode rejects warnings; permissive mode preserves unfamiliar syntax where possible.

## Inspect the source-aware AST

```sh
/tmp/gtroff ast --format text examples/man-tool.1
/tmp/gtroff ast --format json examples/man-tool.1 > /tmp/man-tree.json
/tmp/gtroff ast --select section:OPTIONS --format text examples/man-tool.1
```

Selectors include `section:HEADING`, `node:ID` and structural `path:/0/1`. Obtain IDs from the same document. Package selection is important for names shared between `man` and `ms`; use `--package ms` or `--package roff` where appropriate.

## Render or convert

```sh
/tmp/gtroff render --format html -o /tmp/page.html examples/man-tool.1
/tmp/gtroff render --theme dark --section OPTIONS -o /tmp/options.html examples/man-tool.1
/tmp/gtroff render --fragment --section OPTIONS examples/man-tool.1 > /tmp/options-fragment.html
/tmp/gtroff convert --from html --to roff -i examples/convert.html -o /tmp/page.1
/tmp/gtroff convert --from roff --to html -i /tmp/page.1 -o /tmp/reconverted.html
```

The HTML exporter escapes source text and produces a standalone page. Conversion does not promise byte-identical results for unsupported syntax; check diagnostics and retain the original document.

## For agents and batch jobs

Prefer structured `validate`/`ast` JSON and preserve stdout/stderr separation and exit status. Include invalid flags, incomplete documents, read errors and unknown syntax in test coverage. Do not assume the planned generated CLI and agent command inventory in [issue #59](https://github.com/arran4/golang-gtroff/issues/59) already exists.

Read [CLI contracts](cli.md), [conversion](convert.md) and [development](development.md).
