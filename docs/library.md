# Library guide

Install the root module with `go get github.com/arran4/golang-gtroff`.
Use a reviewed feature revision while the roadmap remains unreleased.
The following snippets illustrate the APIs; complete executable examples live
in `ast_example_test.go` and the individual packages.

### Extract semantic man-page content

The `man` package groups physical source lines into styled paragraphs,
sections, tagged items, indented blocks and preformatted examples. It keeps
source provenance and warns about unsupported semantics.

```go
// Imports: groff "github.com/arran4/golang-gtroff"
//          "github.com/arran4/golang-gtroff/man"
tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
if err != nil {
    log.Fatal(err)
}
document, err := man.Build(tree, man.Options{Strict: true})
if err != nil {
    log.Fatal(err)
}
options, err := document.SelectSection("OPTIONS", false)
if err != nil {
    log.Fatal(err)
}
fmt.Print(options.Text(man.TextOptions{}))
```

Selections retain title metadata and inherited font/indent state. Use
`Raw: true` for original source slices or `Normalize: true` for folded display
text. The [representative man page](../examples/man-tool.1) and its
[tested semantic JSON](../examples/man-tool.semantic.json) demonstrate the supported
subset. See [semantic contracts and scenarios](man.md) for details.

### Select and inspect a man-page section

```go
source := ".TH DEMO 1\n.SH NAME\ndemo \\- inspect roff\n.SH OPTIONS\n.TP\n.B \\-v\nVerbose output.\n"
tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
if err != nil {
    log.Fatal(err)
}
options := tree.Root().FindSection("OPTIONS", false)[0]
fmt.Print(options.Text())
// OPTIONS
// \-v
// Verbose output.

fmt.Print(options.Raw()) // original .SH/.TP/.B spelling and line endings
token, err := options.FindMacro("B")[0].Token()
if err != nil {
    log.Fatal(err)
}
fmt.Println(token.(*groff.TokenManB).Text) // \-v
```

`Walk` supports pruning/stopping. `FindKind`, `FindMacro`, `FindSection`,
`FindText`, and `FindSpan` search a selected subtree in source order. Unknown or
malformed syntax stays visible with diagnostics. Strict mode returns an error
and the preserved tree. `Text()` extracts syntax arguments without evaluating
escapes; `Raw()` retains the original source. See the [AST contracts and fixture
instructions](ast.md) and the executable `ExampleParseAST` for details.

### Document Parsing (Streaming & Structured)

```go
package main

import (
    "fmt"
    "log"
    "strings"

    groff "github.com/arran4/golang-gtroff"
)

func main() {
    input := `.TH MYTOOL 1 2024-01-01 "Linux" "User Commands"
.SH NAME
mytool \- demonstrate groff parsing
.SH SYNOPSIS
.nf
.B mytool [options] file
.fi
.SH DESCRIPTION
Process input files.
`

    doc, err := groff.ParseDocument(strings.NewReader(input),
        groff.WithPackage("man"),
        groff.WithStrict(false),
        groff.WithSourceName("mytool.1"),
    )
    if err != nil {
        log.Fatalf("Parse error: %v", err)
    }

    for _, dt := range doc.Tokens {
        fmt.Printf("%s:%d:%d: %s\n", dt.Span.Start.Filename, dt.Span.Start.Line, dt.Span.Start.Column, dt.Token.String())
    }
}
```

### Line-by-Line Parsing

```go
token, err := groff.ParseLine(`.TH "TOOL" 1 "" "Header" "Footer"`)
if err != nil {
    log.Fatal(err)
}

switch t := token.(type) {
case *groff.TokenMacroTH:
    fmt.Printf("Title: %s, Section: %s\n", t.Title, t.Section)
}
```

### Package Selection (`man` vs `ms`)

```go
// In man mode, .SH takes a heading argument
tokMan, _ := groff.ParseLineWithPackage(`.SH "DESCRIPTION"`, "man")
fmt.Println(tokMan.(*groff.TokenManSH).Heading) // "DESCRIPTION"

// In ms mode, .SH takes an optional match-level argument
tokMs, _ := groff.ParseLineWithPackage(`.SH 2`, "ms")
fmt.Println(tokMs.(*groff.TokenMsSH).Matchlevel) // "2"
```

### Inline Escapes and References

```go
tokens, err := groff.ParseInline(`Check font \*[HF] and symbol \(bu at offset \n[PO]`)
if err != nil {
    log.Fatal(err)
}

for _, tok := range tokens {
    fmt.Printf("[%s] %s (name: %s)\n", tok.Kind, tok.Raw, tok.Name)
}
```

### Optional Bounded Evaluation

```go
input := `.nr PO 72
.ds GREET Hello
\*[GREET] from offset \n[PO]!
`
doc, _ := groff.ParseDocument(strings.NewReader(input))
var raw []groff.Token
for _, dt := range doc.Tokens {
    raw = append(raw, dt.Token)
}

env := groff.NewEnvironment()
evaluated, err := groff.Evaluate(raw, env)
if err != nil {
    log.Fatal(err)
}

for _, tok := range evaluated {
    fmt.Println(tok.String())
}
// Outputs: Hello from offset 72!
```

## Command-Line Tool

The original `cmd/lex` token-dump utility remains available for compatibility:

```bash
# Parse a man page file
go run ./cmd/lex/main.go -package man myfile.1

# Parse in strict mode (fails on unknown commands)
go run ./cmd/lex/main.go -strict myfile.1

# Read from stdin
cat myfile.1 | go run ./cmd/lex/main.go
```


See the [user guide](../readme.md) for commands and limitations.
