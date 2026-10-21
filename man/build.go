package man

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
)

type Options struct {
	Strict                       bool
	MaxNodes, MaxDepth, MaxBytes int
}
type frame struct {
	parent, item, list, group ID
	indents                   []string
}
type builder struct {
	before                                      State
	subsection                                  ID
	preEnd                                      string
	preStyle, prePrevious                       Style
	doc                                         *Document
	parent, section, item, list, paragraph, pre ID
	frames                                      []frame
	state                                       State
	pendingStyle                                string
	expectTag                                   bool
	join                                        bool
	suppressJoin, nextContinuation              bool
	err                                         error
	recovered                                   map[int]bool
}

// Build consumes an existing AST. Raw source remains independent of semantic
// text. Strict mode returns a partial useful model plus an error on diagnostics;
// resource failures always return an error. No includes/evaluation are performed.
func Build(tree *groff.AST, o Options) (*Document, error) {
	if tree == nil {
		return nil, fmt.Errorf("nil roff AST")
	}
	if o.MaxNodes <= 0 {
		o.MaxNodes = 100000
	}
	if o.MaxNodes > 100000 {
		o.MaxNodes = 100000
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = 128
	}
	if o.MaxDepth > 128 {
		o.MaxDepth = 128
	}
	if o.MaxBytes <= 0 {
		o.MaxBytes = 8 << 20
	}
	d := &Document{Version: 1, source: tree.Source()}
	b := &builder{doc: d, state: State{Fill: true}, recovered: map[int]bool{}}
	b.before = cloneState(b.state)
	b.parent = b.add(DocumentNode, 0, tree.Root())
	lines := tree.Root().FindKind(groff.ASTLine)
	for i := 0; i < len(lines); i++ {
		n := lines[i]
		b.before = cloneState(b.state)
		if len(d.source) > o.MaxBytes || len(d.Nodes)+4 > o.MaxNodes || (len(b.frames) >= o.MaxDepth && n.Attributes().Name == ".RS") {
			b.err = ErrLimit
			b.diag(n, "semantic budget exceeded; remaining source preserved as unsupported", "error")
			b.flush()
			tail := b.add(Unsupported, b.parent, n)
			node := d.Node(tail)
			node.Origin.Span.End = tree.Root().Span().End
			node.Runs = []Run{{Text: d.source[n.Span().Start.Offset:], Unsupported: true, Origin: node.Origin, Style: b.state.Style}}
			break
		}
		cmd, args, lexErr := lineArgs(n)
		rawBody := body(n.Raw())
		if len(rawBody) >= 3 && (rawBody[0] == '.' || rawBody[0] == '\'') && rawBody[1] == '\\' && (rawBody[2] == '"' || rawBody[2] == '#') {
			continue
		}
		// Preserve complex definitions/preprocessors as one inspectable fallback.
		if cmd == ".de" || cmd == ".am" || cmd == ".ig" || cmd == ".TS" || cmd == ".EQ" || cmd == ".PS" || cmd == ".[" {
			p := n.Parent()
			if p != nil && p.Kind() == groff.ASTBlock && p.Attributes().Name == cmd {
				if b.pre != 0 {
					node := d.Node(b.pre)
					node.Runs = append(node.Runs, Run{Text: p.Raw(), Style: b.state.Style, Unsupported: true, Origin: origin(p)})
					b.touch(b.pre, p)
				} else if b.expectTag && b.item != 0 {
					b.appendContent(n, []Run{{Text: p.Raw(), Style: b.state.Style, Unsupported: true, Origin: origin(p)}}, false)
					b.touch(b.item, p)
				} else {
					b.flush()
					id := b.add(Unsupported, b.contentParent(), n)
					node := d.Node(id)
					node.Value = cmd
					node.Origin = origin(p)
					node.Runs = []Run{{Text: p.Raw(), Style: b.state.Style, Unsupported: true, Origin: origin(p)}}
				}
				b.diag(n, "unsupported semantics preserved without evaluation: "+cmd, "warning")
				for i+1 < len(lines) && lines[i+1].Span().Start.Offset < p.Span().End.Offset {
					i++
				}
				continue
			}
		}
		if cmd == ".SH" || cmd == ".SS" {
			if lexErr != nil {
				b.unsupported(n, "malformed heading")
				continue
			}
			if b.subsection != 0 {
				d.Node(b.subsection).Origin.Span.End = n.Span().Start
				b.subsection = 0
			}
			if cmd == ".SH" && b.section != 0 {
				d.Node(b.section).Origin.Span.End = n.Span().Start
			}
			headingOrigin := origin(n)
			var headingRuns []Run
			heading := joinArgs(args)
			if heading == "" && i+1 < len(lines) {
				next := lines[i+1]
				nextCmd, nextArgs, nextErr := lineArgs(next)
				if nextErr == nil && (nextCmd == "" || isStyleMacro(nextCmd)) && strings.TrimSpace(body(next.Raw())) != "" {
					if nextCmd == "" {
						heading = body(next.Raw())
					} else {
						for j, arg := range nextArgs {
							if j > 0 && (len(nextCmd) == 2 || nextCmd == ".SM" || nextCmd == ".SB") {
								headingRuns = append(headingRuns, Run{Text: " ", Origin: origin(next)})
							}
							headingRuns = append(headingRuns, b.inline(arg.Value, next, macroStyle(nextCmd, j, Style{}))...)
						}
						heading = textRuns(headingRuns)
					}
					headingOrigin.ASTIDs = append(headingOrigin.ASTIDs, next.ID())
					headingOrigin.Span.End = next.Span().End
					b.recovered[n.Span().Start.Offset] = true
					i++
				}
			}
			if heading == "" {
				b.unsupported(n, "heading has no text")
				continue
			}
			b.flush()
			b.closeContexts(n)
			parent := ID(1)
			kind := Section
			if cmd == ".SS" {
				kind = Subsection
				if b.section != 0 {
					parent = b.section
				}
			}
			distance := b.state.ParagraphDistance
			b.state = State{Fill: true, ParagraphDistance: distance}
			b.before = cloneState(b.state)
			b.pendingStyle = ""
			id := b.add(kind, parent, n)
			runs := headingRuns
			if runs == nil {
				runs = b.inline(heading, n, Style{Bold: true})
			}
			for j := range runs {
				runs[j].Style.Bold = true
			}
			node := d.Node(id)
			node.Heading = textRuns(runs)
			node.Runs = runs
			for j := range node.Runs {
				node.Runs[j].Origin = headingOrigin
			}
			node.Origin = headingOrigin
			if p := n.Parent(); p != nil && p.Span().Start.Offset == n.Span().Start.Offset && (p.Kind() == groff.ASTSection || p.Kind() == groff.ASTSubsection) {
				node.Origin.ASTIDs = append(node.Origin.ASTIDs, p.ID())
			}
			if cmd == ".SH" {
				b.section = id
			} else {
				b.subsection = id
			}
			b.parent = id
			b.item = 0
			b.list = 0
			continue
		}
		if n.Attributes().TokenKind == groff.KindComment {
			continue
		}
		if lexErr != nil || !utf8.ValidString(n.Raw()) {
			b.unsupported(n, "malformed syntax preserved")
			continue
		}
		if b.pre != 0 && cmd == "" {
			b.appendPre(n, rawBody)
			continue
		}

		if b.pre != 0 && cmd != ".fi" && cmd != ".EE" && cmd != ".br" && cmd != ".sp" && cmd != ".ft" && cmd != ".nf" && cmd != ".EX" && !isStyleMacro(cmd) {
			b.unsupportedPre(n, "unsupported preformatted control preserved: "+cmd)
			continue
		}
		if b.expectTag && b.item != 0 && cmd != "" && cmd != ".TP" && cmd != ".TQ" && cmd != ".IP" && !isStyleMacro(cmd) {
			b.unsupported(n, "unsupported tag construct preserved: "+cmd)
			continue
		}
		switch cmd {
		case "":
			if strings.TrimSpace(rawBody) == "" {
				if b.pre != 0 {
					b.appendPre(n, rawBody)
				} else {
					b.flush()
				}
				continue
			}
			var runs []Run
			if b.pendingStyle != "" {
				style := macroStyle(b.pendingStyle, 0, b.state.Style)
				b.pendingStyle = ""
				runs = b.inline(rawBody, n, style)
			} else {
				runs = b.flowInline(rawBody, n)
			}
			b.appendContent(n, runs, true)
		case ".TH":
			if len(args) < 2 {
				b.unsupported(n, "title requires name and manual section")
				continue
			}
			vals := make([]string, 5)
			for j := 0; j < len(args) && j < 5; j++ {
				vals[j] = textRuns(b.inline(args[j].Value, n, Style{}))
			}
			if d.Metadata.Title != "" {
				b.diag(n, "repeated .TH uses the last title; prior source preserved", "warning")
			}
			metadataOrigin := origin(n)
			d.MetadataOrigin = &metadataOrigin
			d.Metadata = Metadata{Title: vals[0], Section: vals[1], Date: vals[2], Source: vals[3], Manual: vals[4]}
			b.touch(1, n)
			if len(args) > 5 {
				b.diag(n, "extra title arguments retained in source", "warning")
			}
		case ".PP", ".P", ".LP":
			if b.pre != 0 {
				b.unsupportedPre(n, "paragraph control inside preformatted block preserved")
				continue
			}
			b.flush()
			b.endItem()
			b.list = 0
			b.state.Style = Style{}
			b.state.Previous = Style{}
			b.state.TagIndent = ""
			b.touch(b.parent, n)
		case ".TP", ".TQ", ".IP":
			b.flush()
			if cmd == ".TQ" && b.item != 0 {
				b.expectTag = true
				continue
			}
			b.endItem()
			b.state.Style = Style{}
			b.state.Previous = Style{}
			b.ensureList(n)
			if len(args) > 0 && cmd != ".IP" {
				b.state.TagIndent = args[0].Value
			}
			if len(args) > 1 && cmd == ".IP" {
				b.state.TagIndent = args[1].Value
			}
			b.item = b.add(Item, b.list, n)
			d.Node(b.item).Value = b.state.TagIndent
			b.expectTag = cmd != ".IP"
			if cmd == ".IP" && len(args) > 0 {
				d.Node(b.item).Tag = b.inline(args[0].Value, n, b.state.Style)
			}
		case ".RS":
			b.flush()
			p := b.contentParent()
			group := b.add(Indent, p, n)
			value := joinArgs(args)
			d.Node(group).Value = value
			b.frames = append(b.frames, frame{parent: b.parent, item: b.item, list: b.list, group: group, indents: append([]string(nil), b.state.Indents...)})
			b.parent = group
			b.item = 0
			b.list = 0
			b.state.Indents = append(append([]string(nil), b.state.Indents...), value)
		case ".RE":
			b.flush()
			if len(b.frames) == 0 {
				b.unsupported(n, "indent close without matching .RS")
				continue
			}
			f := b.frames[len(b.frames)-1]
			b.frames = b.frames[:len(b.frames)-1]
			b.touch(f.group, n)
			b.parent = f.parent
			b.item = f.item
			b.list = f.list
			b.state.Indents = f.indents
			if len(args) > 0 {
				b.diag(n, "numeric .RE levels are not evaluated; one indent closed", "warning")
			}
		case ".nf", ".EX":
			b.flush()
			if b.pre != 0 {
				b.diag(n, "nested preformatted opener retained", "warning")
				b.touch(b.pre, n)
				continue
			}
			b.pre = b.add(Preformatted, b.contentParent(), n)
			d.Node(b.pre).Value = joinArgs(args)
			b.state.Fill = false
			b.preEnd = ".fi"
			if cmd == ".EX" {
				b.preEnd = ".EE"
				b.preStyle = b.state.Style
				b.prePrevious = b.state.Previous
				b.state.Previous = b.state.Style
				b.state.Style = Style{Mono: true}
			}
		case ".fi", ".EE":
			if b.pre != 0 && cmd != b.preEnd {
				b.unsupportedPre(n, "mismatched preformatted delimiter preserved")
				continue
			}
			if b.pre == 0 {
				if cmd == ".EE" {
					b.unsupported(n, "example close without matching opener")
				}
				b.state.Fill = true
				continue
			}
			b.touch(b.pre, n)
			if b.preEnd == ".EE" {
				b.state.Style = b.preStyle
				b.state.Previous = b.prePrevious
			}
			b.pre = 0
			b.state.Fill = true
			b.flush()
		case ".br":
			b.appendContent(n, []Run{{Text: "\n", Break: true, Style: b.state.Style, Origin: origin(n)}}, false)
			b.join = false
		case ".sp":
			if b.pre != 0 {
				b.appendContent(n, []Run{{Text: "\n", Break: true, Style: b.state.Style, Origin: origin(n)}}, false)
				continue
			}
			b.flush()
			id := b.add(Space, b.contentParent(), n)
			d.Node(id).Value = joinArgs(args)
		case ".ft":
			value := "P"
			if len(args) > 0 {
				value = args[0].Value
			}
			b.font(value, n)
			b.touch(b.contentParent(), n)
		case ".PD":
			b.state.ParagraphDistance = joinArgs(args)
			b.touch(b.contentParent(), n)
		case ".UR", ".MT":
			if len(args) == 0 {
				b.unsupported(n, "link requires a target")
				continue
			}
			b.state.Link = textRuns(b.inline(args[0].Value, n, Style{}))
			if cmd == ".MT" {
				b.state.Link = "mailto:" + b.state.Link
			}
			b.touch(b.contentParent(), n)
		case ".UE", ".ME":
			if b.state.Link == "" {
				b.diag(n, "link close without matching opener", "warning")
			}
			b.state.Link = ""
			if len(args) > 0 {
				b.appendContent(n, b.inline(joinArgs(args), n, b.state.Style), false)
			}
			b.touch(b.contentParent(), n)
		case ".MR":
			if len(args) < 2 {
				b.unsupported(n, "manual reference requires name and section")
				continue
			}
			text := textRuns(b.inline(args[0].Value, n, Style{})) + "(" + textRuns(b.inline(args[1].Value, n, Style{})) + ")"
			runs := []Run{{Text: text, Reference: text, Style: b.state.Style, Origin: origin(n)}}
			if len(args) > 2 {
				runs = append(runs, b.inline(joinArgs(args[2:]), n, b.state.Style)...)
			}
			b.appendContent(n, runs, true)
		case ".SY":
			b.flush()
			b.appendContent(n, b.inline(joinArgs(args), n, Style{Bold: true}), false)
		case ".YS":
			b.flush()
		case ".OP":
			runs := []Run{{Text: "[", Style: b.state.Style, Origin: origin(n)}}
			if len(args) > 0 {
				runs = append(runs, b.inline(args[0].Value, n, Style{Bold: true})...)
			}
			if len(args) > 1 {
				runs = append(runs, Run{Text: " ", Style: b.state.Style, Origin: origin(n)})
				runs = append(runs, b.inline(args[1].Value, n, Style{Italic: true})...)
			}
			runs = append(runs, Run{Text: "]", Style: b.state.Style, Origin: origin(n)})
			if len(args) > 2 {
				b.diag(n, "extra option arguments preserved as literal text", "warning")
				runs = append(runs, b.inline(" "+joinArgs(args[2:]), n, b.state.Style)...)
			}
			b.appendContent(n, runs, true)
		default:
			if isStyleMacro(cmd) {
				if b.pendingStyle != "" {
					b.diag(n, "nested next-line style macro approximated; original source preserved", "warning")
					b.pendingStyle = ""
				}
				if len(args) == 0 {
					if cmd == ".B" || cmd == ".I" || cmd == ".SM" || cmd == ".SB" {
						b.pendingStyle = cmd
					} else {
						b.recovered[n.Span().Start.Offset] = true
					}
					continue
				}
				var runs []Run
				for j, arg := range args {
					if j > 0 && (len(cmd) == 2 || cmd == ".SM" || cmd == ".SB") {
						runs = append(runs, Run{Text: " ", Style: b.state.Style, Origin: origin(n)})
					}
					runs = append(runs, b.inline(arg.Value, n, macroStyle(cmd, j, b.state.Style))...)
				}
				b.appendContent(n, runs, true)
			} else {
				b.unsupported(n, "unsupported semantics preserved: "+cmd)
			}
		}
	}
	b.flush()
	if b.section != 0 {
		d.Node(b.section).Origin.Span.End = tree.Root().Span().End
	}
	if b.subsection != 0 {
		d.Node(b.subsection).Origin.Span.End = tree.Root().Span().End
	}
	if b.expectTag && b.item != 0 {
		b.doc.Diagnostics = append(b.doc.Diagnostics, groff.Diagnostic{Message: "tagged paragraph has no tag", Position: d.Node(b.item).Origin.Span.Start, Severity: "warning"})
	}
	if b.state.Link != "" {
		b.doc.Diagnostics = append(b.doc.Diagnostics, groff.Diagnostic{Message: "unterminated link", Position: tree.Root().Span().End, Severity: "warning"})
	}
	for _, diag := range tree.Diagnostics() {
		if b.recovered[diag.Position.Offset] && strings.Contains(diag.Message, "missing required argument") {
			continue
		}
		d.Diagnostics = append(d.Diagnostics, diag)
	}
	sort.SliceStable(d.Diagnostics, func(i, j int) bool { return d.Diagnostics[i].Position.Offset < d.Diagnostics[j].Position.Offset })
	// Extend group origins in reverse creation order, which is topological.
	for i := len(d.Nodes) - 1; i >= 0; i-- {
		n := &d.Nodes[i]
		if n.Parent != 0 {
			p := d.Node(n.Parent)
			if n.Origin.Span.End.Offset > p.Origin.Span.End.Offset {
				p.Origin.Span.End = n.Origin.Span.End
			}
		}
	}
	d.Nodes[0].Origin = origin(tree.Root())
	if b.err != nil {
		return d, b.err
	}
	if o.Strict && len(d.Diagnostics) > 0 {
		return d, fmt.Errorf("man semantics: %s", d.Diagnostics[0].Message)
	}
	return d, nil
}

func (b *builder) add(kind Kind, parent ID, n *groff.ASTNode) ID {
	id := ID(len(b.doc.Nodes) + 1)
	b.doc.Nodes = append(b.doc.Nodes, Node{ID: id, Parent: parent, Kind: kind, State: cloneState(b.before), Origin: origin(n)})
	if parent != 0 {
		p := b.doc.Node(parent)
		p.Children = append(p.Children, id)
	}
	return id
}
func (b *builder) contentParent() ID {
	if b.item != 0 {
		return b.item
	}
	return b.parent
}
func (b *builder) flush() {
	b.paragraph = 0
	b.join = false
	b.suppressJoin = false
	b.nextContinuation = false
}
func (b *builder) endItem() {
	if b.expectTag && b.item != 0 {
		b.doc.Diagnostics = append(b.doc.Diagnostics, groff.Diagnostic{Message: "tagged paragraph has no tag", Position: b.doc.Node(b.item).Origin.Span.Start, Severity: "warning"})
	}
	b.item = 0
	b.expectTag = false
}
func (b *builder) ensureList(n *groff.ASTNode) {
	if b.list != 0 && b.doc.Node(b.list).Parent == b.parent {
		return
	}
	b.list = b.add(DefinitionList, b.parent, n)
}
func (b *builder) touch(id ID, n *groff.ASTNode) {
	node := b.doc.Node(id)
	if node == nil {
		return
	}
	ids := node.Origin.ASTIDs
	if len(ids) == 0 || ids[len(ids)-1] != n.ID() {
		node.Origin.ASTIDs = append(ids, n.ID())
	}
	if n.Span().End.Offset > node.Origin.Span.End.Offset {
		node.Origin.Span.End = n.Span().End
	}
}
func (b *builder) diag(n *groff.ASTNode, message, severity string) {
	b.doc.Diagnostics = append(b.doc.Diagnostics, groff.Diagnostic{Message: message, Position: n.Span().Start, Severity: severity})
}
func (b *builder) unsupported(n *groff.ASTNode, message string) {
	if b.expectTag && b.item != 0 {
		b.appendContent(n, []Run{{Text: body(n.Raw()), Style: b.state.Style, Unsupported: true, Origin: origin(n)}}, false)
		b.diag(n, message, "warning")
		return
	}
	if b.pre != 0 {
		b.unsupportedPre(n, message)
		return
	}
	b.flush()
	id := b.add(Unsupported, b.contentParent(), n)
	node := b.doc.Node(id)
	node.Value = n.Attributes().Name
	node.Runs = []Run{{Text: n.Raw(), Style: b.state.Style, Unsupported: true, Origin: origin(n)}}
	b.diag(n, message, "warning")
}
func (b *builder) closeContexts(n *groff.ASTNode) {
	if len(b.frames) > 0 {
		b.diag(n, "heading closed unbalanced indentation", "warning")
	}
	b.frames = nil
	if b.pre != 0 {
		b.diag(n, "heading closed unbalanced preformatted block", "warning")
	}
	b.pre = 0
	if b.expectTag {
		b.diag(n, "heading closed tagged paragraph without tag", "warning")
	}
	b.expectTag = false
}
func (b *builder) appendContent(n *groff.ASTNode, runs []Run, separate bool) {
	if b.expectTag && b.item != 0 {
		node := b.doc.Node(b.item)
		if len(node.Tag) > 0 {
			node.Tag = append(node.Tag, Run{Text: " ", Style: b.state.Style, Origin: origin(n)})
		}
		node.Tag = append(node.Tag, runs...)
		b.touch(b.item, n)
		b.expectTag = false
		return
	}
	if b.pre != 0 {
		node := b.doc.Node(b.pre)
		node.Runs = append(node.Runs, runs...)
		if strings.HasSuffix(n.Raw(), "\n") && !b.nextContinuation {
			node.Runs = append(node.Runs, Run{Text: "\n", Break: true, Style: b.state.Style, Origin: origin(n)})
		}
		b.touch(b.pre, n)
		b.nextContinuation = false
		return
	}
	if b.paragraph == 0 {
		b.paragraph = b.add(Paragraph, b.contentParent(), n)
	}
	p := b.doc.Node(b.paragraph)
	if separate && b.join && !b.suppressJoin && len(runs) > 0 {
		p.Runs = append(p.Runs, Run{Text: " ", Style: b.state.Style, Origin: origin(n)})
	}
	p.Runs = append(p.Runs, runs...)
	b.touch(b.paragraph, n)
	b.join = len(runs) > 0 && !runs[len(runs)-1].Break
	b.suppressJoin = b.nextContinuation
	b.nextContinuation = false
}
func (b *builder) appendPre(n *groff.ASTNode, text string) {
	b.appendContent(n, b.flowInline(text, n), false)
}
func lineArgs(n *groff.ASTNode) (string, []groff.ArgumentLexeme, error) {
	text := body(n.Raw())
	if text == "" || (text[0] != '.' && text[0] != '\'') {
		return "", nil, nil
	}
	stop := strings.IndexAny(text, " \t")
	if stop < 0 {
		return "." + text[1:], nil, nil
	}
	args, _, err := groff.LexArguments(strings.TrimLeft(text[stop:], " \t"))
	return "." + text[1:stop], args, err
}
func body(s string) string { return strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r") }
func joinArgs(args []groff.ArgumentLexeme) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.Value
	}
	return strings.Join(parts, " ")
}
func isStyleMacro(cmd string) bool {
	switch cmd {
	case ".B", ".I", ".R", ".BR", ".BI", ".IB", ".IR", ".RB", ".RI", ".SM", ".SB":
		return true
	}
	return false
}
func macroStyle(cmd string, index int, base Style) Style {
	if cmd == ".SM" {
		return Style{Small: true}
	}
	if cmd == ".SB" {
		return Style{Bold: true, Small: true}
	}
	letters := cmd[1:]
	ch := letters[index%len(letters)]
	switch ch {
	case 'B':
		return Style{Bold: true}
	case 'I':
		return Style{Italic: true}
	default:
		return Style{}
	}
}

func (b *builder) unsupportedPre(n *groff.ASTNode, message string) {
	node := b.doc.Node(b.pre)
	node.Runs = append(node.Runs, Run{Text: n.Raw(), Style: b.state.Style, Unsupported: true, Origin: origin(n)})
	b.touch(b.pre, n)
	b.diag(n, message, "warning")
}
