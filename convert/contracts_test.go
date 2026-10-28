package convert_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/man"
)

func TestActualExportedExample(t *testing.T) {
	roff, err := os.ReadFile("../examples/man-tool.1")
	if err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile("../examples/man-tool.html")
	if err != nil {
		t.Fatal(err)
	}
	original, err := conversion.ImportMan(bytes.NewReader(roff), conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := conversion.ImportHTML(bytes.NewReader(exported), conversion.Options{Strict: true})
	if err != nil {
		t.Fatalf("actual bundled-font export: %v %+v", err, imported.Document.Diagnostics)
	}
	a, _ := conversion.Normalize(original.Document)
	b, _ := conversion.Normalize(imported.Document)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("actual export changed semantics\n%#v\n%#v", a, b)
	}
	var canonical bytes.Buffer
	if _, err := conversion.WriteMan(&canonical, imported.Document, conversion.Options{Strict: true}); err != nil {
		t.Fatal(err)
	}
	final, err := conversion.ImportMan(&canonical, conversion.Options{Strict: true})
	if err != nil {
		t.Fatalf("canonical example: %v %+v", err, final.Document.Diagnostics)
	}
	c, _ := conversion.Normalize(final.Document)
	if !reflect.DeepEqual(a, c) {
		t.Fatalf("canonical example changed semantics\n%#v\n%#v", a, c)
	}
}
func TestConversionLimitsAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		o            conversion.Options
	}{
		{"input", "<p>big</p>", conversion.Options{MaxInputBytes: 3}},
		{"depth", strings.Repeat("<div>", 8) + "x" + strings.Repeat("</div>", 8), conversion.Options{MaxDepth: 5}},
		{"nodes", "<p>a</p><p>b</p>", conversion.Options{MaxNodes: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			_, e := conversion.HTMLToMan(strings.NewReader(tc.source), &out, tc.o)
			if !errors.Is(e, conversion.ErrLimit) || out.Len() != 0 {
				t.Fatalf("budget wrote output or wrong error: %v %q", e, out.String())
			}
		})
	}
	var out bytes.Buffer
	res, err := conversion.HTMLToMan(strings.NewReader("<script>VISIBLE</script>"), &out, conversion.Options{Strict: true})
	if !errors.Is(err, conversion.ErrStrict) || out.Len() != 0 || len(res.Diagnostics) == 0 {
		t.Fatalf("strict result %v %+v %q", err, res, out.String())
	}
	_, err = conversion.HTMLToMan(strings.NewReader("<p>hello</p>"), &out, conversion.Options{MaxOutputBytes: 2})
	if !errors.Is(err, conversion.ErrLimit) || out.Len() != 0 {
		t.Fatalf("output budget: %v %q", err, out.String())
	}
	_, err = conversion.HTMLToMan(failingReader{}, &out, conversion.Options{})
	if !errors.Is(err, errRead) || out.Len() != 0 {
		t.Fatal("read error lost", err)
	}
	_, err = conversion.HTMLToMan(strings.NewReader("<p>x</p>"), shortWriter{}, conversion.Options{})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	_, err = conversion.HTMLToMan(strings.NewReader("<p>x</p>"), failingWriter{}, conversion.Options{})
	if !errors.Is(err, errWrite) {
		t.Fatal(err)
	}
	if _, err = conversion.ImportHTML(nil, conversion.Options{}); err == nil {
		t.Fatal("nil reader accepted")
	}
	if _, err = conversion.WriteMan(&out, nil, conversion.Options{}); err == nil {
		t.Fatal("nil model accepted")
	}
	in, err := conversion.ImportHTML(strings.NewReader("<p>x</p>"), conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	in.Document.Nodes[0].Children = append(in.Document.Nodes[0].Children, 1)
	if _, err = conversion.WriteMan(&out, in.Document, conversion.Options{}); err == nil {
		t.Fatal("cyclic model accepted")
	}
}

var errRead = errors.New("read failure")
var errWrite = errors.New("write failure")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRead }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

type shortWriter struct{}

func (shortWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }
func TestEscapingCannotCreateRequests(t *testing.T) {
	source := `<h1>.TH &quot;attack&quot; \evil</h1><p>.so /etc/passwd<br>'sy command<br>\&quot;comment</p><pre><code>.de X
.sy attack
..
</code></pre>`
	var out bytes.Buffer
	_, err := conversion.HTMLToMan(strings.NewReader(source), &out, conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	in, err := conversion.ImportMan(strings.NewReader(out.String()), conversion.Options{Strict: true})
	if err != nil {
		t.Fatalf("canonical source: %v %+v\n%s", err, in.Document.Diagnostics, out.String())
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, ".") && !strings.HasPrefix(line, ".TH ") && line != ".PP" && line != ".br" && line != ".nf" && line != ".fi" {
			t.Fatalf("unexpected executable control: %s", line)
		}
		if strings.HasPrefix(line, "'") {
			t.Fatal("unescaped apostrophe control")
		}
	}
}
func TestSpoofedExportChromeDoesNotHideScript(t *testing.T) {
	source := `<article data-man-version="1"><header class="document-header"><h1>Title<script>HEADER SCRIPT</script></h1></header><aside class="conversion-notes"><h2>Conversion notes</h2><script>ASIDE SCRIPT</script></aside><p><a class="permalink" href="#man-node-2"><script>LINK SCRIPT</script>#</a></p></article>`
	in, err := conversion.ImportHTML(strings.NewReader(source), conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Document.Diagnostics) == 0 {
		t.Fatal("spoofed chrome accepted silently")
	}
	var out bytes.Buffer
	_, err = conversion.WriteMan(&out, in.Document, conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"HEADER SCRIPT", "ASIDE SCRIPT", "LINK SCRIPT"} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("dropped %s", s)
		}
	}
}
func TestMetadataAndLinkEscapes(t *testing.T) {
	source := `<h1>quote &quot; \ path</h1><p><a href="https://example.com/path?q=%22&amp;x=\test">quoted link</a></p>`
	in, err := conversion.ImportHTML(strings.NewReader(source), conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err = conversion.WriteMan(&out, in.Document, conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := conversion.ImportMan(&out, conversion.Options{})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := conversion.Normalize(in.Document)
	b, _ := conversion.Normalize(parsed.Document)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("escaped metadata/link changed\n%#v\n%#v", a, b)
	}
}
func FuzzHTMLImport(f *testing.F) {
	for _, s := range []string{"<p>.danger</p>", "<script>x</script>", "<dl><dt>x<dd>y", `<h1>漢字</h1><pre>  x</pre>`, strings.Repeat("<div>", 130)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 16384 {
			t.Skip()
		}
		o := conversion.Options{MaxInputBytes: 16384, MaxOutputBytes: 1 << 20, MaxNodes: 2000, MaxDepth: 32}
		in, e := conversion.ImportHTML(strings.NewReader(s), o)
		if e != nil {
			return
		}
		if in.Source != s {
			t.Fatal("source changed")
		}
		if e = in.Document.Validate(); e != nil {
			t.Fatal(e)
		}
		var out bytes.Buffer
		if _, e = conversion.WriteMan(&out, in.Document, o); e != nil {
			return
		}
		parsed, e := conversion.ImportMan(&out, conversion.Options{MaxInputBytes: 1 << 20, MaxNodes: 10000})
		if e != nil {
			t.Fatal(e)
		}
		if e = parsed.Document.Validate(); e != nil {
			t.Fatal(e)
		}
	})
}

// Ensure outbound defaults still embed fonts and accept the strict profile.
func TestDefaultHTMLConversion(t *testing.T) {
	var out bytes.Buffer
	_, e := conversion.ManToHTML(strings.NewReader(".TH TOOL 1\n.SH NAME\ntext\n"), &out, conversion.Options{})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(out.Bytes(), []byte("data:font/ttf;base64,")) {
		t.Fatal("bundled fonts omitted")
	}
	in, e := conversion.ImportHTML(&out, conversion.Options{Strict: true})
	if e != nil {
		t.Fatalf("own output rejected %v %+v", e, in.Document.Diagnostics)
	}
}

func TestLongFlowTextLoop(t *testing.T) {
	text := strings.Repeat("界", 24000)
	var canonical bytes.Buffer
	if _, err := conversion.HTMLToMan(strings.NewReader("<p>"+text+"</p>"), &canonical, conversion.Options{}); err != nil {
		t.Fatal(err)
	}
	in, err := conversion.ImportMan(&canonical, conversion.Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSuffix(in.Document.Text(1, man.TextOptions{}), "\n") != text {
		t.Fatal("long paragraph changed")
	}
}
