# Getting started

**Work in progress:** gtroff is a pure-Go toolkit for a documented subset of roff/man, not a full GNU groff replacement or a stable binary release.

## Requirements and installation

Use Go 1.26 or later (the project currently specifies Go 1.26.10). The headless command does not require a C toolchain, system groff, browser or installed font package.

```sh
git clone https://github.com/arran4/golang-gtroff.git
cd golang-gtroff
go build -o /tmp/gtroff ./cmd/gtroff
/tmp/gtroff help
/tmp/gtroff version
```

Go developers can also install the development version with `go install github.com/arran4/golang-gtroff/cmd/gtroff@latest`. There is no compatibility promise for untagged development code.

## Create a manual page

Save this source as `/tmp/hello.1`:

```roff
.TH HELLO 1
.SH NAME
hello \- a sample command
.SH SYNOPSIS
.B hello
.SH DESCRIPTION
A small manual page.
```

Now inspect and export:

```sh
/tmp/gtroff validate --strict /tmp/hello.1
/tmp/gtroff ast --format text /tmp/hello.1
/tmp/gtroff ast --format json /tmp/hello.1 > /tmp/hello.ast.json
/tmp/gtroff render --format html -o /tmp/hello.html /tmp/hello.1
```

The default HTML output includes bundled fonts and styles. A successful syntax check does not establish full roff semantic compatibility.

## Select and convert

```sh
/tmp/gtroff ast --select section:NAME --format text /tmp/hello.1
/tmp/gtroff render --section DESCRIPTION -o /tmp/description.html /tmp/hello.1
/tmp/gtroff convert --from roff --to html -i /tmp/hello.1 -o /tmp/converted.html
/tmp/gtroff convert --from html --to roff -i examples/convert.html -o /tmp/converted.1
```

Conversion is semantic and supports a documented subset. Keep the original file.

Read [CLI workflows](workflows.md), [Go API](go-api.md), [CLI reference](cli.md), [AST](ast.md), [semantic man](man.md), [HTML](html.md), and [compatibility](compatibility.md).
