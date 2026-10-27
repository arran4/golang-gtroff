# golang-gtroff

A Go library and CLI for complete, extensible groff and troff document parsing, macro extraction, and bounded evaluation.

`golang-gtroff` is designed as a **loss-aware, structured syntax and document parser**. It provides typed nodes for groff macros and requests, preserves physical lines and source positions, models complex constructs like multiline macro definitions and preprocessor blocks, and provides an optional bounded semantic evaluator. It does **not** attempt to be a full typesetter or pixel/page-layout engine.

## Features

- **Catalogue Coverage**: 518 canonical definitions across macros (141 entries, 111 distinct), requests (206 entries, 179 distinct), strings (59 entries), and registers (112 entries) with 0 unexplained gaps.
- **Package-Aware Macro Resolution**: Full support for both `man` and `ms` packages with automatic collision resolution for shared macros (`.SH`, `.B`, `.BI`, `.NH`, `.DS`, etc.).
- **Roff Request Support**: Typed support for low-level roff requests including `.nf`, `.fi`, `.br`, `.sp`, `.nr`, `.ds`, `.so`, and more.
- **Preprocessor Blocks**: Structured capture of `tbl` (`.TS`/`.TE`), `eqn` (`.EQ`/`.EN`), `pic` (`.PS`/`.PE`), and `refer` (`.[`/`.]`) regions.
- **Multiline & Definition Handlers**: Structured blocks for macro definitions (`.de`, `.am`), ignored text (`.ig`), and source directives (`.so`, `.mso`).
- **Inline Escapes & References**: Reusable inline parser for string interpolations (`\*[HF]`), register references (`\n[PO]`), special characters (`\(bu`, `\[bu]`), font changes (`\fB`), and size escapes.
- **Streaming Document Parser**: `ParseDocument` preserves every physical line, source positions (line, column, byte offset), blank lines, comments, and unterminated final lines.
- **Optional Bounded Evaluator**: Opt-in semantic evaluation for number registers (`.nr`, `.rr`), strings (`.ds`, `.as`), conditionals (`.if`, `.ie`, `.el`), and macro expansion with strict recursion and step budgets.
- **Deterministic Code Generation**: Atomic, `go/format`-verified code generation backed by schema validation.

## Installation

```bash
go get github.com/arran4/golang-gtroff
```

## Usage

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

A CLI lexer is included for inspecting groff files:

```bash
# Parse a man page file
go run ./cmd/lex/main.go -package man myfile.1

# Parse in strict mode (fails on unknown commands)
go run ./cmd/lex/main.go -strict myfile.1

# Read from stdin
cat myfile.1 | go run ./cmd/lex/main.go
```

## Supported Scope & Honest Boundaries

- **Syntax & AST Layer (Complete)**: Dot control lines, apostrophe control lines, line arguments, quotes, escapes, preprocessor blocks, macro definitions, comments, and unknown controls.
- **Macro Packages**: `man` and `ms` macro packages are fully typed with package disambiguation.
- **Requests**: 179 distinct roff requests are typed and dispatched.
- **Semantic Evaluation (Opt-in)**: Core register arithmetic (`.nr`, `.rr`), strings (`.ds`, `.as`), conditionals (`.if`, `.ie`, `.el`), and macro definitions (`.de`).
- **Out of Scope**: Full groff typesetting, hyphenation engines, diversion rendering, and font rasterization/page layout.

## Development

### Code Generation

To regenerate macro and request definitions, ParseLine dispatchers, and catalogue metadata:

```bash
go generate ./...
```

The generator uses:
- `cmd/gen/in.json` - Raw catalogue definitions
- `cmd/gen/schema.go` - Normalization and validation schema
- `cmd/gen/*.template` - Code generation templates

### Testing & Validation

```bash
go test -v ./...
go vet ./...
```

## License

This project follows the licensing of the source material:
- Documentation derived from the [GNU groff manual](https://www.gnu.org/software/groff/manual/groff.html#Copying-This-Manual)
- Source code - GPL (to maintain compatibility with groff source material)

See [LICENSE](LICENSE) for details.