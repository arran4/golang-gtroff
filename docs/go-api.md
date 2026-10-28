# Go library API examples

gtroff's Go packages can be used without invoking its CLI. The source AST, semantic man model and conversion APIs have deliberately different fidelity guarantees.

## Inspect a source-aware AST

```go
package main

import (
    "fmt"
    "log"
    "strings"

    groff "github.com/arran4/golang-gtroff"
)

func main() {
    source := ".TH HELLO 1\n.SH NAME\nhello sample\n"
    tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
    if err != nil {
        log.Fatal(err)
    }
    for _, section := range tree.Root().FindSection("NAME", false) {
        fmt.Println(section.Text())
        fmt.Println(section.Raw())
    }
}
```

`Raw()` returns the original source spelling; `Text()` extracts parsed text. Nodes also expose source positions, document-local IDs, and navigation/search capabilities such as `FindMacro`, `FindKind`, `FindText` and `FindSpan`. The AST does not execute roff input or open include files.

## Build semantic man content

```go
package main

import (
    "fmt"
    "log"
    "strings"

    groff "github.com/arran4/golang-gtroff"
    "github.com/arran4/golang-gtroff/man"
)

func main() {
    source := ".TH HELLO 1\n.SH OPTIONS\n.TP\n.B -v\nVerbose mode.\n"
    tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
    if err != nil {
        log.Fatal(err)
    }
    doc, err := man.Build(tree, man.Options{Strict: true})
    if err != nil {
        log.Fatal(err)
    }
    options, err := doc.SelectSection("OPTIONS", false)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(options.Text(man.TextOptions{}))
}
```

The semantic model supports common man-page structures but is not the source AST or a complete typesetter.

For token-level access, use `ParseDocument`, `ParseLine`, `ParseLineWithPackage` and `ParseInline`. For detail on source preservation and limits, see [AST](ast.md) and [semantic man](man.md).
