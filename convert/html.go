package convert

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
	"golang.org/x/net/html"
)

type importer struct {
	o            Options
	d            *man.Document
	origin       man.Origin
	err          error
	profile      bool
	titleHeading bool
}

// ImportHTML parses HTML with x/net/html. Unsupported content remains visible
// literal markup, with diagnostics. Original bytes live in Input.Source; DOM
// repairs, CSS and browser layout are not represented as roff semantics.
func ImportHTML(r io.Reader, o Options) (Input, error) {
	o = defaults(o)
	source, err := read(r, o)
	if err != nil {
		return Input{}, err
	}
	in := Input{Source: source}
	b := &importer{o: o, d: &man.Document{Version: 1}, origin: man.Origin{Span: groff.Span{End: groff.Position{Offset: len(source)}}}}
	b.add(man.DocumentNode, 0)
	in.Document = b.d
	if !utf8.ValidString(source) {
		b.warn("invalid UTF-8 repaired by the HTML parser")
	}
	if err = b.scan(source); err != nil {
		return in, err
	}
	dom, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return in, err
	}
	// Bound the repaired DOM as well as the tokenizer's original nesting.
	type entry struct {
		n     *html.Node
		depth int
	}
	stack := []entry{{dom, 0}}
	count := 0
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		if count > o.MaxNodes || e.depth > o.MaxDepth {
			return in, ErrLimit
		}
		if e.n.Type == html.ElementNode && e.n.Data == "article" && attr(e.n, "data-man-version") == "1" {
			if b.profile {
				b.warn("multiple exported articles; metadata uses the first")
			} else {
				b.profile = true
				b.d.Metadata = man.Metadata{Title: attr(e.n, "data-man-title"), Section: attr(e.n, "data-man-section"), Date: attr(e.n, "data-man-date"), Source: attr(e.n, "data-man-source"), Manual: attr(e.n, "data-man-manual")}
			}
		}
		for c := e.n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, entry{c, e.depth + 1})
		}
	}
	b.blocks(1, children(dom))
	if b.err != nil {
		return in, b.err
	}
	if err := b.d.Validate(); err != nil {
		return in, err
	}
	if o.Strict && len(b.d.Diagnostics) > 0 {
		return in, ErrStrict
	}
	return in, nil
}
func (b *importer) scan(source string) error {
	z := html.NewTokenizer(strings.NewReader(source))
	var stack []string
	count := 0
	repaired := false
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() != io.EOF {
				return z.Err()
			}
			break
		}
		count++
		if count > b.o.MaxNodes {
			return ErrLimit
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := z.Token()
			if tt == html.StartTagToken && !void(t.Data) {
				stack = append(stack, t.Data)
				if len(stack) > b.o.MaxDepth {
					return ErrLimit
				}
			}
		} else if tt == html.EndTagToken {
			name := z.Token().Data
			if len(stack) > 0 && stack[len(stack)-1] == name {
				stack = stack[:len(stack)-1]
			} else {
				repaired = true
			}
		}
	}
	if len(stack) > 0 || repaired {
		b.warn("HTML has omitted or mismatched closing tags; parser repairs applied")
	}
	return nil
}
func void(s string) bool {
	return strings.Contains(" area base br col embed hr img input link meta param source track wbr ", " "+s+" ")
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key && a.Namespace == "" {
			return a.Val
		}
	}
	return ""
}
func class(n *html.Node, value string) bool {
	for _, s := range strings.Fields(attr(n, "class")) {
		if s == value {
			return true
		}
	}
	return false
}
func children(n *html.Node) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, c)
	}
	return out
}
func content(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var s strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.WriteString(content(c))
	}
	return s.String()
}
func markup(n *html.Node) string { var buf bytes.Buffer; _ = html.Render(&buf, n); return buf.String() }
func flow(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			return ' '
		}
		return r
	}, s)
}
func asciiSpace(s string) bool { return strings.Trim(s, " \t\r\n\f") == "" }
func (b *importer) warn(s string) {
	b.d.Diagnostics = append(b.d.Diagnostics, groff.Diagnostic{Severity: "warning", Message: s, Position: groff.Position{Filename: b.o.SourceName, Offset: -1}})
}
func (b *importer) add(kind man.Kind, parent man.ID) man.ID {
	if len(b.d.Nodes) >= b.o.MaxNodes {
		b.err = ErrLimit
		return 1
	}
	id := man.ID(len(b.d.Nodes) + 1)
	b.d.Nodes = append(b.d.Nodes, man.Node{ID: id, Parent: parent, Kind: kind, State: man.State{Fill: kind != man.Preformatted}, Origin: b.origin})
	if parent != 0 {
		p := b.d.Node(parent)
		p.Children = append(p.Children, id)
	}
	return id
}
func (b *importer) unsupported(parent man.ID, n *html.Node, why string) {
	b.warn(why + "; literal markup retained")
	id := b.add(man.Unsupported, parent)
	b.d.Node(id).Runs = []man.Run{{Text: markup(n), Unsupported: true, Origin: b.origin}}
}
func inlineElement(n *html.Node) bool {
	if n.Type == html.TextNode {
		return true
	}
	return n.Type == html.ElementNode && strings.Contains(" strong b em i code small span a br ", " "+n.Data+" ")
}

// blocks handles heading flow inside a container. Explicit exported section
// wrappers retain their kind even when a selected subsection uses an h2 rank.
func (b *importer) blocks(parent man.ID, nodes []*html.Node) {
	current, section := parent, parent
	var pending []*html.Node
	flush := func() {
		if len(pending) == 0 {
			return
		}
		rs := b.runs(pending, false)
		pending = nil
		if len(rs) > 0 {
			id := b.add(man.Paragraph, current)
			b.d.Node(id).Runs = rs
		}
	}
	for _, n := range nodes {
		if b.err != nil {
			return
		}
		if inlineElement(n) {
			pending = append(pending, n)
			continue
		}
		flush()
		switch n.Type {
		case html.DocumentNode:
			b.blocks(current, children(n))
		case html.DoctypeNode:
			if n.Data != "html" {
				b.unsupported(current, n, "unsupported document type")
			}
		case html.CommentNode:
			b.unsupported(current, n, "HTML comment has no man representation")
		case html.ElementNode:
			switch n.Data {
			case "html", "body", "head":
				b.attributes(n)
				b.blocks(current, children(n))
			case "meta":
				for _, a := range n.Attr {
					if a.Namespace != "" || (a.Key != "charset" && a.Key != "name" && a.Key != "content" && a.Key != "http-equiv") {
						b.warn("unsupported metadata attribute " + a.Key + " retained only in original HTML")
					}
				}
				if !(attr(n, "charset") == "utf-8" || attr(n, "name") == "viewport" || (b.profile && attr(n, "http-equiv") == "Content-Security-Policy")) {
					b.unsupported(current, n, "unsupported metadata")
				}
			case "title":
				b.attributes(n)
				for _, c := range children(n) {
					if c.Type != html.TextNode {
						b.warn("title markup is represented as text")
					}
				}
				if !b.profile && b.d.Metadata.Title == "" {
					b.d.Metadata.Title = strings.TrimSpace(content(n))
				}
			case "style":
				b.attributes(n)
				if !renderhtml.IsGeneratedStylesheet(content(n)) {
					b.unsupported(current, n, "arbitrary CSS is unsupported")
				}
			case "article":
				b.attributes(n)
				b.blocks(current, children(n))
			case "header":
				b.attributes(n)
				for _, c := range children(n) {
					if c.Type == html.ElementNode {
						b.attributes(c)
					}
				}
				if !(b.profile && class(n, "document-header") && displayHeader(n)) {
					b.unsupported(current, n, "unsupported header")
				}
			case "aside":
				b.attributes(n)
				// Notes are retained as observable diagnostics, rather than page content.
				if b.profile && class(n, "conversion-notes") && notes(n) {
					for _, c := range children(n) {
						b.checkAttributes(c)
					}
					b.warn("exported conversion notes: " + strings.Join(strings.Fields(content(n)), " "))
				} else {
					b.unsupported(current, n, "unsupported aside")
				}
			case "section":
				kind := man.Kind(attr(n, "data-kind"))
				h := firstHeading(n)
				if b.profile && h != nil && (kind == man.Section || kind == man.Subsection) {
					if strings.TrimSpace(content(h)) == "" {
						b.unsupported(current, n, "empty exported section has no man heading representation")
						continue
					}
					b.attributes(n)
					id := b.add(kind, current)
					rs := b.runs(children(h), false)
					b.d.Node(id).Runs = rs
					b.d.Node(id).Heading = runText(rs)
					var rest []*html.Node
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c != h {
							rest = append(rest, c)
						}
					}
					b.blocks(id, rest)
				} else {
					b.attributes(n)
					b.blocks(current, children(n))
				}
			case "h1", "h2", "h3":
				b.attributes(n)
				if strings.TrimSpace(content(n)) == "" {
					b.unsupported(current, n, "empty HTML heading has no man representation")
					continue
				}
				if n.Data == "h1" && !b.profile && !b.titleHeading {
					b.titleHeading = true
					b.d.Metadata.Title = runText(b.runs(children(n), false))
					if b.d.Metadata.Section == "" {
						b.d.Metadata.Section = "1"
					}
					continue
				}
				kind := man.Section
				scope := parent
				if n.Data == "h3" {
					kind = man.Subsection
					scope = section
				}
				if p := b.d.Node(scope); p.Kind != man.DocumentNode && !(kind == man.Subsection && p.Kind == man.Section) {
					b.warn("heading inside a non-section container has no equivalent man hierarchy")
				}
				id := b.add(kind, scope)
				rs := b.runs(children(n), false)
				for i := range rs {
					rs[i].Style.Bold = true
				}
				b.d.Node(id).Runs = rs
				b.d.Node(id).Heading = runText(rs)
				current = id
				if kind == man.Section {
					section = id
				}
			case "p":
				b.attributes(n)
				rs := b.runs(children(n), false)
				id := b.add(man.Paragraph, current)
				b.d.Node(id).Runs = rs
				b.d.Node(id).State.ParagraphDistance = attr(n, "data-paragraph-distance")
			case "pre":
				b.attributes(n)
				kind := man.Preformatted
				if b.profile && attr(n, "data-kind") == "unsupported" {
					kind = man.Unsupported
				}
				id := b.add(kind, current)
				b.d.Node(id).Runs = b.runs(children(n), true)
				if kind == man.Unsupported {
					for i := range b.d.Node(id).Runs {
						b.d.Node(id).Runs[i].Unsupported = true
					}
					b.warn("exported unsupported block retained")
				}
			case "dl", "ul", "ol":
				b.list(current, n)
			case "div":
				if b.profile && attr(n, "data-kind") == "space" {
					b.attributes(n)
					id := b.add(man.Space, current)
					b.d.Node(id).Value = attr(n, "data-value")
				} else if b.profile && attr(n, "data-kind") == "indent" {
					b.attributes(n)
					id := b.add(man.Indent, current)
					b.d.Node(id).Value = attr(n, "data-value")
					b.blocks(id, children(n))
				} else {
					b.attributes(n)
					if class(n, "selection-context") {
						b.warn("selected HTML has inherited visual context; import the containing section for structural indentation")
					}
					b.blocks(current, children(n))
				}
			default:
				b.unsupported(current, n, "unsupported <"+n.Data+">")
			}
		default:
			b.unsupported(current, n, "unsupported HTML node")
		}
	}
	flush()
}
func firstHeading(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && asciiSpace(c.Data) {
			continue
		}
		if c.Type == html.ElementNode && (c.Data == "h2" || c.Data == "h3") {
			return c
		}
		return nil
	}
	return nil
}
func displayHeader(n *html.Node) bool {
	seen := false
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && asciiSpace(c.Data) {
			continue
		}
		if c.Type != html.ElementNode || !(c.Data == "h1" && !seen || c.Data == "p" && seen && class(c, "manual-meta")) {
			return false
		}
		for x := c.FirstChild; x != nil; x = x.NextSibling {
			if x.Type != html.TextNode {
				return false
			}
		}
		seen = true
	}
	return seen
}
func notes(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			if !asciiSpace(c.Data) {
				return false
			}
			continue
		}
		if c.Type != html.ElementNode {
			return false
		}
		if c.Data == "h2" {
			for x := c.FirstChild; x != nil; x = x.NextSibling {
				if x.Type != html.TextNode {
					return false
				}
			}
			continue
		}
		if c.Data != "ul" {
			return false
		}
		for x := c.FirstChild; x != nil; x = x.NextSibling {
			if x.Type == html.TextNode && asciiSpace(x.Data) {
				continue
			}
			if x.Type != html.ElementNode || x.Data != "li" {
				return false
			}
			for y := x.FirstChild; y != nil; y = y.NextSibling {
				if y.Type != html.TextNode {
					return false
				}
			}
		}
	}
	return true
}
func (b *importer) attributes(n *html.Node) {
	for _, a := range n.Attr {
		if a.Namespace != "" {
			b.warn("unsupported namespaced attribute on <" + n.Data + ">")
			continue
		}
		if a.Key == "href" && n.Data == "a" || a.Key == "start" && n.Data == "ol" || a.Key == "lang" && n.Data == "html" {
			continue
		}
		if b.profile && (strings.HasPrefix(a.Key, "data-") || a.Key == "id" || a.Key == "class" || a.Key == "aria-labelledby" || a.Key == "aria-label" || a.Key == "aria-hidden" || a.Key == "rel") {
			continue
		}
		if b.profile && a.Key == "style" && (n.Data == "dd" || n.Data == "p" || n.Data == "div") {
			// Only a single numeric margin/height generated by the exporter is ignored.
			key, v, ok := strings.Cut(a.Val, ":")
			v = strings.TrimSuffix(v, "em")
			f, err := strconv.ParseFloat(v, 64)
			if ok && err == nil && f >= 0 && f <= 100 && (key == "margin-inline-start" || key == "height" || key == "margin-block") {
				continue
			}
		}
		b.warn("unsupported attribute " + a.Key + " on <" + n.Data + "> retained only in original HTML")
	}
}
func (b *importer) runs(nodes []*html.Node, pre bool) []man.Run {
	var out []man.Run
	var walk func(*html.Node, man.Style, string, string, bool)
	walk = func(n *html.Node, s man.Style, link, ref string, unsupported bool) {
		if n.Type == html.TextNode {
			text := n.Data
			if !pre {
				text = flow(text)
			}
			if text != "" {
				out = append(out, man.Run{Text: text, Style: s, Link: link, Reference: ref, Unsupported: unsupported, Origin: b.origin})
			}
			return
		}
		if n.Type != html.ElementNode {
			b.warn("unsupported inline HTML retained literally")
			out = append(out, man.Run{Text: markup(n), Style: s, Unsupported: true, Origin: b.origin})
			return
		}
		b.attributes(n)
		if b.profile && n.Data == "a" && class(n, "permalink") && content(n) == "#" && n.FirstChild == n.LastChild && n.FirstChild != nil && n.FirstChild.Type == html.TextNode && strings.HasPrefix(attr(n, "href"), "#man-node-") {
			return
		}
		switch n.Data {
		case "strong", "b":
			s.Bold = true
		case "em", "i":
			s.Italic = true
		case "code":
			s.Mono = true
		case "small":
			s.Small = true
		case "span":
			if b.profile && class(n, "man-reference") && attr(n, "data-reference") != "" {
				ref = attr(n, "data-reference")
			}
			if b.profile && class(n, "unsupported-inline") {
				unsupported = true
			}
		case "a":
			if target := attr(n, "href"); renderhtml.SafeURL(target) {
				link = target
			} else {
				b.warn("unsafe or absent HTML link omitted; label retained")
			}
		case "br":
			out = append(out, man.Run{Text: "\n", Break: true, Style: s, Origin: b.origin})
			return
		default:
			b.warn("unsupported inline <" + n.Data + "> retained literally")
			out = append(out, man.Run{Text: markup(n), Style: s, Unsupported: true, Origin: b.origin})
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, s, link, ref, unsupported)
		}
	}
	for _, n := range nodes {
		walk(n, man.Style{Mono: pre}, "", "", false)
	}
	return normalizeRuns(out, pre, false)
}
func runText(rs []man.Run) string {
	var s strings.Builder
	for _, r := range rs {
		s.WriteString(r.Text)
	}
	return s.String()
}
func (b *importer) list(parent man.ID, n *html.Node) {
	empty := true
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.TextNode || !asciiSpace(c.Data) {
			empty = false
			break
		}
	}
	if empty {
		b.unsupported(parent, n, "empty list has no man item representation")
		return
	}
	b.attributes(n)
	id := b.add(man.DefinitionList, parent)
	var item man.ID
	index := 1
	if n.Data == "ol" && attr(n, "start") != "" {
		v, e := strconv.Atoi(attr(n, "start"))
		if e != nil || v < 1 || v > 1000000 {
			b.warn("unsupported ordered-list start; numbering starts at 1")
		} else {
			index = v
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && asciiSpace(c.Data) {
			continue
		}
		if c.Type != html.ElementNode {
			b.listFallback(id, c)
			continue
		}
		switch {
		case n.Data == "dl" && c.Data == "dt":
			b.attributes(c)
			item = b.add(man.Item, id)
			b.d.Node(item).Tag = b.runs(children(c), false)
			b.d.Node(item).Value = attr(c, "data-value")
		case n.Data == "dl" && c.Data == "dd":
			b.attributes(c)
			if item == 0 {
				b.warn("definition description has no term; empty term inserted")
				item = b.add(man.Item, id)
			}
			b.blocks(item, children(c))
		case n.Data != "dl" && c.Data == "li":
			b.attributes(c)
			item = b.add(man.Item, id)
			tag := "•"
			if n.Data == "ol" {
				tag = fmt.Sprintf("%d.", index)
				index++
			}
			b.d.Node(item).Tag = []man.Run{{Text: tag, Origin: b.origin}}
			b.blocks(item, children(c))
		default:
			b.listFallback(id, c)
		}
	}
}
func (b *importer) listFallback(list man.ID, n *html.Node) {
	id := b.add(man.Item, list)
	b.unsupported(id, n, "unsupported list child")
}

func (b *importer) checkAttributes(n *html.Node) {
	if n.Type == html.ElementNode {
		b.attributes(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.checkAttributes(c)
	}
}
