package groff

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

// WithASTDepth bounds nested block structure. The default is 128; sections and
// line/inline wrappers do not consume this budget. Nonpositive values use default.
func WithASTDepth(n int) Option { return func(o *ParserOptions) { o.ASTDepth = n } }

// WithASTBytes bounds bytes read by ParseAST (default 8 MiB). One extra byte may
// be read to detect overflow. All bytes actually read remain available in Source.
func WithASTBytes(n int) Option { return func(o *ParserOptions) { o.ASTBytes = n } }

// WithASTNodes bounds tree size (default 100000). On exhaustion the unread syntax
// is represented by one opaque tail node, and ParseAST returns an error.
func WithASTNodes(n int) Option { return func(o *ParserOptions) { o.ASTNodes = n } }

// ParseAST builds a source-preserving syntax tree without executing macros or
// resolving includes. It uses generated/custom line tokens and ParseInline, while
// owning physical source boundaries independently of legacy Document formatting.
//
// Syntax errors produce diagnostics and preserved nodes. Strict mode also returns
// an error, with the complete tree. I/O/budget errors always return a non-nil error
// and the source/tree collected so far. IncludeResolver is deliberately ignored.
func ParseAST(r io.Reader, opts ...Option) (*AST, error) {
	o := ParserOptions{SourceName: "<input>", MaxLineLength: 64 * 1024, MaxLines: 100000, ASTDepth: 128, ASTBytes: 8 << 20, ASTNodes: 100000}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	registry := registryForOptions(o)
	o.MaxBlockDepth = blockDepthLimit(o.MaxBlockDepth)
	if o.ASTDepth <= 0 {
		o.ASTDepth = 128
	}
	if o.ASTBytes <= 0 {
		o.ASTBytes = 8 << 20
	}
	if o.ASTNodes <= 0 {
		o.ASTNodes = 100000
	}
	data, complete, sourceErr := readASTInput(r, o)
	var readErr error
	a := &AST{source: string(data), sourceComplete: complete}
	root := &ASTNode{tree: a, kind: ASTDocument, span: Span{Start: Position{Filename: o.SourceName, Line: 1, Column: 1}}}
	a.root = root
	pos := root.span.Start
	parent := root
	var blocks []*ASTNode
	captures := map[*ASTNode]*astBlockCapture{}
	var section *ASTNode
	count := 1
	var failure error
	add := func(p *ASTNode, kind ASTKind, start Position) *ASTNode {
		n := &ASTNode{tree: a, parent: p, kind: kind, sibling: len(p.children), span: Span{Start: start, End: start}}
		p.children = append(p.children, n)
		count++
		return n
	}
	diagnosticAt := func(n *ASTNode, message, severity string, position Position) {
		d := Diagnostic{Message: message, Position: position, Severity: severity}
		n.diagnostics = append(n.diagnostics, d)
		a.diagnostics = append(a.diagnostics, d)
		if severity == "error" && failure == nil {
			failure = fmt.Errorf("%s:%d:%d: %s", d.Position.Filename, d.Position.Line, d.Position.Column, message)
		}
	}
	diagnostic := func(n *ASTNode, message, severity string) { diagnosticAt(n, message, severity, n.span.Start) }

	for off, lineNumber := 0, 1; off < len(a.source); lineNumber++ {
		end := strings.IndexByte(a.source[off:], '\n')
		if end < 0 {
			end = len(a.source)
		} else {
			end += off + 1
		}
		raw := a.source[off:end]
		body := strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
		lineEnd := advanceSourcePosition(pos, raw)
		// Reserve enough space for recovery; a single opaque tail preserves all bytes.
		if (o.MaxLines > 0 && lineNumber > o.MaxLines) || count+4 >= o.ASTNodes {
			tail := add(parent, ASTLine, pos)
			tail.span.End = advanceSourcePosition(pos, a.source[off:])
			tail.attrs.Opaque = true
			tail.text = a.source[off:]
			for _, state := range captures {
				state.capture.source.Tail = BlockLine{RawSource: a.source[off:], Span: Span{Start: pos, End: tail.span.End}}
			}
			diagnostic(tail, "AST line/node budget exceeded; remaining source preserved as opaque text", "error")
			readErr = fmt.Errorf("AST line/node budget exceeded")
			pos = tail.span.End
			break
		}
		cmd, args := "", ""
		if len(body) > 0 && (body[0] == '.' || body[0] == '\'') {
			cmd, args = splitCommandAndArgs(body)
			cmd = "." + cmd[1:]
		}
		// Only the active structural block decides closure. Other active macro
		// captures retain every physical line, including nested syntax, as data.
		opaque, closing := false, false
		var top *ASTNode
		if len(blocks) > 0 {
			top = blocks[len(blocks)-1]
			closing = cmd == top.attrs.EndMarker
			opaque = top.attrs.Opaque && !closing
		}
		physical := BlockLine{RawSource: raw, Span: Span{Start: pos, End: lineEnd}}
		if state := captures[top]; state != nil {
			if !state.machine {
				if closing {
					state.capture.source.Terminator = physical
					state.capture.source.Closed = true
				}
			} else if state.stopped {
				state.capture.source.Body = append(state.capture.source.Body, physical)
				closing = false
				opaque = true
			} else {
				if err := state.capture.consume(physical); err != nil {
					state.stopped = true
					readErr = err
					message := err.Error()
					var inputErr *DocumentInputError
					if errors.As(err, &inputErr) {
						message = inputErr.Err.Error()
					}
					diagnosticAt(top, message, "error", pos)
				}
				closing = state.capture.source.Closed
				opaque = !closing
			}
		}

		var selectedLine CustomHandler
		var selectedBlock BlockHandler
		var candidate *blockCapture
		var headerErr error
		originalCmd, _ := splitCommandAndArgs(body)
		if cmd != "" && !closing && !opaque && (o.MaxLineLength <= 0 || len(raw) <= o.MaxLineLength) {
			selectedLine, selectedBlock, headerErr = registry.documentHandler(originalCmd, o.Package)
			if headerErr == nil && selectedBlock != nil {
				selectedBlock = withBlockNameResolver(selectedBlock, o.BlockNameResolver)
				candidate, headerErr = beginBlockCapture(selectedBlock, BlockHeader{Command: originalCmd, Args: args, Package: o.Package, Line: physical}, o.MaxBlockDepth)
			}
		}

		if len(blocks) == 0 && !opaque && selectedLine == nil && selectedBlock == nil && headerErr == nil && o.Package != "ms" && o.Package != "roff" && (cmd == ".SH" || cmd == ".SS") {
			lex, _, e := LexArguments(args)
			if e == nil && len(lex) > 0 {
				heading := argumentASTText(lex)
				if cmd == ".SH" {
					section = add(root, ASTSection, pos)
					section.attrs.Heading = heading
					parent = section
				} else {
					p := section
					if p == nil {
						p = root
					}
					subsection := add(p, ASTSubsection, pos)
					subsection.attrs.Heading = heading
					parent = subsection
				}
			}
		}
		marker, blockOpaque := astEndMarker(cmd, args, o.Package)
		if selectedLine != nil || headerErr != nil {
			marker = ""
		}
		if candidate != nil {
			if len(candidate.source.Rule.Ends) > 0 {
				marker = candidate.source.Rule.Ends[0]
			}
			blockOpaque = true
		}
		// Distinct explicit macro terminators describe nested syntax only. No
		// user callbacks run while inspecting an opaque definition body.
		if opaque && top != nil && top.attrs.UnresolvedEnd == "" && isMacroDefinitionBlock(top.attrs.Name) && (isMacroDefinitionBlock(cmd) || cmd == ".ig") {
			selectedBlock = generatedBlockHandler(cmd)
			if selectedBlock != nil {
				candidate, headerErr = beginBlockCapture(selectedBlock, BlockHeader{Command: originalCmd, Args: args, Package: o.Package, Line: physical}, o.MaxBlockDepth)
				if headerErr == nil && candidate.source.Rule.UnresolvedEnd == "" {
					marker = candidate.source.Rule.Ends[0]
					blockOpaque = true
				} else {
					marker = ""
					candidate = nil
				}
			}
		}
		opening := (marker != "" || candidate != nil && !opaque) && !closing && (!opaque || ((isMacroDefinitionBlock(cmd) || cmd == ".ig") && top != nil && marker != top.attrs.EndMarker && isMacroDefinitionBlock(top.attrs.Name)))

		if opening && len(blocks) >= o.ASTDepth {
			opening = false
			diagnostic(parent, "AST block depth budget exceeded; opener preserved as opaque text", "error")
			readErr = fmt.Errorf("AST depth budget exceeded")
			opaque = true
		}
		if opening {
			b := add(parent, ASTBlock, pos)
			b.attrs = ASTAttributes{Name: cmd, EndMarker: marker, Opaque: blockOpaque}
			if candidate != nil {
				_, builtin := selectedBlock.(builtinBlockHandler)
				b.attrs.UnresolvedEnd = candidate.source.Rule.UnresolvedEnd
				captures[b] = &astBlockCapture{handler: selectedBlock, capture: candidate, custom: !builtin, machine: !builtin || len(candidate.source.Rule.Nested) > 0 || candidate.source.Rule.UnresolvedEnd != ""}
			}

			blocks = append(blocks, b)
			parent = b
			opaque = false
		}
		n := add(parent, ASTLine, pos)
		n.span.End = lineEnd
		n.attrs.Name = cmd
		n.attrs.Opaque = opaque
		if !closing && !opaque && selectedLine == nil && selectedBlock == nil && astClosingMarker(cmd, o.Package) {
			diagnostic(n, "unmatched block delimiter "+cmd, "error")
		}
		if closing {
			n.attrs.TokenKind = KindControl
			n.text = ""
		} else if o.MaxLineLength > 0 && len(raw) > o.MaxLineLength {
			n.attrs.Opaque = true
			n.text = raw
			diagnostic(n, "AST line length budget exceeded; line preserved as opaque text", "error")
			readErr = fmt.Errorf("AST line length budget exceeded")
		} else if opaque {
			n.attrs.TokenKind = KindText
			n.text = raw
		} else if !utf8.ValidString(body) {
			n.attrs.Opaque = true
			n.text = raw
			diagnostic(n, "invalid UTF-8 preserved as opaque source", "error")
		} else {
			var tok Token
			var err error
			if headerErr != nil {
				err = headerErr
			} else if opening && candidate != nil {
				if _, builtin := selectedBlock.(builtinBlockHandler); builtin {
					tok, err = parseLineWithRegistry(body, o.Package, false, NewRegistry())
				}
				if tok == nil || err != nil {
					tok = &TokenBlockHeader{Command: originalCmd, Args: args, Package: o.Package}
					err = nil
				}
			} else {
				tok, err = parseLineWithRegistry(body, o.Package, false, registry)
			}
			if err != nil {
				n.attrs.TokenKind = KindUnknown
				n.text = raw
				severity := "error"
				if errors.Is(err, ErrNoTokenMatch) {
					severity = "warning"
					if o.Strict {
						severity = "error"
					}
				}
				diagnostic(n, err.Error(), severity)
				tok = &TokenUnknownControl{ControlChar: rune(body[0]), Name: cmd, RawArgs: args, RawLine: body}
			} else {
				if meta, ok := tok.(Node); ok {
					n.attrs.TokenKind = meta.TokenKind()
					n.attrs.Package = meta.PackageName()
				}
				switch {
				case cmd == "":
					n.text = raw
				case n.attrs.TokenKind == KindComment:
					n.text = ""
				default:
					lex, _, lexErr := LexArguments(args)
					if lexErr != nil && opening && candidate != nil {
						n.text = args + astLineEnding(raw)
					} else if lexErr != nil {
						n.text = raw
						diagnostic(n, lexErr.Error(), "error")
					} else {
						n.attrs.Arguments = lex
						n.text = argumentASTText(lex)
						if n.text != "" {
							n.text += astLineEnding(raw)
						}
					}
				}
			}
			if tok != nil {
				snapshot, e := json.Marshal(tok)
				if e != nil {
					diagnostic(n, "cannot snapshot typed token: "+e.Error(), "error")
				} else {
					n.tokenType = reflect.TypeOf(tok)
					n.tokenData = snapshot
				}
			}
			// Inline spans cover the full physical line body, including exact control/
			// argument spelling, with byte spans independent of canonical token strings.
			if n.attrs.TokenKind != KindComment {
				inline, _ := ParseInline(body)
				inlinePos := pos
				if count+len(inline) >= o.ASTNodes {
					diagnostic(n, "AST inline node budget exceeded; line source retained", "error")
					readErr = fmt.Errorf("AST node budget exceeded")
				} else {
					for _, in := range inline {
						c := add(n, ASTInline, inlinePos)
						v := in
						c.inline = &v
						c.text = in.Raw
						inlinePos = advanceSourcePosition(inlinePos, in.Raw)
						c.span.End = inlinePos
						if malformedASTInline(in) {
							diagnostic(c, "incomplete inline escape preserved: "+in.Raw, "error")
						}
					}
				}
			}
		}
		if closing {
			top := blocks[len(blocks)-1]
			if state := captures[top]; state != nil {
				copy := state.capture.source.clone()
				copy.Body = nil
				top.block = &copy
				if err := state.capture.finishError(); err != nil {
					var syntax *BlockSyntaxError
					position := top.span.Start
					if errors.As(err, &syntax) {
						position = syntax.Position
					}
					diagnosticAt(top, err.Error(), "error", position)
				} else if state.custom {
					token, err := completeCapturedBlock(state.handler, state.capture)
					if err != nil {
						diagnostic(top, err.Error(), "error")
					} else if err := snapshotASTBlockToken(top, token); err != nil {
						diagnostic(top, err.Error(), "error")
					}
				}
				delete(captures, top)
			}

			top.attrs.Closed = true
			blocks = blocks[:len(blocks)-1]
			parent = top.parent
		}
		pos = lineEnd
		off = end
	}
	for _, b := range blocks {
		if state := captures[b]; state != nil {
			copy := state.capture.source.clone()
			copy.Body = nil
			b.block = &copy
			if state.capture.failure != nil {
				position := b.span.Start
				var syntax *BlockSyntaxError
				if errors.As(state.capture.failure, &syntax) {
					position = syntax.Position
				}
				diagnosticAt(b, state.capture.failure.Error(), "error", position)
			}
			if state.stopped || copy.Tail.RawSource != "" || copy.Rule.UnresolvedEnd != "" {
				continue
			}
		}
		diagnostic(b, "unterminated block "+b.attrs.Name+" (expected "+b.attrs.EndMarker+")", "error")
	}
	root.span.End = pos
	// Build the preorder index iteratively; IDs, skip ranges and container spans
	// are computed in linear time, including very wide and deeply nested sources.
	stack := []*ASTNode{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		n.id = NodeID(len(a.nodes) + 1)
		a.nodes = append(a.nodes, n)
		for i := len(n.children) - 1; i >= 0; i-- {
			stack = append(stack, n.children[i])
		}
	}
	for i := len(a.nodes) - 1; i >= 0; i-- {
		n := a.nodes[i]
		n.limit = i + 1
		if len(n.children) > 0 {
			last := n.children[len(n.children)-1]
			n.limit = last.limit
			if n.kind != ASTLine {
				n.span.End = last.span.End
			}
		}
	}
	if sourceErr != nil {
		var inputErr *ASTInputError
		if errors.As(sourceErr, &inputErr) {
			diagnosticAt(root, inputErr.Err.Error(), "error", inputErr.Position)
		}
	}
	if readErr != nil {
		var documentErr *DocumentInputError
		if !errors.As(readErr, &documentErr) {
			diagnostic(root, readErr.Error(), "error")
		}
	}
	if sourceErr != nil && readErr != nil {
		return a, errors.Join(sourceErr, readErr)
	}
	if sourceErr != nil {
		return a, sourceErr
	}
	if readErr != nil {
		return a, readErr
	}
	if o.Strict && failure != nil {
		return a, failure
	}
	return a, nil
}

func astLineEnding(s string) string {
	if strings.HasSuffix(s, "\r\n") {
		return "\r\n"
	}
	if strings.HasSuffix(s, "\n") {
		return "\n"
	}
	return ""
}
func argumentASTText(args []ArgumentLexeme) string {
	v := make([]string, len(args))
	for i, a := range args {
		v[i] = a.Value
	}
	return strings.Join(v, " ")
}
func astEndMarker(cmd, args, pkg string) (string, bool) {
	switch cmd {
	case ".EX":
		if pkg != "ms" && pkg != "roff" {
			return ".EE", true
		}
	case ".nf":
		return ".fi", false
	case ".RS":
		return ".RE", false
	case ".DS":
		if pkg == "ms" {
			return ".DE", false
		}
	case ".KS", ".KF":
		if pkg == "ms" {
			return ".KE", false
		}
	case ".FS":
		if pkg == "ms" {
			return ".FE", false
		}
	}
	return "", false
}

func astClosingMarker(cmd, pkg string) bool {
	if generatedBlockClosing(cmd) {
		return true
	}
	switch cmd {
	case "..", ".RE":
		return true
	case ".EE":
		return pkg != "ms" && pkg != "roff"
	case ".DE", ".KE", ".FE":
		return pkg == "ms"
	}
	return false
}
func malformedASTInline(in InlineToken) bool {
	raw := in.Raw
	switch in.Kind {
	case InlineStringRef, InlineRegisterRef, InlineFontChange, InlineSizeChange:
		if len(raw) == 2 {
			return true
		}
		if len(raw) > 2 && raw[2] == '[' {
			return !strings.HasSuffix(raw, "]")
		}
		if len(raw) > 2 && raw[2] == '(' {
			return utf8.RuneCountInString(raw) < 5
		}
	case InlineSpecialChar:
		if strings.HasPrefix(raw, `\[`) {
			return !strings.HasSuffix(raw, "]")
		}
		if strings.HasPrefix(raw, `\(`) {
			return utf8.RuneCountInString(raw) < 4
		}
	}
	return false
}

type astBlockCapture struct {
	handler                  BlockHandler
	capture                  *blockCapture
	custom, machine, stopped bool
}

func snapshotASTBlockToken(node *ASTNode, token Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("cannot snapshot typed block token: %w", err)
	}
	node.tokenType = reflect.TypeOf(token)
	node.tokenData = data
	return nil
}
