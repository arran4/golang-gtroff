package renderhtml

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

type Result struct {
	Diagnostics []groff.Diagnostic
	Bytes       int
}
type renderer struct {
	doc     *man.Document
	o       Options
	result  Result
	buffer  bytes.Buffer
	err     error
	notes   map[string]bool
	origins map[int]bool
}

// Render writes a complete page or an options-selected subtree. Construction and
// strict failures leave w untouched. Writer failures can leave a partial stream.
func Render(w io.Writer, d *man.Document, o Options) (Result, error) {
	if d == nil {
		return Result{}, fmt.Errorf("nil semantic document")
	}
	if err := d.Validate(); err != nil {
		return Result{}, err
	}
	var selection *man.Selection
	var err error
	switch {
	case o.Section != "":
		selection, err = d.SelectSection(o.Section, false)
	case o.Node != 0:
		selection, err = d.SelectNode(o.Node)
	default:
		selection, err = d.SelectNode(1)
	}
	if err != nil {
		return Result{}, err
	}
	return RenderSelection(w, d, selection, o)
}

// RenderSelection uses roots from a semantic selection and source-owned node
// data from d. Selection metadata/copies never replace the validated document.
func RenderSelection(w io.Writer, d *man.Document, s *man.Selection, o Options) (Result, error) {
	if w == nil {
		return Result{}, fmt.Errorf("nil HTML writer")
	}
	if d == nil || s == nil || len(s.Roots) == 0 {
		return Result{}, fmt.Errorf("empty semantic selection")
	}
	if err := d.Validate(); err != nil {
		return Result{}, err
	}
	o, err := defaults(o)
	if err != nil {
		return Result{}, err
	}
	r := &renderer{doc: d, o: o, notes: map[string]bool{}, origins: map[int]bool{}, result: Result{Diagnostics: append([]groff.Diagnostic(nil), d.Diagnostics...)}}
	for _, diag := range r.result.Diagnostics {
		r.origins[diag.Position.Offset] = true
	}
	for _, n := range d.Nodes {
		if (n.Kind == man.Paragraph || n.Kind == man.Preformatted || n.Kind == man.Unsupported || n.Kind == man.Space) && len(n.Children) > 0 {
			return r.result, fmt.Errorf("semantic leaf %d has children", n.ID)
		}
		for _, id := range n.Children {
			if (n.Kind == man.DefinitionList) != (d.Node(id).Kind == man.Item) {
				return r.result, fmt.Errorf("definition list/item structure is inconsistent")
			}
		}
	}
	for _, id := range s.Roots {
		if d.Node(id) == nil {
			return r.result, fmt.Errorf("selected node %d is absent", id)
		}
	}
	var roots []man.ID
	rootSet := map[man.ID]bool{}
	for _, id := range s.Roots {
		rootSet[id] = true
	}
	for id := range rootSet {
		nested := false
		for p := d.Node(id).Parent; p != 0; p = d.Node(p).Parent {
			if rootSet[p] {
				nested = true
				break
			}
		}
		if !nested {
			roots = append(roots, id)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		a, b := d.Node(roots[i]), d.Node(roots[j])
		if a.Origin.Span.Start.Offset != b.Origin.Span.Start.Offset {
			return a.Origin.Span.Start.Offset < b.Origin.Span.Start.Offset
		}
		return a.ID < b.ID
	})
	title := o.Title
	if title == "" {
		title = d.Metadata.Title
		if d.Metadata.Section != "" {
			title += "(" + d.Metadata.Section + ")"
		}
		if title == "" {
			title = "Manual page"
		}
	}
	if !o.Fragment {
		r.write("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; font-src data:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'\">\n<title>")
		r.escape(title)
		r.write("</title>\n")
	}
	r.write("<style>\n")
	r.write(styleSheet(o))
	r.write("</style>\n")
	if !o.Fragment {
		r.write("</head>\n<body class=\"gtroff-document\">\n")
	}
	r.write("<article class=\"man-page theme-" + o.Theme + "\" data-man-version=\"1\"")
	for _, field := range []struct{ name, value string }{{"title", d.Metadata.Title}, {"section", d.Metadata.Section}, {"date", d.Metadata.Date}, {"source", d.Metadata.Source}, {"manual", d.Metadata.Manual}} {
		r.write(" data-man-" + field.name + "=\"")
		r.escape(field.value)
		r.write("\"")
	}
	r.write(">\n<header class=\"document-header\"><h1>")
	r.escape(title)
	r.write("</h1>")
	var fields []string
	for _, field := range []string{d.Metadata.Manual, d.Metadata.Source, d.Metadata.Date} {
		if field != "" {
			fields = append(fields, field)
		}
	}
	if len(fields) > 0 {
		r.write("<p class=\"manual-meta\">")
		r.escape(strings.Join(fields, " · "))
		r.write("</p>")
	}
	r.write("</header>\n")
	for _, id := range roots {
		n := d.Node(id)
		context := r.contextMargin(n)
		if context != 0 {
			r.write("<div class=\"selection-context\" style=\"margin-inline-start:" + number(context) + "em\">\n")
		}
		wrapItem := n.Kind == man.Item
		if wrapItem {
			r.write("<dl class=\"definitions\">\n")
		}
		r.walk(id)
		if wrapItem {
			r.write("</dl>\n")
		}
		if context != 0 {
			r.write("</div>\n")
		}
	}
	sort.SliceStable(r.result.Diagnostics, func(i, j int) bool {
		return r.result.Diagnostics[i].Position.Offset < r.result.Diagnostics[j].Position.Offset
	})
	if len(r.result.Diagnostics) > 0 {
		r.write("<aside class=\"conversion-notes\" aria-label=\"Conversion notes\"><h2>Conversion notes</h2><ul>\n")
		for _, diag := range r.result.Diagnostics {
			r.write("<li>")
			r.escape(fmt.Sprintf("%s:%d:%d: %s: %s", diag.Position.Filename, diag.Position.Line, diag.Position.Column, diag.Severity, diag.Message))
			r.write("</li>\n")
		}
		r.write("</ul></aside>\n")
	}
	r.write("</article>\n")
	if !o.Fragment {
		r.write("</body>\n</html>\n")
	}
	if r.err != nil {
		return r.result, r.err
	}
	if o.Strict && len(r.result.Diagnostics) > 0 {
		return r.result, ErrStrict
	}
	written, err := w.Write(r.buffer.Bytes())
	r.result.Bytes = written
	if err == nil && written != r.buffer.Len() {
		err = io.ErrShortWrite
	}
	return r.result, err
}
func (r *renderer) write(s string) {
	if r.err != nil {
		return
	}
	if len(s) > r.o.MaxOutputBytes-r.buffer.Len() {
		r.err = ErrLimit
		return
	}
	r.buffer.WriteString(s)
}
func (r *renderer) escape(s string) {
	if r.err != nil {
		return
	}
	if len(s) > r.o.MaxOutputBytes-r.buffer.Len() {
		r.err = ErrLimit
		return
	}
	r.write(html.EscapeString(strings.ReplaceAll(strings.ToValidUTF8(s, "�"), "\x00", "�")))
}
func (r *renderer) diag(n *man.Node, message string) {
	key := strconv.Itoa(n.Origin.Span.Start.Offset) + message
	if r.notes[key] {
		return
	}
	r.notes[key] = true
	r.origins[n.Origin.Span.Start.Offset] = true
	r.result.Diagnostics = append(r.result.Diagnostics, groff.Diagnostic{Message: message, Position: n.Origin.Span.Start, Severity: "warning"})
}
func (r *renderer) attr(n *man.Node) {
	r.write(" data-kind=\"" + string(n.Kind) + "\" data-node-id=\"" + strconv.FormatUint(uint64(n.ID), 10) + "\" data-source-start=\"" + strconv.Itoa(n.Origin.Span.Start.Offset) + "\"")
	if n.Value != "" {
		r.write(" data-value=\"")
		r.escape(n.Value)
		r.write("\"")
	}
	if n.State.ParagraphDistance != "" {
		r.write(" data-paragraph-distance=\"")
		r.escape(n.State.ParagraphDistance)
		r.write("\"")
	}

}
func (r *renderer) id(n *man.Node) string { return "man-node-" + strconv.FormatUint(uint64(n.ID), 10) }
func (r *renderer) contextMargin(n *man.Node) float64 {
	// Parent structure retains the actual item widths even when later TP calls
	// changed the global tag-width state. No source is re-evaluated.
	margin := 0.0
	for id := n.Parent; id != 0; id = r.doc.Node(id).Parent {
		parent := r.doc.Node(id)
		if parent.Kind == man.Indent || parent.Kind == man.Item {
			value, ok := length(parent.Value, 2)
			if !ok {
				r.diag(n, "unsupported inherited indentation; using 2em")
			}
			margin += value
		}
	}
	return margin
}

func (r *renderer) walk(root man.ID) {
	type frame struct {
		id         man.ID
		exit       bool
		hasSection bool
	}
	stack := []frame{{id: root}}
	for len(stack) > 0 && r.err == nil {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		n := r.doc.Node(f.id)
		if f.exit {
			switch n.Kind {
			case man.Section, man.Subsection:
				r.write("</section>\n")
			case man.DefinitionList:
				r.write("</dl>\n")
			case man.Item:
				r.write("</dd>\n")
			case man.Indent:
				r.write("</div>\n")
			}
			continue
		}
		switch n.Kind {
		case man.DocumentNode:
		case man.Section, man.Subsection:
			level := "h2"
			if n.Kind == man.Subsection && f.hasSection {
				level = "h3"
			}
			r.write("<section")
			r.attr(n)
			if r.o.NoAnchors {
				r.write(" aria-label=\"")
				r.escape(n.Heading)
			} else {
				r.write(" aria-labelledby=\"" + r.id(n))
			}
			r.write("\">\n<" + level)
			if !r.o.NoAnchors {
				r.write(" id=\"" + r.id(n) + "\"")
			}
			r.write(">")
			if len(n.Runs) > 0 {
				r.runs(n, n.Runs, false)
			} else {
				r.escape(n.Heading)
			}
			if !r.o.NoAnchors {
				r.write("<a class=\"permalink\" href=\"#" + r.id(n) + "\" aria-label=\"Link to ")
				r.escape(n.Heading)
				r.write("\">#</a>")
			}
			r.write("</" + level + ">\n")
		case man.Paragraph:
			r.write("<p")
			r.attr(n)
			if n.State.ParagraphDistance != "" {
				v, ok := vertical(n.State.ParagraphDistance, r.o.LineHeight)
				if !ok {
					r.diag(n, "unsupported paragraph spacing; using stylesheet defaults")
				} else {
					r.write(" style=\"margin-block:" + number(v) + "em\"")
				}
			}
			r.write(">")
			r.runs(n, n.Runs, false)
			r.write("</p>\n")
		case man.Preformatted, man.Unsupported:
			class := "preformatted"
			if n.Kind == man.Unsupported {
				class = "unsupported"
				r.ensureUnsupported(n)
			}
			r.write("<pre class=\"" + class + "\"")
			r.attr(n)
			r.write("><code>")
			r.runs(n, n.Runs, true)
			r.write("</code></pre>\n")
		case man.DefinitionList:
			r.write("<dl class=\"definitions\"")
			r.attr(n)
			r.write(">\n")
		case man.Item:
			r.write("<dt")
			r.attr(n)
			if !r.o.NoAnchors {
				r.write(" id=\"" + r.id(n) + "\"")
			}
			if len(n.Tag) == 0 {
				r.write(" aria-label=\"Indented paragraph\"")
			}
			r.write(">")
			r.runs(n, n.Tag, false)
			r.write("</dt>\n<dd")
			value, ok := length(n.Value, 2)
			if !ok {
				r.diag(n, "unsupported tag width; using 2em")
			}
			r.write(" style=\"margin-inline-start:" + number(value) + "em\">")
		case man.Indent:
			value, ok := length(n.Value, 2)
			if !ok {
				r.diag(n, "unsupported indent; using 2em")
			}
			r.write("<div class=\"indent\"")
			r.attr(n)
			r.write(" style=\"margin-inline-start:" + number(value) + "em\">\n")
		case man.Space:
			value, ok := vertical(n.Value, r.o.LineHeight)
			if !ok {
				r.diag(n, "unsupported vertical spacing; using one line")
			}
			r.write("<div class=\"vertical-space\" aria-hidden=\"true\"")
			r.attr(n)
			r.write(" style=\"height:" + number(value) + "em\"></div>\n")
		}
		stack = append(stack, frame{id: n.ID, exit: true, hasSection: f.hasSection})
		for i := len(n.Children) - 1; i >= 0; i-- {
			stack = append(stack, frame{id: n.Children[i], hasSection: f.hasSection || n.Kind == man.Section})
		}
	}
}
func (r *renderer) runs(n *man.Node, runs []man.Run, pre bool) {
	currentLink := ""
	for _, run := range runs {
		where := *n
		where.Origin = run.Origin
		link := run.Link
		if link != "" && !SafeURL(link) {
			r.diag(&where, "unsafe or unsupported link target omitted")
			link = ""
		}
		if link != currentLink {
			if currentLink != "" {
				r.write("</a>")
			}
			currentLink = link
			if link != "" {
				r.write("<a href=\"")
				r.escape(link)
				r.write("\" rel=\"noreferrer noopener\">")
			}
		}
		if !utf8.ValidString(run.Text) || strings.ContainsRune(run.Text, 0) {
			r.diag(&where, "invalid HTML text bytes replaced; original source retained")
		}
		if run.Reference != "" {
			r.write("<span class=\"man-reference\" data-reference=\"")
			r.escape(run.Reference)
			r.write("\">")
		}
		if run.Style.Bold {
			r.write("<strong>")
		}
		if run.Style.Italic {
			r.write("<em>")
		}
		if run.Style.Mono && !pre {
			r.write("<code>")
		}
		if run.Style.Small {
			r.write("<small>")
		}
		if run.Unsupported {
			r.ensureUnsupported(&where)
			r.write("<span class=\"unsupported-inline\">")
		}
		if run.Break && !pre {
			parts := strings.Split(run.Text, "\n")
			for i, p := range parts {
				if i > 0 {
					r.write("<br>")
				}
				r.escape(p)
			}
		} else {
			r.escape(run.Text)
		}
		if run.Unsupported {
			r.write("</span>")
		}
		if run.Style.Small {
			r.write("</small>")
		}
		if run.Style.Mono && !pre {
			r.write("</code>")
		}
		if run.Style.Italic {
			r.write("</em>")
		}
		if run.Style.Bold {
			r.write("</strong>")
		}
		if run.Reference != "" {
			r.write("</span>")
		}
	}
	if currentLink != "" {
		r.write("</a>")
	}
}

func (r *renderer) ensureUnsupported(n *man.Node) {
	if !r.origins[n.Origin.Span.Start.Offset] {
		r.diag(n, "unsupported semantic content rendered as literal text")
	}
}
