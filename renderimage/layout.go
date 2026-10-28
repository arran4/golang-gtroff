package renderimage

import (
	"fmt"
	"image"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
	"golang.org/x/image/math/fixed"
)

type layouter struct {
	plan                  *Plan
	doc                   *man.Document
	book                  *fontBook
	y, pendingGap, glyphs int
	err                   error
	notes                 map[string]bool
	customMono            bool
	lastKind              man.Kind
}

// Layout validates and measures a whole document or an options-selected subtree.
// No RGBA image is allocated until the complete plan passes all budgets/strictness.
func Layout(d *man.Document, o Options) (*Plan, error) {
	if d == nil {
		return nil, fmt.Errorf("nil semantic document")
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	var s *man.Selection
	var err error
	switch {
	case o.Section != "":
		s, err = d.SelectSection(o.Section, false)
	case o.Node != 0:
		s, err = d.SelectNode(o.Node)
	default:
		s, err = d.SelectNode(1)
	}
	if err != nil {
		return nil, err
	}
	return LayoutSelection(d, s, o)
}
func LayoutSelection(d *man.Document, s *man.Selection, o Options) (*Plan, error) {
	if d == nil || s == nil || len(s.Roots) == 0 {
		return nil, fmt.Errorf("empty semantic selection")
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	customMono := len(o.Fonts.Mono) > 0
	o, err := defaults(o)
	if err != nil {
		return nil, err
	}
	data, err := fontData(o)
	if err != nil {
		return nil, err
	}
	p := &Plan{o: o, data: data, result: Result{Diagnostics: append([]groff.Diagnostic(nil), d.Diagnostics...)}}
	b := &layouter{plan: p, doc: d, book: newBook(data, o.DPI), notes: map[string]bool{}, customMono: customMono}
	defer b.book.close()
	for _, n := range d.Nodes {
		if (n.Kind == man.Paragraph || n.Kind == man.Preformatted || n.Kind == man.Unsupported || n.Kind == man.Space) && len(n.Children) > 0 {
			return nil, fmt.Errorf("semantic leaf has children")
		}
		for _, c := range n.Children {
			if (n.Kind == man.DefinitionList) != (d.Node(c).Kind == man.Item) {
				return nil, fmt.Errorf("inconsistent definition list/item structure")
			}
		}
	}
	roots := map[man.ID]bool{}
	for _, id := range s.Roots {
		if d.Node(id) == nil {
			return nil, fmt.Errorf("selected node absent")
		}
		roots[id] = true
	}
	var ordered []man.ID
	for id := range roots {
		nested := false
		for parent := d.Node(id).Parent; parent != 0; parent = d.Node(parent).Parent {
			if roots[parent] {
				nested = true
				break
			}
		}
		if !nested {
			ordered = append(ordered, id)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	b.newPage()
	title := o.Title
	if title == "" {
		title = d.Metadata.Title
		if d.Metadata.Section != "" {
			title += "(" + d.Metadata.Section + ")"
		}
	}
	if title != "" {
		b.text(1, man.DocumentNode, []man.Run{{Text: title, Style: man.Style{Bold: true}}}, 0, 1.6, false)
		b.gap(.25)
	}
	var meta []string
	for _, s := range []string{d.Metadata.Manual, d.Metadata.Source, d.Metadata.Date} {
		if s != "" {
			meta = append(meta, s)
		}
	}
	if len(meta) > 0 {
		b.text(1, man.DocumentNode, []man.Run{{Text: strings.Join(meta, " · "), Style: man.Style{Small: true}}}, 0, 1, false)
		b.gap(.75)
	}
	for _, id := range ordered {
		indent := 0.0
		for parent := d.Node(id).Parent; parent != 0; parent = d.Node(parent).Parent {
			n := d.Node(parent)
			if n.Kind == man.Indent || n.Kind == man.Item {
				indent += b.length(n, n.Value, 2, false)
			}
		}
		b.node(id, indent)
	}
	if b.err != nil {
		return nil, b.err
	}
	if o.Single {
		height := b.y + o.Margin
		if height < 2*o.Margin+1 {
			height = 2*o.Margin + 1
		}
		if height > o.MaxHeight {
			return nil, ErrLimit
		}
		p.pages[0].info.Height = height
	}
	var pixels int64
	for i := range p.pages {
		pg := &p.pages[i]
		pg.info.Lines = make([]Line, len(pg.lines))
		for j, l := range pg.lines {
			pg.info.Lines[j] = l.info
		}
		pixels += int64(pg.info.Width) * int64(pg.info.Height)
		if pixels > o.MaxPixels {
			return nil, ErrLimit
		}
	}
	p.result.Pages = len(p.pages)
	p.result.Pixels = pixels
	if o.Strict && len(p.result.Diagnostics) > 0 {
		return p, ErrStrict
	}
	return p, nil
}
func (b *layouter) warn(id man.ID, message string) {
	key := fmt.Sprintf("%d:%s", id, message)
	if b.notes[key] {
		return
	}
	b.notes[key] = true
	if len(b.notes) > 256 {
		if len(b.notes) == 257 {
			b.plan.result.Diagnostics = append(b.plan.result.Diagnostics, groff.Diagnostic{Message: "additional image diagnostics suppressed", Severity: "warning"})
		}
		return
	}
	pos := groff.Position{}
	if n := b.doc.Node(id); n != nil {
		pos = n.Origin.Span.Start
	}
	b.plan.result.Diagnostics = append(b.plan.result.Diagnostics, groff.Diagnostic{Message: message, Severity: "warning", Position: pos})
}
func (b *layouter) newPage() {
	if b.err != nil {
		return
	}
	o := b.plan.o
	if len(b.plan.pages) >= o.MaxPages || int64(len(b.plan.pages)+1)*int64(o.Width)*int64(o.PageHeight) > o.MaxPixels && !o.Single {
		b.err = ErrLimit
		return
	}
	b.plan.pages = append(b.plan.pages, page{info: Page{Number: len(b.plan.pages) + 1, Width: o.Width, Height: o.PageHeight}})
	b.y = o.Margin
	b.pendingGap = 0
}
func (b *layouter) gap(em float64) {
	n := int(math.Ceil(em * b.plan.o.FontSize * b.plan.o.DPI / 72))
	if n > b.pendingGap {
		b.pendingGap = n
	}
}
func (b *layouter) node(id man.ID, indent float64) {
	if b.err != nil {
		return
	}
	n := b.doc.Node(id)
	switch n.Kind {
	case man.DocumentNode:
	case man.Section, man.Subsection:
		b.gap(1)
		scale := 1.25
		if n.Kind == man.Subsection {
			scale = 1.1
		}

		rs := n.Runs
		if len(rs) == 0 {
			rs = []man.Run{{Text: n.Heading}}
		}
		copyRuns := append([]man.Run(nil), rs...)
		for i := range copyRuns {
			copyRuns[i].Style.Bold = true
		}
		b.text(id, n.Kind, copyRuns, indent, scale, false)
		b.gap(.35)
	case man.Paragraph:
		gap := .45
		if n.State.ParagraphDistance != "" {
			gap = b.length(n, n.State.ParagraphDistance, .45, true)
			if b.lastKind == man.Paragraph {
				b.pendingGap = 0
			}
		}
		b.gap(gap)
		b.text(id, n.Kind, n.Runs, indent, 1, false)
		b.gap(gap)
	case man.DefinitionList:
		b.gap(.25)
	case man.Item:
		b.gap(.4)
		if len(n.Tag) > 0 {
			b.text(id, n.Kind, n.Tag, indent, 1, false)
		}
		indent += b.length(n, n.Value, 2, false)
		b.pendingGap = 0
	case man.Indent:
		indent += b.length(n, n.Value, 2, false)
	case man.Preformatted, man.Unsupported:
		b.gap(.5)
		if n.Kind == man.Unsupported {
			b.warn(id, "unsupported source rendered as literal mono text")
		}
		b.text(id, n.Kind, n.Runs, indent, 1, true)
		b.gap(.5)
	case man.Space:
		em := b.length(n, n.Value, 1.35, true)
		b.pendingGap += int(math.Ceil(em * b.plan.o.FontSize * b.plan.o.DPI / 72))
	}
	for _, c := range n.Children {
		b.node(c, indent)
	}
}

var lengthPattern = regexp.MustCompile(`^([+]?[0-9]+(?:\.[0-9]+)?)(v|n|m|em)?$`)

func (b *layouter) length(n *man.Node, value string, fallback float64, vertical bool) float64 {
	if value == "" {
		return fallback
	}
	m := lengthPattern.FindStringSubmatch(value)
	if m == nil {
		b.warn(n.ID, "unsupported spacing expression; using documented default")
		return fallback
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return fallback
	}
	switch m[2] {
	case "n":
		v /= 2
	case "v":
		if !vertical {
			b.warn(n.ID, "vertical unit used for indentation; using default")
			return fallback
		}
		v *= b.plan.o.LineHeight
	case "":
		if vertical {
			v *= b.plan.o.LineHeight
		} else {
			v /= 2
		}
	}
	if v < 0 || v > 20 {
		b.warn(n.ID, "spacing exceeds 20em; using documented default")
		return fallback
	}
	return v
}
func (b *layouter) text(id man.ID, kind man.Kind, rs []man.Run, indent, scale float64, pre bool) {
	if b.err != nil {
		return
	}
	o := b.plan.o
	em := o.FontSize * o.DPI / 72
	left := o.Margin + int(math.Round(indent*em))
	right := o.Width - o.Margin
	if left > right-32 {
		b.warn(id, "indentation leaves insufficient width; clamped to retain visible content")
		left = right - 32
	}
	var line, word []glyph
	var pending *glyph
	flush := func(hard bool) { b.line(id, kind, line, left, right, scale, hard); line = nil; pending = nil }
	width := func(gs []glyph) fixed.Int26_6 {
		var n fixed.Int26_6
		for _, g := range gs {
			n += g.advance
		}
		return n
	}
	limit := fixed.I(right - left)
	var lineWidth fixed.Int26_6
	add := func(g glyph) { line = append(line, g); lineWidth += g.advance }
	flushWord := func() {
		if len(word) == 0 {
			return
		}
		w := width(word)
		space := fixed.Int26_6(0)
		if pending != nil && len(line) > 0 {
			space = pending.advance
		}
		if lineWidth+space+w > limit && len(line) > 0 {
			flush(false)
			lineWidth = 0
			space = 0
		}
		if w > limit {
			b.warn(id, "unbreakable word split to fit available width")
		}
		if space > 0 {
			add(*pending)
		}
		pending = nil
		for _, g := range word {
			if g.advance > limit {
				b.err = fmt.Errorf("glyph cannot fit drawable width; increase width or reduce font size")
				return
			}
			if lineWidth+g.advance > limit && len(line) > 0 {
				flush(false)
				lineWidth = 0
			}
			add(g)
		}
		word = nil
	}
	column := 0
	for _, r := range rs {
		if r.Link != "" && !renderhtml.SafeURL(r.Link) {
			b.warn(id, "unsafe link target omitted; label remains visible")
			r.Link = ""
		}
		style := r.Style
		if pre {
			style.Mono = true
		}
		if r.Unsupported {
			b.warn(id, "unsupported inline source rendered literally")
		}
		if b.customMono && style.Mono && (style.Bold || style.Italic) {
			b.warn(id, "custom mono face has no style variants; using the requested mono face")
		}
		if !utf8.ValidString(r.Text) {
			b.warn(id, "invalid UTF-8 replaced visibly; original source retained")
		}
		for _, c := range r.Text {
			if b.err != nil {
				return
			}
			hard := c == '\n' && (pre || r.Break)
			if hard {
				flushWord()
				flush(true)
				lineWidth = 0
				column = 0
				continue
			}
			if c == '\r' && !pre {
				c = ' '
			}
			if !pre && (c == '\n' || c == '\t' || c == '\r' || c == '\f') {
				c = ' '
			}
			chars := string(c)
			source := string(c)
			if pre && c == '\t' {
				chars = strings.Repeat(" ", 4-column%4)
			}
			if (unicode.IsControl(c) || unicode.Is(unicode.Cf, c)) && c != '\t' {
				b.warn(id, fmt.Sprintf("control U+%04X rendered as visible label", c))
				chars = fmt.Sprintf("[U+%04X]", c)
			}
			if unicode.Is(unicode.Mn, c) || unicode.Is(unicode.Arabic, c) || unicode.Is(unicode.Hebrew, c) {
				b.warn(id, "complex text shaping and bidi are unsupported; logical-order glyphs shown")
			}
			gs := b.measure(id, chars, source, style, scale, r.Link, r.Reference)
			if !pre && c == ' ' {
				flushWord()
				if len(gs) > 0 {
					g := gs[0]
					pending = &g
				}
				continue
			}
			if pre {
				for _, g := range gs {
					if g.advance > limit {
						b.err = fmt.Errorf("glyph cannot fit drawable width")
						return
					}
					if lineWidth+g.advance > limit && len(line) > 0 {
						b.warn(id, "preformatted line wrapped without clipping")
						flush(false)
						lineWidth = 0
					}
					add(g)
				}
			} else {
				word = append(word, gs...)
			}
			column += utf8.RuneCountInString(chars)
		}
	}
	flushWord()
	if len(line) > 0 {
		flush(false)
	} else if len(rs) == 0 {
		flush(false)
	}
}
func (b *layouter) measure(id man.ID, text, source string, style man.Style, scale float64, link, reference string) []glyph {
	k := key(style, b.plan.o.FontSize*scale)
	f, err := b.book.get(k)
	if err != nil {
		b.err = err
		return nil
	}
	var out []glyph
	first := true
	for _, c := range text {
		bounds, advance, ok := f.GlyphBounds(c)
		if !ok {
			b.warn(id, fmt.Sprintf("font lacks U+%04X; visible code-point label substituted", c))
			fallback := fmt.Sprintf("[U+%04X]", c)
			// The fallback label uses a bundled face, independent of custom coverage.
			fallbackKey := faceKey{"fallback", k.size}
			if b.book.data["fallback"] == nil {
				b.book.data["fallback"] = fontset.Default().Mono
				b.plan.data["fallback"] = b.book.data["fallback"]
			}
			ff, e := b.book.get(fallbackKey)
			if e != nil {
				b.err = e
				return nil
			}
			for _, r := range fallback {
				bb, aa, _ := ff.GlyphBounds(r)
				s := ""
				if first {
					s = source
					first = false
				}
				out = append(out, b.measured(r, s, fallbackKey, style, link, reference, bb, aa))
			}
			continue
		}
		s := ""
		if first {
			s = source
			first = false
		}
		out = append(out, b.measured(c, s, k, style, link, reference, bounds, advance))
	}
	return out
}
func (b *layouter) measured(c rune, source string, k faceKey, s man.Style, link, ref string, bounds fixed.Rectangle26_6, advance fixed.Int26_6) glyph {
	b.glyphs++
	if b.glyphs > b.plan.o.MaxGlyphs {
		b.err = ErrLimit
	}
	minX := min(fixed.Int26_6(0), bounds.Min.X)
	maxX := max(advance, bounds.Max.X)
	return glyph{char: c, source: source, face: k, style: s, link: link, reference: ref, x: -minX, advance: maxX - minX, bounds: bounds}
}
func (b *layouter) line(id man.ID, kind man.Kind, gs []glyph, left, right int, scale float64, hard bool) {
	if b.err != nil {
		return
	}
	f, err := b.book.get(key(man.Style{}, b.plan.o.FontSize*scale))
	if err != nil {
		b.err = err
		return
	}
	metrics := f.Metrics()
	ascent, descent := metrics.Ascent.Ceil(), metrics.Descent.Ceil()
	height := metrics.Height.Ceil()
	for _, g := range gs {
		face, e := b.book.get(g.face)
		if e != nil {
			b.err = e
			return
		}
		m := face.Metrics()
		ascent = max(ascent, m.Ascent.Ceil(), (-g.bounds.Min.Y).Ceil())
		descent = max(descent, m.Descent.Ceil(), g.bounds.Max.Y.Ceil())
		height = max(height, m.Height.Ceil())
	}
	height = max(height, ascent+descent)
	height = int(math.Ceil(float64(height) * b.plan.o.LineHeight))
	height = max(height, 1)
	o := b.plan.o
	bottom := o.PageHeight - o.Margin
	if o.Single {
		bottom = o.MaxHeight - o.Margin
	}
	if height > bottom-o.Margin {
		b.err = fmt.Errorf("text line cannot fit page height; reduce font size or margins")
		return
	}
	gap := b.pendingGap
	if b.y == o.Margin {
		gap = 0
	}
	needed := height
	if !o.Single && (kind == man.Section || kind == man.Subsection) {
		face, e := b.book.get(key(man.Style{}, o.FontSize))
		if e != nil {
			b.err = e
			return
		}
		m := face.Metrics()
		following := int(math.Ceil(float64(max(m.Height.Ceil(), m.Ascent.Ceil()+m.Descent.Ceil())) * o.LineHeight))
		if height+following <= bottom-o.Margin {
			needed += following
		}
	}
	if b.y+gap+needed > bottom {
		if o.Single {
			b.err = ErrLimit
			return
		}
		b.newPage()
		gap = 0
	}
	if b.err != nil {
		return
	}
	b.y += gap
	b.pendingGap = 0
	baseline := b.y + (height-ascent-descent)/2 + ascent
	info := Line{Node: id, Kind: kind, Top: b.y, Baseline: baseline, Bottom: b.y + height, HardBreak: hard}
	var cursor fixed.Int26_6
	var text, source strings.Builder
	var current Fragment
	active := false
	flushFragment := func() {
		if active {
			current.Text = text.String()
			current.Source = source.String()
			info.Fragments = append(info.Fragments, current)
			text = strings.Builder{}
			source = strings.Builder{}
			active = false
		}
	}
	for i := range gs {
		g := &gs[i]
		g.x += fixed.I(left) + cursor
		cursor += g.advance
		bounds := image.Rect((g.x + g.bounds.Min.X).Floor(), baseline+g.bounds.Min.Y.Floor(), (g.x + g.bounds.Max.X).Ceil(), baseline+g.bounds.Max.Y.Ceil())
		face := g.face.label()
		if active && (current.Face != face || current.Style != g.style || current.Link != g.link || current.Reference != g.reference) {
			flushFragment()
		}
		if !active {
			current = Fragment{Face: face, Style: g.style, Link: g.link, Reference: g.reference, X: float64(g.x) / 64, Bounds: bounds}
			active = true
		}
		text.WriteRune(g.char)
		source.WriteString(g.source)
		current.Advance += float64(g.advance) / 64
		current.Bounds = current.Bounds.Union(bounds)
	}
	flushFragment()

	// A line owns the glyph slice; subsequent wrapping starts a new slice.
	pg := &b.plan.pages[len(b.plan.pages)-1]
	pg.lines = append(pg.lines, paintedLine{info: info, glyphs: gs, baseline: fixed.I(baseline), left: left, right: right})
	b.y += height
	b.lastKind = kind
}
