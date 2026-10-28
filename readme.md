# golang-gtroff

A pure-Go library and lexer CLI for inspecting roff documents, navigating a source-aware AST, and extracting syntax text. Linux man pages are the primary target; typed man/ms macros and roff requests are available.

`ParseAST` preserves original source independently of canonical token formatting. Inspect sections, select a subtree, search controls or content, and walk parent/child links without running roff commands or loading includes. Headless builds require no CGO, groff, browser, or system font tools. A separate semantic man model now renders whole pages or selections to self-contained HTML with embedded Go fonts. The `convert` command also converts a documented semantic HTML/man subset in both directions. PNG/JPEG export uses the same model with bounded pure-Go pagination.

## Features

- **Catalogue Coverage**: 518 canonical definitions across macros (141 entries, 111 distinct), requests (206 entries, 179 distinct), strings (59 entries), and registers (112 entries) recorded in the generated catalogue; coverage validation is still tracked in #15.
- **Package-Aware Macro Resolution**: Typed `man` and `ms` macros with package-specific collision resolution for shared macros (`.SH`, `.B`, `.BI`, `.NH`, `.DS`, etc.).
- **Roff Request Support**: Typed support for low-level roff requests including `.nf`, `.fi`, `.br`, `.sp`, `.nr`, `.ds`, `.so`, and more.
- **Preprocessor Blocks**: Structured capture of `tbl` (`.TS`/`.TE`), `eqn` (`.EQ`/`.EN`), `pic` (`.PS`/`.PE`), and `refer` (`.[`/`.]`) regions.
- **Multiline & Definition Handlers**: Structured blocks for macro definitions (`.de`, `.am`), ignored text (`.ig`), and source directives (`.so`, `.mso`).
- **Inline Escapes & References**: Reusable inline parser for string interpolations (`\*[HF]`), register references (`\n[PO]`), special characters (`\(bu`, `\[bu]`), font changes (`\fB`), and size escapes.
- **Source-aware AST**: immutable structure, stable document-local IDs, exact source slices, Unicode positions, typed token inspection, section/node selection, and deterministic traversal/search.
- **Document Parser**: the existing `ParseDocument` token API remains available. Use `ParseAST` for byte-exact source preservation.
- **Optional Bounded Evaluator**: Opt-in semantic evaluation for number registers (`.nr`, `.rr`), strings (`.ds`, `.as`), conditionals (`.if`, `.ie`, `.el`), and macro expansion with execution limits; remaining evaluator gaps are tracked in #13.
- **Deterministic Code Generation**: Sorted, `go/format`-verified output backed by schema validation; multi-output atomicity remains tracked in #16.

## Installation

Requires Go 1.26 or newer; the tested toolchain is Go 1.26.10. The HTML/font
dependencies use patched current releases. Headless builds require no C compiler.

```bash
go get github.com/arran4/golang-gtroff
```

## Usage

### Validate and inspect from the command line

```sh
go build -o gtroff ./cmd/gtroff
./gtroff validate --strict testdata/simple.1
# testdata/simple.1: valid syntax (semantic validation not performed)

./gtroff ast --select section:OPTIONS --format text testdata/simple.1
./gtroff ast --select section:OPTIONS testdata/simple.1 > options.json
printf '.B fragment\n' | ./gtroff validate --format json -
./gtroff help ast
./gtroff version
```

`validate` checks syntax; it does not promise successful typesetting. Unknown
controls are preserved with warnings; `--strict` rejects warnings. Malformed
syntax fails in either mode. Exit codes are 0 for accepted syntax, 1 for invalid
syntax, 2 for usage/selection errors and 3 for input/output/resource failures.

`ast` writes versioned JSON or text with source spans, raw content and stable
node IDs. Select by `section:HEADING`, `node:ID`, or `path:/0/1`. Omit FILE or use
`-` for stdin. JSON diagnostics are included in the envelope; text diagnostics
go to stderr. The command never evaluates roff or opens include targets. Output
is stdout-only for `ast`; HTML render also supports atomic `-o FILE` output.
See [CLI contracts and scenarios](docs/cli.md) for details.

### Export a readable HTML page

```sh
./gtroff render --format html -o page.html examples/man-tool.1
./gtroff render --theme dark --section OPTIONS -o options.html examples/man-tool.1
./gtroff render --fragment --section OPTIONS examples/man-tool.1 > fragment.html
```

Open the [generated example HTML](examples/man-tool.html) locally to read the
[representative man page](examples/man-tool.1). It includes Go Regular, Bold,
Italic and Mono fonts and responsive CSS. No installed fonts, groff or browser
process is needed to create it. `--font-regular`, `--font-bold`, `--font-italic`
and `--font-mono` optionally load explicitly named TTF/OTF files.

HTML preserves headings, paragraphs, definitions, styles, links and examples.
Unknown source stays escaped and visible with diagnostics; `--strict` leaves
existing output intact when diagnostics occur. Source text never executes as
HTML or JavaScript. See [HTML API, selection, safety and output contracts](docs/html.md).

### Export PNG or JPEG pages

```sh
./gtroff render --format png -o page.png examples/man-tool.1
./gtroff render --format jpg --theme dark -o page.jpg examples/man-tool.1
./gtroff render --format png --section OPTIONS --single -o options.png examples/man-tool.1
```

See the [PNG example](examples/man-tool.png) and [dark JPEG example](examples/man-tool.jpg).
Multi-page output uses `page-001.png`, `page-002.png`, etc. Stdout accepts one
image; `--single` derives height within guarded limits. Width/margins/page height
use pixels; image font size uses points at the requested DPI. Default bundled Go
fonts need no installed files. Unsupported syntax and missing Unicode glyphs
remain visible with diagnostics. See [image options, APIs, limits and tests](docs/images.md).

### Browse a local man library

```sh
./gtroff serve --root examples --listen 127.0.0.1:8080
# Open http://127.0.0.1:8080 and search for man-tool.
# On Linux, choose an installed collection explicitly:
./gtroff serve --root /usr/share/man/man1 --root /usr/share/man/man3
```

The read-only browser lists plain/gzip pages, searches names/descriptions/full
text and links to actual page headings. It uses the same semantic HTML renderer
and bundled Go fonts on Linux, Windows and macOS. No system `man` utility is
required; `--system-search` optionally supplements built-in results when available.
Requests never select filesystem paths or load includes. The snapshot refreshes
on restart. See [HTTP API, roots, budgets and scenarios](docs/serve.md).

### Convert HTML and man structure

```sh
./gtroff convert --from html --to roff --strict -i examples/convert.html -o converted.1
./gtroff convert --from roff --to html -i converted.1 -o converted.html
```

Headings, paragraphs, nested emphasis, definitions/lists, preformatted text and
safe links round-trip as documented semantics. Unknown tags, scripts, tables and
arbitrary CSS produce visible literal fallback and diagnostics. `--strict` rejects
warnings before replacing output. Original source, canonical roff and normalized
semantics remain separate. See the [tested HTML input](examples/convert.html),
[canonical man output](examples/convert.1) and [conversion contracts](docs/convert.md).

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
text. The [representative man page](examples/man-tool.1) and its
[tested semantic JSON](examples/man-tool.semantic.json) demonstrate the supported
subset. See [semantic contracts and scenarios](docs/man.md) for details.

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
instructions](docs/ast.md) and the executable `ExampleParseAST` for details.

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

## Supported Scope & Honest Boundaries

- **Syntax & AST Layer**: Dot control lines, apostrophe control lines, line arguments, quotes, escapes, preprocessor blocks, macro definitions, comments, and unknown controls.
- **Macro Packages**: generated `man` and `ms` types support package disambiguation; unsupported syntax produces AST diagnostics.
- **Requests**: 179 distinct roff requests are typed and dispatched.
- **Semantic Evaluation (Opt-in)**: Core register arithmetic (`.nr`, `.rr`), strings (`.ds`, `.as`), conditionals (`.if`, `.ie`, `.el`), and macro definitions (`.de`).
- **Headless CLI**: `cmd/gtroff` provides syntax validation, source-aware AST export, selection, help and version output; `cmd/lex` remains available.
- **Semantic man model**: `man.Build` provides flowing/styled content, definition lists, preformatted text, references, provenance, selection and canonical semantic comparison.
- **HTML export**: `gtroff render --format html` exports whole/selected semantic content with self-contained CSS/fonts and safe atomic file output.
- **Semantic conversion**: `gtroff convert` maps a documented HTML/man subset in both directions, with canonical escaping and semantic loop tests.
- **Image export**: paginated PNG/JPEG and bounded single-image output use bundled Go fonts and pure-Go measurement/painting.
- **HTTP library**: `gtroff serve --root DIR` provides read-only catalogue/search/page navigation with explicit filesystem roots and optional system search.
- **Planned**: optional viewer and release integration remain tracked in #26, #28 and #29.
- **Parser dependencies**: #3–#17 remain open contracts, including include safety, full argument/document fidelity, evaluation, and generation coverage. The AST never follows includes. See [specific dependencies](docs/ast.md#parser-dependencies).

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