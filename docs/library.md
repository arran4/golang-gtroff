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

`Document.Source()` and each token's `RawSource` retain read bytes independently
of typed/canonical formatting. `SourceComplete` records observed EOF; partial
results are returned with strict syntax or input errors. See
[document source, span and budget contracts](document.md), including the
executable `ExampleDocument_Source`. The independent AST is the structural API.

### Custom line handlers

Implement `CustomHandler` and register it with `NewRegistry().RegisterCustom`.
Pass `WithRegistry(registry)` to `ParseDocument` or `ParseAST`, or call
`registry.ParseLine(line, packageName)` / `registry.ParseLinePermissive` directly.
`WithCustomHandler(handler)` adds a handler for just that parse without changing
the supplied registry. See the executable `ExampleWithRegistry` in
[`registry_example_test.go`](../registry_example_test.go).

An explicit empty registry isolates a parse from process-global registrations.
With no explicit registry (or `WithRegistry(nil)`), legacy global registration
via `RegisterCustomHandler` remains available. Document/AST parsing snapshots
registrations before reading input; includes use the same snapshot. Later
registration affects subsequent parses. Handler objects themselves are shared:
protect their mutable state when parsing concurrently.

Custom handlers precede generated dispatch for ordinary control lines. `cmd`
retains the original `.`/`'` prefix and `args` retains quoting/escaping with leading
argument whitespace removed. `CanHandle` must be a pure predicate. Exactly one
matching custom handler may parse: overlapping predicates/duplicate matches return
`ErrAmbiguousHandler` before any `Parse` callback. Nil registrations and successful
nil token results return `ErrInvalidHandler`. Predicate overlap is checked for
encountered requests, not exhaustively across all possible command names.

No match uses the generated token parser or the usual unknown-control behavior.
Handler syntax errors stay errors; permissive document/AST parsing preserves the
line and records a diagnostic. AST source bytes and copied typed token snapshots
remain independent of a custom token's formatted `String()` value. Custom tokens
must support JSON round trips for typed AST inspection; only serialized state is
copied, and JSON encoding failures produce diagnostics. For document/AST
structure, [BlockHandler](blocks.md) supplies declarative bounded capture with
shared registry precedence. Line handlers do not execute macro definitions.

This #5 repair connects per-parse handlers and makes dispatch safe. Catalogue-wide
validation of missing handwritten implementations remains separate; this is not
a claim that all #5/#12 acceptance criteria are complete.

### Explicit legacy includes

`ParseDocument` reads no included files unless given `WithIncludeResolver`.
For a caller-selected directory, use
`WithIncludeResolver(NewSafeDirectoryIncludeResolver(directory, 10))` and
`WithMaxIncludeDepth(10)`. The directory resolver uses Go's `os.Root` to keep
relative symlink targets within that tree; absolute paths and escaping symlinks
are rejected. Treat the chosen root and its filesystem contents as trusted;
`os.Root` does not prohibit mount points or special files beneath a root.

`IncludeResolver.Resolve` keeps its `io.Reader` signature. A successful call
transfers ownership to the parser: readers implementing `io.Closer` are closed
on success or failure, and close errors are reported. Wrap a borrowed reader
as `struct{ io.Reader }{reader}` to retain ownership. The original reader passed
to `ParseDocument` remains caller-owned.

The parser bounds nested includes for every resolver (default 10, maximum 128),
detects repeated active include names, and forwards line/line-length limits to
each included source. Line limits apply per source; `WithDocumentBytes` bounds
total reader bytes across the root and includes (default 8 MiB). The directory
resolver also detects active file aliases
and bounds open include readers with its `MaxDepth`; closing a reader releases
that reservation, so sequential/repeated sibling includes and resolver reuse
work. Configure resolver fields before concurrent use. Included diagnostics and
source locations are retained. `Document.Includes` retains child sources and
directive spans, even when a child returns partial results. Strict mode rejects
failed includes; permissive mode preserves the refused directive and reports a
warning for resolver/syntax failures. Terminal child reader/resource errors
always propagate as errors.

`ParseAST`, syntax-only CLI commands, exporters and the man-library server never
execute includes. The separate [source-set API and `sources` command](sources.md)
resolve bounded local sources without macro execution.
This repair addresses containment, reader ownership and recursion under #12;
the [block capture follow-up](blocks.md) separately supplies handler registration
and exact block-source inspection.

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

### Argument source and formatting

`ParseArguments` retains raw bytes, comments, byte spans and partial arguments
alongside decoded values. `ArgumentParser` and `LexArguments` share its scanner
and retain legacy rune indices. `CanonicalArguments` reports unrepresentable
values instead of claiming a universal quoting round trip. See the
[argument contracts](arguments.md) and [executable example](../arguments_example_test.go).
Generated zero-argument parsers now report extras; broader catalogue grammar is
still tracked separately under #7.

### Optional Bounded Evaluation

Evaluation is a separate opt-in pass. It returns partial output on errors and
keeps earlier state changes. Inspect `env.Diagnostics` for preserved requests;
use `EvaluateContext` for cancellation. The [evaluation guide](evaluation.md)
documents checked expressions, alias/append/remove semantics, positional macro
arguments, byte/work limits and remaining unsupported syntax. `Environment.State`
accepts a four-method storage interface for composing state access.

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

### Source-backed in-memory transforms

The experimental [`edit` package](edit.md) shares source/query/model APIs and
keeps mutable working bytes separate from original ASTs. Hash/version/span plans
validate before Apply; filesystem writes require an explicit Commit callback or
writer. [Complete Go examples](../edit/example_test.go) exercise source-exact
patches, literal named transforms, document creation and commit acknowledgement.
