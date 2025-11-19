# golang-gtroff

A Go library for parsing groff/troff documents. This parser focuses on extracting macros and requests from groff documents, particularly those using the `man` and `ms` macro packages.

## Features

- **Macro Parsing**: Parses groff macros and requests into structured Go types
- **Man Page Support**: Comprehensive support for `man` macro package (`.TH`, `.SH`, `.SS`, `.TP`, `.IP`, etc.)
- **MS Macro Support**: Support for `ms` macro package (`.TL`, `.AU`, `.AI`, `.AB`, `.NH`, etc.)
- **Text Extraction**: Primarily designed for extracting text content from groff documents
- **Token-based**: Line-by-line parsing that returns typed tokens for each macro/request
- **Extensible**: Easy to add support for additional macros via code generation

## Installation

```bash
go get github.com/arran4/golang-gtroff
```

## Usage

### Basic Parsing

```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    
    groff "github.com/arran4/golang-gtroff"
)

func main() {
    file, err := os.Open("example.1")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        token, err := groff.ParseLine(line)
        if err != nil {
            log.Printf("Error parsing line: %v", err)
            continue
        }
        
        // Process the token
        switch t := token.(type) {
        case *groff.TokenMacroTH:
            fmt.Printf("Title: %s, Section: %s\n", t.Title, t.Section)
        case *groff.TokenMacroSH:
            fmt.Printf("Section Heading: %s\n", t.Heading)
        case groff.TokenText:
            fmt.Printf("Text: %s\n", t)
        }
    }
}
```

### Using the Command-Line Tool

A simple lexer tool is included for parsing groff files:

```bash
go run ./cmd/lex/main.go example.1
```

## Supported Macros

### Man Macro Package

- **Document Structure**: `.TH`, `.SH`, `.SS`
- **Paragraphs**: `.P`, `.PP`, `.LP`, `.TP`, `.IP`, `.HP`
- **Font Styles**: `.B`, `.I`, `.BI`, `.IB`, `.BR`, `.RB`, `.IR`, `.RI`
- **Text Formatting**: `.SM`, `.SB`, `.EX`, `.EE`
- **Indentation**: `.RS`, `.RE`
- **Hyperlinks**: `.UR`, `.UE`, `.MT`, `.ME`, `.MR`
- **Synopsis**: `.SY`, `.YS`, `.OP`, `.TQ`
- **Miscellaneous**: `.DT`, `.PD`, `.AT`, `.UC`, `.PT`, `.BT`, `.CT`, `.CW`

### MS Macro Package

- **Document Metadata**: `.DA`, `.ND`, `.TL`, `.AU`, `.AI`
- **Abstract**: `.AB`, `.AE`
- **Paragraphs**: `.PP`, `.LP`, `.QP`, `.QE`, `.XP`
- **Headings**: `.NH`
- **Displays**: `.DS`, `.DE`, `.ID`, `.BD`, `.CD`, `.LD`, `.RD`
- **Keeps**: `.KS`, `.KE`, `.KF`
- **Footnotes**: `.FS`, `.FE`
- **Tables/Figures**: `.TS`, `.TE`, `.PS`, `.PE`, `.EQ`, `.EN`
- **Boxes**: `.B1`, `.B2`, `.BX`
- **Columns**: `.1C`, `.2C`, `.MC`
- **Index/TOC**: `.IX`, `.XS`, `.XA`, `.XE`, `.TC`, `.PX`
- **Headers/Footers**: `.HD`, `.EH`, `.OH`, `.EF`, `.OF`

### Other

- **Comments**: `.\"`
- **References**: `.[`, `.]`

## API

### Core Types

```go
type Token interface {
    String() string
}
```

All parsed macros implement the `Token` interface and can be converted back to their string representation.

### Main Functions

- `ParseLine(line string) (Token, error)` - Parse a single line of groff input
- `ArgumentParser(l string) ([]string, int, error)` - Parse macro arguments with quote handling

## Current Limitations

This is a **minimal parser** built for specific purposes. Notable limitations:

- **Escapes**: Escape sequences are not fully processed
- **Registers**: Number registers are ignored
- **Conditionals**: No support for conditional processing (`.if`, `.ie`, `.el`)
- **String Definitions**: String definitions (`.ds`) are not tracked
- **Diversions**: No support for diversions
- **Requests**: Many low-level groff requests are not implemented

**Contributions welcome!** PRs to add missing functionality are appreciated.

## Development

### Code Generation

Many macros are generated from templates. To regenerate:

```bash
go run ./cmd/gen/maingen.go
```

The generator uses:
- `cmd/gen/in.json` - Macro definitions
- `cmd/gen/macros.go.template` - Template for macro structs and parsers
- `cmd/gen/parseline.go.template` - Template for the ParseLine switch cases

### Testing

```bash
go test ./...
```

## License

This project follows the licensing of the source material:

- Documentation derived from the [GNU groff manual](https://www.gnu.org/software/groff/manual/groff.html#Copying-This-Manual) - See manual for terms
- All other code - GPL (to maintain compatibility with groff source material)

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Areas where help is particularly appreciated:

- Adding support for missing macros and requests
- Implementing escape sequence processing
- Adding register tracking
- Improving documentation
- Adding more test cases
- Performance improvements

Please feel free to open issues or submit pull requests.