# Test Data

This directory contains test man page files used for testing the groff parser.

## Test Files

### Custom Test Files

- **simple.1** - A simple man page demonstrating basic macros like TH, SH, SY/YS, OP, TP, B, EX/EE, MR, and MT/ME.
  
- **complex.7** - A more complex man page that tests various font style macros (B, I, BI, BR, IB, IR, RB, RI), list formatting (IP, TP), indentation (RS/RE), hyperlinks (UR/UE, MT/ME), and code blocks (EX/EE).

- **minimal.1** - A minimal man page with just the essential TH and SH macros for basic parsing tests.

### Real Man Page Snippets

- **environ_snippet.7** - First 100 lines from the actual `environ(7)` man page, providing a real-world test case with comments, various macros, and typical man page structure.

## Usage

These files are used by `parser_file_test.go` to test:

1. **Complete file parsing** - Verifying that entire man page files can be parsed
2. **Token type verification** - Ensuring specific macros are correctly identified
3. **Real-world compatibility** - Testing against actual man page content
4. **Line-by-line parsing** - Checking that individual lines parse correctly

## Adding New Test Files

When adding new test files:

1. Use appropriate file extensions (.1 for commands, .7 for miscellaneous, etc.)
2. Include a variety of macros to test different parsing scenarios
3. Add corresponding test cases in `parser_file_test.go`
4. Document the file's purpose in this README
