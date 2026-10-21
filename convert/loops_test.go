package convert_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/renderhtml"
)

func TestLoops(t *testing.T) {
	sources := []string{
		".SH FLOW\n.ft B\nbold\n.br\nnext\n",
		".TH TOOL 1 date source manual\n.SH NAME\ntool \\- handy\n.SH DESCRIPTION\nplain\n.B bold\n.I italic\nplain\n.br\nnext\n.SS DETAIL\n.TP 8\n.B --flag\nflag body\n.RS 2\nnested indent\n.RE\n.TP\n.B --other\nother body\n.PP\nafter\n.EX\n  x\n y\n.EE\n",
		".TH \"Title \\(dqquoted\\(dq \\e path\" 1\n.SH NAME\nA\\~B 漢字 café\n.UR https://example.com/x?q=a&b=c\nlink\n.UE\n.MR printf 3 ,\n.SM small\n",
		".SH PRE\n.nf\n\n first\n\nlast\\c\n.fi\n",
	}
	for _, source := range sources {
		t.Run(source[:min(len(source), 20)], func(t *testing.T) {
			original, e := conversion.ImportMan(strings.NewReader(source), conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			var html bytes.Buffer
			_, e = renderhtml.Render(&html, original.Document, renderhtml.Options{Fragment: true, Assets: renderhtml.OmitFonts})
			if e != nil {
				t.Fatal(e)
			}
			imported, e := conversion.ImportHTML(&html, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(imported.Document.Diagnostics) > 0 {
				t.Fatalf("unexpected HTML diagnostics: %+v", imported.Document.Diagnostics)
			}
			var roff bytes.Buffer
			res, e := conversion.WriteMan(&roff, imported.Document, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(res.Diagnostics) > 0 {
				t.Fatalf("unexpected write diagnostics: %+v", res.Diagnostics)
			}
			final, e := conversion.ImportMan(strings.NewReader(roff.String()), conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(final.Document.Diagnostics) > 0 {
				t.Fatalf("unexpected canonical diagnostics: %+v\n%s", final.Document.Diagnostics, roff.String())
			}
			a, _ := conversion.Normalize(original.Document)
			b, _ := conversion.Normalize(imported.Document)
			c, _ := conversion.Normalize(final.Document)
			if !reflect.DeepEqual(a, b) {
				t.Errorf("HTML changed semantics\noriginal %#v\nimported %#v", a, b)
			}
			if !reflect.DeepEqual(a, c) {
				t.Errorf("roff changed semantics\noriginal %#v\nfinal %#v\n%s", a, c, roff.String())
			}
			var again bytes.Buffer
			_, e = conversion.WriteMan(&again, final.Document, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if roff.String() != again.String() {
				t.Errorf("canonical not stable\n%s\nAGAIN\n%s", roff.String(), again.String())
			}
		})
	}
}

func TestHTMLLoops(t *testing.T) {
	for _, source := range []string{
		`<h1>TOOL</h1><h2>NAME</h2><p>A <strong>bold <em>both <code>mono</code></em></strong> word<br><br>Next <small>small <b>bold</b></small></p><h3>DETAIL</h3><p><a href="https://example.com?q=a&amp;b=c">link</a> after</p>`,
		`<h1>List</h1><dl><dt><b>--foo</b></dt><dd><p>first</p><p>second</p><ul><li>nested <em>item</em></li><li>two</li></ul><p>after list</p></dd><dt>--bar</dt><dd>other</dd></dl><p>end</p>`,
		`<h1>Unicode &quot;漢字&quot; \</h1><h2>Code</h2><pre><code>  .danger
'also
\literal &lt;tag&gt;
last</code></pre><ol start="3"><li>three</li><li>four</li></ol>`,
		`<p></p><p>one</p><p></p><p>.danger<br>'request<br>\f[B] &quot;quotes&quot; &amp; x&nbsp;y</p>`,
	} {
		t.Run(source[:min(len(source), 20)], func(t *testing.T) {
			original, e := conversion.ImportHTML(strings.NewReader(source), conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(original.Document.Diagnostics) > 0 {
				t.Fatalf("diagnostics %+v", original.Document.Diagnostics)
			}
			var roff bytes.Buffer
			res, e := conversion.WriteMan(&roff, original.Document, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(res.Diagnostics) > 0 {
				t.Fatalf("write diagnostics %+v", res.Diagnostics)
			}
			parsed, e := conversion.ImportMan(strings.NewReader(roff.String()), conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if len(parsed.Document.Diagnostics) > 0 {
				t.Fatalf("parse diagnostics %+v\n%s", parsed.Document.Diagnostics, roff.String())
			}
			a, _ := conversion.Normalize(original.Document)
			b, _ := conversion.Normalize(parsed.Document)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("roff changed semantics\n%#v\n%#v\n%s", a, b, roff.String())
			}
			var html bytes.Buffer
			_, e = conversion.ManToHTML(strings.NewReader(roff.String()), &html, conversion.Options{HTML: renderhtml.Options{Assets: renderhtml.OmitFonts}})
			if e != nil {
				t.Fatal(e)
			}
			imported, e := conversion.ImportHTML(&html, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			c, _ := conversion.Normalize(imported.Document)
			if !reflect.DeepEqual(a, c) {
				t.Errorf("HTML changed semantics\n%#v\n%#v", a, c)
			}
			var again bytes.Buffer
			_, e = conversion.WriteMan(&again, parsed.Document, conversion.Options{})
			if e != nil {
				t.Fatal(e)
			}
			if roff.String() != again.String() {
				t.Errorf("canonical unstable\n%s\nAGAIN\n%s", roff.String(), again.String())
			}
		})
	}
}
