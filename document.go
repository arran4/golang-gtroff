package groff

import (
	"errors"
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

// DocumentToken couples a token with a half-open source span. RawSource contains
// exact consumed bytes, including line endings. RawLine retains its legacy line
// text/canonical-block behavior; Token.String is canonical formatting.
type DocumentToken struct {
	Block     *BlockSource
	Token     Token
	Span      Span
	RawLine   string
	RawSource string
}

// Diagnostic represents an error or warning encountered during document parsing.
type Diagnostic struct {
	Message  string
	Position Position
	Severity string // "error" or "warning"
}

// Document contains structured tokens, diagnostics and captured source, including
// partial results when parsing returns an error.
type Document struct {
	Tokens         []DocumentToken
	Diagnostics    []Diagnostic
	SourceName     string
	SourceComplete bool // True only when this source's reader reported EOF.
	Includes       []DocumentInclude
	source         string
}

// DocumentInclude retains the child document even when explicit inclusion fails
// after reading partial source. Directive belongs to the parent's source.
type DocumentInclude struct {
	Directive Span
	Document  *Document
}

// Source returns bytes from validated Read results from this document's reader.
// An invalid reader count cannot establish source bytes for that call. On a terminal
// failure, buffered but unparsed bytes remain visible as an opaque tail token.
// Included sources are separate documents in Includes.
func (d *Document) Source() string { return d.source }

// ParserOptions configures document parsing behaviour.
type ParserOptions struct {
	ASTDepth          int // ParseAST only
	ASTBytes          int // ParseAST only
	ASTNodes          int // ParseAST only
	DocumentBytes     int // ParseDocument only; shared across explicit includes
	Package           string
	Strict            bool
	SourceName        string
	MaxLineLength     int
	MaxLines          int
	MaxIncludeDepth   int // default 10, capped at 128; ParseDocument only
	IncludeResolver   IncludeResolver
	Registry          *Registry
	CustomHandlers    []CustomHandler
	BlockHandlers     []BlockHandler
	BlockNameResolver BlockNameResolver
	MaxBlockDepth     int
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

// WithDocumentBytes bounds reader bytes across a document and explicit includes.
// Default 8 MiB, maximum 128 MiB; non-positive values use the default. One extra
// byte may be read to detect exhaustion and remains available in Source.
func WithDocumentBytes(n int) Option { return func(o *ParserOptions) { o.DocumentBytes = n } }

func WithIncludeResolver(resolver IncludeResolver) Option {
	return func(o *ParserOptions) { o.IncludeResolver = resolver }
}

// WithMaxIncludeDepth limits nested includes, including for custom resolvers.
// Non-positive values use 10; values above 128 are capped. Line budgets apply
// separately to each source. ParseAST never executes includes.
func WithMaxIncludeDepth(n int) Option {
	return func(o *ParserOptions) { o.MaxIncludeDepth = n }
}

// WithRegistry selects custom handlers for this parse instead of the global
// registry. An empty NewRegistry isolates parsing from global handlers; nil uses
// the legacy global default. Registrations are snapshotted when parsing starts.
func WithRegistry(r *Registry) Option {
	return func(o *ParserOptions) { o.Registry = r }
}

// WithCustomHandler adds a parse-local handler to the selected registry snapshot.
// It does not register globally or mutate a registry supplied via WithRegistry.
func WithCustomHandler(h CustomHandler) Option {
	return func(o *ParserOptions) { o.CustomHandlers = append(o.CustomHandlers, h) }
}

// WithBlockHandler adds a parse-local, parser-bounded document/AST block handler.
func WithBlockHandler(h BlockHandler) Option {
	return func(o *ParserOptions) { o.BlockHandlers = append(o.BlockHandlers, h) }
}

// WithBlockNameResolver supplies explicit state for indirect end-name references.
// It is inherited by includes; it never enables macro evaluation or file access.
func WithBlockNameResolver(resolver BlockNameResolver) Option {
	return func(o *ParserOptions) { o.BlockNameResolver = resolver }
}

// WithMaxBlockDepth bounds declared nested block capture (default/max 128).
// ParseAST additionally applies its existing structural ASTDepth limit.
func WithMaxBlockDepth(n int) Option { return func(o *ParserOptions) { o.MaxBlockDepth = n } }

func blockDepthLimit(n int) int {
	if n <= 0 || n > 128 {
		return 128
	}
	return n
}

// ParseDocument reads roff tokens and optionally expands explicitly resolved
// includes. RawSource and Source retain read bytes independently of canonical
// formatting. Strict syntax failures and terminal input failures return a non-nil
// partial document; input failures always return an error in either mode.
func ParseDocument(r io.Reader, opts ...Option) (*Document, error) {
	options := ParserOptions{
		SourceName:    "<input>",
		MaxLineLength: 64 * 1024,
		MaxLines:      1000000,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.MaxIncludeDepth <= 0 {
		options.MaxIncludeDepth = 10
	}
	if options.MaxIncludeDepth > 128 {
		options.MaxIncludeDepth = 128
	}
	options.Registry = registryForOptions(options)
	options.MaxBlockDepth = blockDepthLimit(options.MaxBlockDepth)
	if options.DocumentBytes <= 0 {
		options.DocumentBytes = 8 << 20
	}
	if options.DocumentBytes > 128<<20 {
		options.DocumentBytes = 128 << 20
	}
	return parseDocument(r, options, &includeState{active: make(map[string]bool), budget: &documentByteBudget{limit: options.DocumentBytes}})
}

type includeState struct {
	depth  int
	active map[string]bool
	budget *documentByteBudget
}

func parseDocument(r io.Reader, options ParserOptions, includes *includeState) (doc *Document, resultErr error) {
	doc = &Document{SourceName: options.SourceName}
	if nilInterface(r) {
		inputErr := &DocumentInputError{Position: Position{Filename: options.SourceName, Line: 1, Column: 1}, Err: fmt.Errorf("nil document reader")}
		doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: inputErr.Err.Error(), Position: inputErr.Position, Severity: "error"})
		return doc, inputErr
	}
	stream := newDocumentStream(r, options, includes.budget)
	defer func() {
		if tail := stream.drainTail(); tail.raw != "" {
			doc.Tokens = append(doc.Tokens, opaqueDocumentToken(tail))
		}
		doc.source = stream.input.source.String()
		doc.SourceComplete = stream.input.terminal == io.EOF
		// A reader may return data and an error together before buffering exposes the
		// error to a strict syntax failure. Never hide that already observed failure.
		if pending := stream.input.terminal; pending != nil && pending != io.EOF && !errors.Is(resultErr, pending) {
			inputErr := &DocumentInputError{Position: stream.position, Err: pending}
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: pending.Error(), Position: inputErr.Position, Severity: "error"})
			resultErr = errors.Join(resultErr, inputErr)
		}
	}()
	failInput := func(err error) error {
		var inputErr *DocumentInputError
		if errors.As(err, &inputErr) {
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: inputErr.Err.Error(), Position: inputErr.Position, Severity: "error"})
		}
		return err
	}
	for {
		line, readErr := stream.next()
		if readErr != nil && readErr != io.EOF {
			if line.raw != "" {
				doc.Tokens = append(doc.Tokens, opaqueDocumentToken(line))
			}
			return doc, failInput(readErr)
		}
		if line.raw == "" && readErr == io.EOF {
			return doc, nil
		}
		body := documentLineBody(line.raw)
		cmd, args := "", ""
		if len(body) > 0 && (body[0] == '.' || body[0] == '\'') {
			cmd, args = splitCommandAndArgs(body)
			cmd = "." + cmd[1:]
		}
		var selectedLine CustomHandler
		var blockHandler BlockHandler
		var selectionErr error
		if cmd != "" {
			originalCmd, _ := splitCommandAndArgs(body)
			selectedLine, blockHandler, selectionErr = options.Registry.documentHandler(originalCmd, options.Package)
		}
		var capture *blockCapture
		if selectionErr == nil && blockHandler != nil {
			blockHandler = withBlockNameResolver(blockHandler, options.BlockNameResolver)
			originalCmd, _ := splitCommandAndArgs(body)
			capture, selectionErr = beginBlockCapture(blockHandler, BlockHeader{Command: originalCmd, Args: args, Package: options.Package, Line: BlockLine{RawSource: line.raw, Span: line.span}}, options.MaxBlockDepth)
		}
		if selectionErr == nil && blockHandler == nil && selectedLine == nil && (generatedBlockClosing(cmd) || cmd == "..") {
			selectionErr = blockSyntax(line.span.Start, fmt.Errorf("unmatched block delimiter %s", cmd))
		}
		if selectionErr != nil {
			doc.Tokens = append(doc.Tokens, opaqueDocumentToken(line))
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: selectionErr.Error(), Position: line.span.Start, Severity: "error"})
			if options.Strict {
				return doc, documentSyntaxError(line.span.Start, selectionErr)
			}
			continue
		}
		if capture != nil {
			token, blockErr := readCapturedBlock(blockHandler, capture, stream)
			source := capture.source.clone()
			doc.Tokens = append(doc.Tokens, DocumentToken{Token: token, Span: source.Span(), RawLine: token.String(), RawSource: source.Source(), Block: &source})
			if blockErr != nil {
				var inputErr *DocumentInputError
				if errors.As(blockErr, &inputErr) {
					return doc, failInput(blockErr)
				}
				position := line.span.Start
				var syntax *BlockSyntaxError
				if errors.As(blockErr, &syntax) {
					position = syntax.Position
				}
				doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: blockErr.Error(), Position: position, Severity: "error"})
				if options.Strict {
					return doc, documentSyntaxError(position, blockErr)
				}
			}
			continue
		}
		if (cmd == ".so" || cmd == ".mso") && options.IncludeResolver != nil && args != "" && selectedLine == nil {
			included, includeErr := resolveInclude(args, options, includes)
			if included != nil {
				doc.Includes = append(doc.Includes, DocumentInclude{Directive: line.span, Document: included})
				doc.Diagnostics = append(doc.Diagnostics, included.Diagnostics...)
			}
			if includeErr == nil {
				doc.Tokens = append(doc.Tokens, included.Tokens...)
				if readErr == io.EOF {
					return doc, nil
				}
				continue
			}
			severity := "warning"
			var inputErr *DocumentInputError
			inputFailure := errors.As(includeErr, &inputErr)
			if options.Strict || inputFailure {
				severity = "error"
			}
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: includeErr.Error(), Position: line.span.Start, Severity: severity})
			if options.Strict || inputFailure {
				doc.Tokens = append(doc.Tokens, opaqueDocumentToken(line))
				return doc, documentSyntaxError(line.span.Start, includeErr)
			}
		}
		token, parseErr := parseLineWithRegistry(body, options.Package, !options.Strict, options.Registry)
		if parseErr != nil {
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{Message: parseErr.Error(), Position: line.span.Start, Severity: "error"})
			doc.Tokens = append(doc.Tokens, opaqueDocumentToken(line))
			if options.Strict {
				return doc, documentSyntaxError(line.span.Start, parseErr)
			}
		} else {
			doc.Tokens = append(doc.Tokens, DocumentToken{Token: token, Span: line.span, RawLine: body, RawSource: line.raw})
		}
		if readErr == io.EOF {
			return doc, nil
		}
	}
}

func documentSyntaxError(position Position, err error) error {
	return fmt.Errorf("%s:%d:%d: %w", position.Filename, position.Line, position.Column, err)
}

func opaqueDocumentToken(line documentLine) DocumentToken {
	body := documentLineBody(line.raw)
	return DocumentToken{Token: &TokenUnknownControl{RawLine: body}, Span: line.span, RawLine: body, RawSource: line.raw}
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

func resolveInclude(args string, opts ParserOptions, state *includeState) (doc *Document, err error) {
	cleanFile := strings.TrimSpace(args)
	if state.active[cleanFile] {
		return nil, fmt.Errorf("include cycle detected: %s", cleanFile)
	}
	if state.depth >= opts.MaxIncludeDepth {
		return nil, fmt.Errorf("maximum include depth exceeded (%d)", opts.MaxIncludeDepth)
	}
	reader, err := opts.IncludeResolver.Resolve(cleanFile)
	if err != nil {
		return nil, err
	}

	if nilInterface(reader) {
		return nil, fmt.Errorf("include resolver returned a nil reader: %s", cleanFile)
	}
	state.active[cleanFile] = true
	state.depth++
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			err = errors.Join(err, closer.Close())
		}
		delete(state.active, cleanFile)
		state.depth--
		if err != nil {
			err = fmt.Errorf("include %q: %w", cleanFile, err)
		}
	}()
	opts.SourceName = cleanFile
	return parseDocument(reader, opts, state)
}
