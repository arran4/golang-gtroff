package man

import (
	groff "github.com/arran4/golang-gtroff"
	"strings"
)

func (b *builder) inline(text string, n *groff.ASTNode, initial Style) []Run {
	runs, _, _ := b.decode(text, n, initial)
	return runs
}
func (b *builder) flowInline(text string, n *groff.ASTNode) []Run {
	runs, style, previous := b.decode(text, n, b.state.Style)
	b.state.Style = style
	b.state.Previous = previous
	return runs
}
func (b *builder) decode(text string, n *groff.ASTNode, initial Style) ([]Run, Style, Style) {
	tokens, _ := groff.ParseInline(text)
	style := initial
	previous := b.state.Previous
	var runs []Run
	appendText := func(text string, unsupported bool) {
		if text != "" {
			runs = append(runs, Run{Text: text, Style: style, Link: b.state.Link, Unsupported: unsupported, Origin: origin(n)})
		}
	}
	for _, tok := range tokens {
		switch tok.Kind {
		case groff.InlineText:
			appendText(tok.Raw, false)
		case groff.InlineFontChange:
			if tok.Name == "P" {
				small := style.Small
				style, previous = previous, style
				style.Small = small
				previous.Small = small
				continue
			}
			if next, ok := fontStyle(tok.Name); ok {
				next.Small = style.Small
				previous = style
				style = next
			} else {
				b.diag(n, "unsupported font escape preserved: "+tok.Raw, "warning")
				appendText(tok.Raw, true)
			}
		case groff.InlineSpecialChar:
			if value, ok := groff.DecodeDefaultSpecialChar(tok.Name); ok {
				appendText(value, false)
			} else {
				b.diag(n, "unsupported special character preserved: "+tok.Raw, "warning")
				appendText(tok.Raw, true)
			}
		case groff.InlineEscape:
			switch tok.Name {
			case "e", "\\":
				appendText("\\", false)
			case "-":
				appendText("-", false)
			case " ":
				appendText(" ", false)
			case "~":
				appendText("\u00a0", false)
			case "t":
				appendText("\t", false)
			case "&":
			case "c":
				b.nextContinuation = true
			case "\"", "#":
				return runs, style, previous
			default:
				b.diag(n, "unsupported escape preserved: "+tok.Raw, "warning")
				appendText(tok.Raw, true)
			}
		case groff.InlineStringRef, groff.InlineRegisterRef, groff.InlineSizeChange:
			b.diag(n, "unevaluated inline syntax preserved: "+tok.Raw, "warning")
			appendText(tok.Raw, true)
		default:
			b.diag(n, "unsupported inline syntax preserved: "+tok.Raw, "warning")
			appendText(tok.Raw, true)
		}
	}
	return runs, style, previous
}
func fontStyle(value string) (Style, bool) {
	switch strings.ToUpper(value) {
	case "R", "1":
		return Style{}, true
	case "I", "2":
		return Style{Italic: true}, true
	case "B", "3":
		return Style{Bold: true}, true
	case "BI", "4":
		return Style{Bold: true, Italic: true}, true
	case "C", "CW", "CR":
		return Style{Mono: true}, true
	case "CB":
		return Style{Mono: true, Bold: true}, true
	case "CI":
		return Style{Mono: true, Italic: true}, true
	}
	return Style{}, false
}
func (b *builder) font(value string, n *groff.ASTNode) {
	if value == "P" {
		b.state.Style, b.state.Previous = b.state.Previous, b.state.Style
		return
	}
	if style, ok := fontStyle(value); ok {
		b.state.Previous = b.state.Style
		b.state.Style = style
	} else {
		b.diag(n, "unsupported font request preserved in source: "+value, "warning")
	}
}
