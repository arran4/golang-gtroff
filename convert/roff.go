package convert

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
)

// EscapeText encodes literal data, never executable roff. ProtectLine additionally
// prevents a leading dot/apostrophe from becoming a control line. Newlines are
// handled by the canonical writer, rather than accepted inside macro arguments.
func EscapeText(s string) string {
	var out strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			out.WriteString(`\e`)
		case '"':
			out.WriteString(`\(dq`)
		case '\t':
			out.WriteString(`\t`)
		case '\u00a0':
			out.WriteString(`\~`)
		case '\r', '\n', '\f':
			out.WriteByte(' ')
		case 0:
			out.WriteRune('\ufffd')
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// ProtectLine encodes a complete literal text line with a neutral prefix.
func ProtectLine(s string) string { return `\&` + EscapeText(s) }

// QuoteArgument encodes one literal argument, including its enclosing quotes.
func QuoteArgument(s string) string { return `"` + EscapeText(s) + `"` }

type roffWriter struct {
	buffer   bytes.Buffer
	o        Options
	result   Result
	err      error
	distance string
}

func (w *roffWriter) put(s string) {
	if w.err != nil {
		return
	}
	if w.buffer.Len()+len(s) > w.o.MaxOutputBytes {
		w.err = ErrLimit
		return
	}
	w.buffer.WriteString(s)
}
func (w *roffWriter) warn(s string) {
	w.result.Diagnostics = append(w.result.Diagnostics, groff.Diagnostic{Message: s, Severity: "warning", Position: groff.Position{Filename: w.o.SourceName, Offset: -1}})
}

// WriteMan emits canonical source from a validated model. All text is escaped;
// unsupported source becomes literal preformatted text, never active requests.
// Construction, strict and budget failures leave the writer untouched.
func WriteMan(out io.Writer, d *man.Document, o Options) (Result, error) {
	o = defaults(o)
	if d == nil {
		return Result{}, fmt.Errorf("nil semantic document")
	}
	if err := d.Validate(); err != nil {
		return Result{}, err
	}
	if len(d.Nodes) > o.MaxNodes {
		return Result{}, ErrLimit
	}
	type entry struct {
		id    man.ID
		depth int
	}
	stack := []entry{{1, 0}}
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if e.depth > o.MaxDepth {
			return Result{}, ErrLimit
		}
		for _, c := range d.Node(e.id).Children {
			stack = append(stack, entry{c, e.depth + 1})
		}
	}
	w := &roffWriter{o: o, result: Result{Diagnostics: append([]groff.Diagnostic(nil), d.Diagnostics...)}}
	for _, n := range d.Nodes {
		if (n.Kind == man.Paragraph || n.Kind == man.Preformatted || n.Kind == man.Unsupported || n.Kind == man.Space) && len(n.Children) > 0 {
			return w.result, fmt.Errorf("semantic leaf %d has children", n.ID)
		}
		for _, c := range n.Children {
			if (n.Kind == man.DefinitionList) != (d.Node(c).Kind == man.Item) {
				return w.result, fmt.Errorf("inconsistent definition list/item structure")
			}
		}
		if n.Kind == man.Section || n.Kind == man.Subsection {
			p := d.Node(n.Parent)
			if p == nil || !(p.Kind == man.DocumentNode || n.Kind == man.Subsection && p.Kind == man.Section) {
				w.warn("heading hierarchy cannot be represented by man; heading lifted into page flow")
			}
		}
	}
	m := d.Metadata
	if m != (man.Metadata{}) {
		w.put(".TH " + QuoteArgument(m.Title) + " " + QuoteArgument(m.Section) + " " + QuoteArgument(m.Date) + " " + QuoteArgument(m.Source) + " " + QuoteArgument(m.Manual) + "\n")
	}
	var visit func(man.ID, bool)
	visit = func(id man.ID, inItem bool) {
		if w.err != nil {
			return
		}
		n := d.Node(id)
		switch n.Kind {
		case man.DocumentNode:
		case man.Section, man.Subsection:
			cmd := ".SH"
			if n.Kind == man.Subsection {
				cmd = ".SS"
			}
			rs := n.Runs
			if len(rs) == 0 {
				rs = []man.Run{{Text: n.Heading}}
			}
			w.put(cmd + " \"" + w.inline(normalizeRuns(rs, false, true)) + "\"\n")
			inItem = false
		case man.Paragraph:
			if inItem {
				w.put("\n")
			} else {
				w.put(".PP\n")
			}
			if n.State.ParagraphDistance != w.distance {
				w.distance = n.State.ParagraphDistance
				w.checkDimension(w.distance, true)
				w.put(".PD " + QuoteArgument(w.distance) + "\n")
			}
			rs := normalizeRuns(n.Runs, false, false)
			if len(rs) == 0 {
				w.put("\\&\\c\n")
			} else {
				w.runs(rs, false)
			}
		case man.DefinitionList:
			if inItem {
				w.put(".RS \"0\"\n")
			}
			if !inItem {
				w.put(".PP\n")
			}
		case man.Item:
			w.checkDimension(n.Value, false)
			w.put(".TP " + QuoteArgument(n.Value) + "\n")
			w.put(`\&` + w.inline(normalizeRuns(n.Tag, false, false)) + "\n")
			inItem = true
		case man.Indent:
			w.checkDimension(n.Value, false)
			if dimension(n.Value, "4") == "0em" {
				for _, c := range n.Children {
					visit(c, inItem)
				}
				return
			}
			w.put(".RS " + QuoteArgument(n.Value) + "\n")
			inItem = false
		case man.Preformatted, man.Unsupported:
			if n.Kind == man.Unsupported {
				w.warn("unsupported block emitted as escaped literal preformatted text")
			}
			w.put(".nf\n")
			w.runs(normalizeRuns(n.Runs, true, false), true)
			w.put(".fi\n")
		case man.Space:
			w.checkDimension(n.Value, true)
			w.put(".sp " + QuoteArgument(n.Value) + "\n")
		}
		for _, c := range n.Children {
			visit(c, inItem)
		}
		if n.Kind == man.Indent || (n.Kind == man.DefinitionList && inItem) {
			w.put(".RE\n")
		}
	}
	visit(1, false)
	if w.err != nil {
		return w.result, w.err
	}
	return output(out, w.buffer.Bytes(), w.result, o)
}
func font(s man.Style) string {
	v := ""
	if s.Mono {
		v = "C"
	}
	if s.Bold {
		v += "B"
	}
	if s.Italic {
		v += "I"
	}
	if v == "" || v == "C" {
		v += "R"
	}
	return v
}
func (w *roffWriter) inline(rs []man.Run) string {
	var b strings.Builder
	for _, r := range rs {
		if r.Link != "" || r.Reference != "" || r.Style.Small || r.Break || r.Unsupported {
			w.warn("heading/tag annotation has no inline canonical representation; styled text retained")
		}
		b.WriteString(`\f[` + font(r.Style) + `]` + EscapeText(r.Text) + `\f[R]`)
	}
	return b.String()
}

var referencePattern = regexp.MustCompile(`^([^()\s]+)\(([^()\s]+)\)$`)

func (w *roffWriter) runs(rs []man.Run, pre bool) {
	// Unsupported markers are diagnostics, not executable roff annotations.
	// Merge their literal output just as the second canonicalization will.
	clean := make([]man.Run, len(rs))
	copy(clean, rs)
	for i := range clean {
		if clean[i].Unsupported {
			w.warn("unsupported inline annotation emitted as escaped literal text")
			clean[i].Unsupported = false
		}
	}
	rs = normalizeRuns(clean, pre, false)
	link := ""
	for _, r := range rs {
		if !utf8.ValidString(r.Text) || strings.ContainsRune(r.Text, 0) {
			w.warn("invalid text bytes replaced; original input retained separately")
		}
		target := r.Link
		if pre && (target != "" || r.Reference != "") {
			w.warn("preformatted link/reference annotation retained only as text")
			target = ""
		}
		if target != "" && !renderhtml.SafeURL(target) {
			w.warn("unsafe link omitted; label retained")
			target = ""
		}
		if target != link {
			if link != "" {
				w.put(".UE\n")
			}
			link = target
			if link != "" {
				w.put(".UR " + QuoteArgument(link) + "\n")
			}
		}
		if !pre && r.Break {
			w.put(".br\n")
			continue
		}
		if !pre && r.Reference != "" {
			m := referencePattern.FindStringSubmatch(r.Reference)
			if m != nil && r.Text == r.Reference && !r.Style.Small {
				w.put(".ft " + font(r.Style) + "\n.MR " + QuoteArgument(m[1]) + " " + QuoteArgument(m[2]) + " \"\\c\"\n.ft R\n")
				continue
			}
			w.warn("manual-reference annotation cannot be canonicalized; text retained")
		}
		parts := strings.Split(r.Text, "\n")
		for i, p := range parts {
			if i > 0 {
				if pre {
					w.put("\n")
				} else {
					w.put(".br\n")
				}
			}
			if p == "" {
				continue
			}
			encoded := `\&\f[` + font(r.Style) + `]` + EscapeText(p) + `\f[R]\c`
			if r.Style.Small {
				w.put(".SM \"" + encoded + "\"\n")
			} else {
				w.put(encoded + "\n")
			}
		}
	}
	if link != "" {
		w.put(".UE\n")
	}
}

var verticalDimensionPattern = regexp.MustCompile(`^([+]?[0-9]+(?:\.[0-9]+)?)(v|n|m|em)?$`)

func (w *roffWriter) checkDimension(value string, vertical bool) {
	if value == "" {
		return
	}
	p := dimensionPattern
	if vertical {
		p = verticalDimensionPattern
	}
	if !p.MatchString(value) {
		w.warn("unsupported spacing expression retained as a quoted argument: " + value)
	}
}
