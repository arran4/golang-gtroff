package groff

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Position describes a point in a source document.
type Position struct {
	Filename string
	Offset   int // 0-indexed byte offset
	Line     int // 1-indexed line number
	Column   int // 1-indexed column number
}

// Span represents a start and end range in a source document.
type Span struct {
	Start Position
	End   Position
}

// DocumentToken couples a parsed Token with its source span and verbatim line text.
type DocumentToken struct {
	Token   Token
	Span    Span
	RawLine string
}

// Diagnostic represents an error or warning encountered during document parsing.
type Diagnostic struct {
	Message  string
	Position Position
	Severity string // "error" or "warning"
}

// Document is the complete parsed document containing structured tokens and diagnostics.
type Document struct {
	Tokens      []DocumentToken
	Diagnostics []Diagnostic
}

// ParserOptions configures document parsing behaviour.
type ParserOptions struct {
	Package         string
	Strict          bool
	SourceName      string
	MaxLineLength   int
	MaxLines        int
	IncludeResolver IncludeResolver
	CustomHandlers  []CustomHandler
}

// Option configures document parsing.
type Option func(*ParserOptions)

func WithPackage(pkg string) Option {
	return func(o *ParserOptions) { o.Package = pkg }
}

func WithStrict(strict bool) Option {
	return func(o *ParserOptions) { o.Strict = strict }
}

func WithSourceName(name string) Option {
	return func(o *ParserOptions) { o.SourceName = name }
}

func WithMaxLineLength(n int) Option {
	return func(o *ParserOptions) { o.MaxLineLength = n }
}

func WithMaxLines(n int) Option {
	return func(o *ParserOptions) { o.MaxLines = n }
}

func WithIncludeResolver(resolver IncludeResolver) Option {
	return func(o *ParserOptions) { o.IncludeResolver = resolver }
}

func WithCustomHandler(h CustomHandler) Option {
	return func(o *ParserOptions) { o.CustomHandlers = append(o.CustomHandlers, h) }
}

// ParseDocument reads and tokenizes an entire groff document from r without loss.
func ParseDocument(r io.Reader, opts ...Option) (*Document, error) {
	options := ParserOptions{
		SourceName:    "<input>",
		MaxLineLength: 64 * 1024,
		MaxLines:      1000000,
	}
	for _, opt := range opts {
		opt(&options)
	}

	doc := &Document{}
	br := bufio.NewReader(r)

	offset := 0
	lineNum := 1

	for {
		if options.MaxLines > 0 && lineNum > options.MaxLines {
			return nil, fmt.Errorf("line budget exceeded (%d lines)", options.MaxLines)
		}

		line, readErr := br.ReadString('\n')
		if len(line) == 0 && readErr == io.EOF {
			break
		}

		if options.MaxLineLength > 0 && len(line) > options.MaxLineLength {
			return nil, fmt.Errorf("line length budget exceeded on line %d (%d bytes)", lineNum, len(line))
		}

		lineStartPos := Position{
			Filename: options.SourceName,
			Offset:   offset,
			Line:     lineNum,
			Column:   1,
		}

		lineBytes := len(line)
		offset += lineBytes

		// Strip trailing newline characters from line for parsing
		trimmedLine := strings.TrimRight(line, "\r\n")

		lineEndPos := Position{
			Filename: options.SourceName,
			Offset:   offset,
			Line:     lineNum,
			Column:   len([]rune(trimmedLine)) + 1,
		}

		span := Span{
			Start: lineStartPos,
			End:   lineEndPos,
		}

		// Handle multiline requests & preprocessors
		if strings.HasPrefix(trimmedLine, ".") || strings.HasPrefix(trimmedLine, "'") {
			cmd, args := splitCommandAndArgs(trimmedLine)

			// 1. Preprocessor blocks: .TS, .EQ, .PS, .[
			if cmd == ".TS" || cmd == ".EQ" || cmd == ".PS" || cmd == ".[" {
				blockToken, blockSpan, err := readPreprocessorBlock(cmd, args, line, br, &offset, &lineNum, lineStartPos, options)
				if err != nil {
					if options.Strict {
						return nil, err
					}
					doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
						Message:  err.Error(),
						Position: lineStartPos,
						Severity: "error",
					})
				}
				doc.Tokens = append(doc.Tokens, DocumentToken{
					Token:   blockToken,
					Span:    blockSpan,
					RawLine: blockToken.String(),
				})
				if _, fatal := err.(*blockInputError); fatal {
					return doc, err
				}
				lineNum++
				if readErr == io.EOF {
					break
				}
				continue
			}

			// 2. Multiline macro definitions (.de, .am)
			if cmd == ".de" || cmd == ".am" {
				defToken, defSpan, err := readMacroDefinition(cmd, args, line, br, &offset, &lineNum, lineStartPos, options)
				if err != nil {
					if options.Strict {
						return nil, err
					}
					doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
						Message:  err.Error(),
						Position: lineStartPos,
						Severity: "error",
					})
				}
				doc.Tokens = append(doc.Tokens, DocumentToken{
					Token:   defToken,
					Span:    defSpan,
					RawLine: defToken.String(),
				})
				if _, fatal := err.(*blockInputError); fatal {
					return doc, err
				}
				lineNum++
				if readErr == io.EOF {
					break
				}
				continue
			}

			// 3. Ignored block (.ig)
			if cmd == ".ig" {
				igToken, igSpan, err := readIgnoredBlock(args, line, br, &offset, &lineNum, lineStartPos, options)
				if err != nil {
					if options.Strict {
						return nil, err
					}
					doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
						Message:  err.Error(),
						Position: lineStartPos,
						Severity: "error",
					})
				}
				doc.Tokens = append(doc.Tokens, DocumentToken{
					Token:   igToken,
					Span:    igSpan,
					RawLine: igToken.String(),
				})
				if _, fatal := err.(*blockInputError); fatal {
					return doc, err
				}
				lineNum++
				if readErr == io.EOF {
					break
				}
				continue
			}

			// 4. Source directive (.so, .mso)
			if (cmd == ".so" || cmd == ".mso") && options.IncludeResolver != nil && args != "" {
				incTokens, err := resolveInclude(args, options)
				if err != nil {
					if options.Strict {
						return nil, err
					}
					doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
						Message:  err.Error(),
						Position: lineStartPos,
						Severity: "warning",
					})
				} else {
					doc.Tokens = append(doc.Tokens, incTokens...)
					lineNum++
					if readErr == io.EOF {
						break
					}
					continue
				}
			}
		}

		// Standard line parsing
		var tok Token
		var parseErr error

		if options.Strict {
			tok, parseErr = ParseLineWithPackage(trimmedLine, options.Package)
		} else {
			tok, parseErr = ParseLinePermissive(trimmedLine, options.Package)
		}

		if parseErr != nil {
			if options.Strict {
				return nil, fmt.Errorf("line %d: %w", lineNum, parseErr)
			}
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
				Message:  parseErr.Error(),
				Position: lineStartPos,
				Severity: "error",
			})
			tok = &TokenUnknownControl{
				ControlChar: rune(trimmedLine[0]),
				Name:        trimmedLine,
				RawLine:     trimmedLine,
			}
		}

		doc.Tokens = append(doc.Tokens, DocumentToken{
			Token:   tok,
			Span:    span,
			RawLine: trimmedLine,
		})

		lineNum++
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read line %d: %w", lineNum, readErr)
		}
	}

	return doc, nil
}

func splitCommandAndArgs(line string) (string, string) {
	wsIdx := -1
	for i := 1; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			wsIdx = i
			break
		}
	}
	if wsIdx == -1 {
		return line, ""
	}
	return line[:wsIdx], strings.TrimLeft(line[wsIdx:], " \t")
}

// blockInputError stops document parsing after an I/O or resource failure.
type blockInputError struct{ error }

func (e *blockInputError) Unwrap() error { return e.error }

func recoverBlock(raw string, start Position, err error) (Token, Span, error) {
	return &TokenUnknownControl{RawLine: raw}, Span{Start: start, End: advanceSourcePosition(start, raw)}, err
}

func readPreprocessorBlock(cmd, args, header string, br *bufio.Reader, offset *int, lineNum *int, startPos Position, opts ParserOptions) (Token, Span, error) {
	var raw strings.Builder
	raw.WriteString(header)
	endMarker := ""
	switch cmd {
	case ".TS":
		endMarker = ".TE"
	case ".EQ":
		endMarker = ".EN"
	case ".PS":
		endMarker = ".PE"
	case ".[":
		endMarker = ".]"
	}

	var lines []string
	for {
		line, readErr := br.ReadString('\n')
		raw.WriteString(line)
		if len(line) == 0 && readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated block %s (expected %s)", cmd, endMarker))
		}
		*offset += len(line)
		*lineNum++

		if readErr != nil && readErr != io.EOF {
			return recoverBlock(raw.String(), startPos, &blockInputError{readErr})
		}
		if (opts.MaxLines > 0 && *lineNum > opts.MaxLines) || (opts.MaxLineLength > 0 && len(line) > opts.MaxLineLength) {
			return recoverBlock(raw.String(), startPos, &blockInputError{fmt.Errorf("block input budget exceeded on line %d", *lineNum)})
		}
		trimmed := strings.TrimRight(line, "\r\n")
		checkCmd, _ := splitCommandAndArgs(trimmed)
		if checkCmd == endMarker {
			// Found end delimiter
			endPos := Position{
				Filename: opts.SourceName,
				Offset:   *offset,
				Line:     *lineNum,
				Column:   len([]rune(trimmed)) + 1,
			}
			span := Span{Start: startPos, End: endPos}
			switch cmd {
			case ".TS":
				return &TokenTableBlock{Options: args, Lines: lines}, span, nil
			case ".EQ":
				return &TokenEquationBlock{Args: args, Lines: lines}, span, nil
			case ".PS":
				return &TokenPictureBlock{Args: args, Lines: lines}, span, nil
			case ".[":
				return &TokenReferenceBlock{Lines: lines}, span, nil
			}
		}

		lines = append(lines, trimmed)
		if readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated block %s (expected %s)", cmd, endMarker))
		}
	}
}

func readMacroDefinition(cmd, args, header string, br *bufio.Reader, offset *int, lineNum *int, startPos Position, opts ParserOptions) (Token, Span, error) {
	var raw strings.Builder
	raw.WriteString(header)
	parts := strings.Fields(args)
	name := ""
	endDelim := ".."
	if len(parts) > 0 {
		name = parts[0]
	}
	if len(parts) > 1 {
		endDelim = "." + parts[1]
	}

	var body []string
	for {
		line, readErr := br.ReadString('\n')
		raw.WriteString(line)
		if len(line) == 0 && readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated macro definition %s", cmd))
		}
		*offset += len(line)
		*lineNum++

		if readErr != nil && readErr != io.EOF {
			return recoverBlock(raw.String(), startPos, &blockInputError{readErr})
		}
		if (opts.MaxLines > 0 && *lineNum > opts.MaxLines) || (opts.MaxLineLength > 0 && len(line) > opts.MaxLineLength) {
			return recoverBlock(raw.String(), startPos, &blockInputError{fmt.Errorf("block input budget exceeded on line %d", *lineNum)})
		}
		trimmed := strings.TrimRight(line, "\r\n")
		checkCmd, _ := splitCommandAndArgs(trimmed)
		if checkCmd == endDelim {
			endPos := Position{
				Filename: opts.SourceName,
				Offset:   *offset,
				Line:     *lineNum,
				Column:   len([]rune(trimmed)) + 1,
			}
			span := Span{Start: startPos, End: endPos}
			endArg := ""
			if len(parts) > 1 {
				endArg = parts[1]
			}
			if cmd == ".am" {
				return &TokenMacroAppend{Name: name, End: endArg, Body: body}, span, nil
			}
			return &TokenMacroDefinition{Name: name, End: endArg, Body: body}, span, nil
		}

		body = append(body, trimmed)
		if readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated macro definition %s", cmd))
		}
	}
}

func readIgnoredBlock(args, header string, br *bufio.Reader, offset *int, lineNum *int, startPos Position, opts ParserOptions) (Token, Span, error) {
	var raw strings.Builder
	raw.WriteString(header)
	endDelim := ".."
	endArg := strings.TrimSpace(args)
	if endArg != "" {
		endDelim = "." + endArg
	}

	var body []string
	for {
		line, readErr := br.ReadString('\n')
		raw.WriteString(line)
		if len(line) == 0 && readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated ignored block (.ig)"))
		}
		*offset += len(line)
		*lineNum++

		if readErr != nil && readErr != io.EOF {
			return recoverBlock(raw.String(), startPos, &blockInputError{readErr})
		}
		if (opts.MaxLines > 0 && *lineNum > opts.MaxLines) || (opts.MaxLineLength > 0 && len(line) > opts.MaxLineLength) {
			return recoverBlock(raw.String(), startPos, &blockInputError{fmt.Errorf("block input budget exceeded on line %d", *lineNum)})
		}
		trimmed := strings.TrimRight(line, "\r\n")
		checkCmd, _ := splitCommandAndArgs(trimmed)
		if checkCmd == endDelim {
			endPos := Position{
				Filename: opts.SourceName,
				Offset:   *offset,
				Line:     *lineNum,
				Column:   len([]rune(trimmed)) + 1,
			}
			span := Span{Start: startPos, End: endPos}
			return &TokenIgnoredBlock{End: endArg, Body: body}, span, nil
		}

		body = append(body, trimmed)
		if readErr == io.EOF {
			return recoverBlock(raw.String(), startPos, fmt.Errorf("unterminated ignored block (.ig)"))
		}
	}
}

func resolveInclude(args string, opts ParserOptions) ([]DocumentToken, error) {
	cleanFile := strings.TrimSpace(args)
	reader, err := opts.IncludeResolver.Resolve(cleanFile)
	if err != nil {
		return nil, err
	}

	subDoc, err := ParseDocument(reader,
		WithPackage(opts.Package),
		WithStrict(opts.Strict),
		WithSourceName(cleanFile),
		WithIncludeResolver(opts.IncludeResolver),
	)
	if err != nil {
		return nil, err
	}
	return subDoc.Tokens, nil
}
