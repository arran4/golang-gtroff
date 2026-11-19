package groff_test

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	groff "github.com/arran4/golang-gtroff"
	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/renderhtml"
	"github.com/arran4/golang-gtroff/renderimage"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/scenarios testdata/golden/scenarios examples/man-tool.1
var integrationCases embed.FS
var updateIntegration = flag.Bool("update-integration", false, "explicitly refresh cross-feature TXTAR and binary goldens")

type integrationOptions struct {
	Example        bool     `json:"example,omitempty"`
	NoFinalNewline bool     `json:"noFinalNewline,omitempty"`
	CRLF           bool     `json:"crlf,omitempty"`
	SkipLoopReason string   `json:"skipLoopReason,omitempty"`
	Retain         []string `json:"retain,omitempty"`
	Status         string   `json:"status"`
	ValidateCode   int      `json:"validateCode"`
	StrictCode     int      `json:"strictCode"`
	Server         bool     `json:"server,omitempty"`
	ServerQuery    string   `json:"serverQuery,omitempty"`
	ServerHeading  string   `json:"serverHeading,omitempty"`
	ServerText     string   `json:"serverText,omitempty"`
	Snapshot       string   `json:"snapshot,omitempty"`
	PageHeight     int      `json:"pageHeight,omitempty"`
	MinPages       int      `json:"minPages,omitempty"`
}

// Each case crosses the same public boundaries. Feature-specific suites retain
// detailed API contracts; this suite checks agreement between those boundaries.
func TestIntegrationScenarios(t *testing.T) {
	var paths []string
	mustIntegration(t, fs.WalkDir(integrationCases, "testdata/scenarios", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".txtar") {
			paths = append(paths, p)
		}
		return err
	}))
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no integration scenarios discovered")
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/scenarios/"), func(t *testing.T) {
			raw, err := integrationCases.ReadFile(path)
			mustIntegration(t, err)
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				switch f.Name {
				case "options.json", "input.1", "input.html", "expected-ast.json", "expected-validation.json", "expected-semantics.json", "expected.html", "expected.1", "expected-layout.json", "expected-diagnostics.json":
				default:
					t.Fatalf("unrecognized archive member %s", f.Name)
				}
				if _, ok := files[f.Name]; ok {
					t.Fatalf("duplicate %s", f.Name)
				}
				files[f.Name] = f.Data
			}
			var o integrationOptions
			decoder := json.NewDecoder(bytes.NewReader(files["options.json"]))
			decoder.DisallowUnknownFields()
			mustIntegration(t, decoder.Decode(&o))
			if o.Status != "valid" && o.Status != "partial" && o.Status != "invalid" {
				t.Fatal("explicit expected validation status required")
			}
			if o.Snapshot != "" && o.Snapshot != "png" && o.Snapshot != "jpg" {
				t.Fatal("snapshot must be png or jpg")
			}
			source, roffOK := files["input.1"]
			htmlSource, htmlOK := files["input.html"]
			if o.Example {
				if roffOK || htmlOK {
					t.Fatal("example and inline source are exclusive")
				}
				source, err = integrationCases.ReadFile("examples/man-tool.1")
				mustIntegration(t, err)
				roffOK = true
			}
			if roffOK == htmlOK {
				t.Fatal("require exactly one explicit source")
			}
			if o.NoFinalNewline {
				source = bytes.TrimSuffix(source, []byte("\n"))
			}
			if o.CRLF {
				source = bytes.ReplaceAll(source, []byte("\n"), []byte("\r\n"))
			}
			var doc *man.Document
			notes := map[string][]groff.Diagnostic{}
			note := func(stage string, ds []groff.Diagnostic) {
				if len(ds) > 0 {
					notes[stage] = ds
				}
			}
			if htmlOK {
				imported, e := conversion.ImportHTML(bytes.NewReader(htmlSource), conversion.Options{SourceName: "input.html"})
				mustIntegration(t, e)
				doc = imported.Document
				note("input-html", doc.Diagnostics)
				if imported.Source != string(htmlSource) {
					t.Fatal("HTML source changed")
				}
				var canonical bytes.Buffer
				_, e = conversion.WriteMan(&canonical, doc, conversion.Options{})
				mustIntegration(t, e)
				source = canonical.Bytes()
			}
			tree, err := groff.ParseAST(bytes.NewReader(source), groff.WithPackage("man"), groff.WithSourceName("input.1"))
			mustIntegration(t, err)
			if tree.Source() != string(source) || tree.Root().Raw() != string(source) {
				t.Fatal("AST lost source bytes")
			}
			if doc == nil {
				doc, err = man.Build(tree, man.Options{})
				mustIntegration(t, err)
			}
			mustIntegration(t, doc.Validate())
			if roffOK && doc.Source() != string(source) {
				t.Fatal("semantic source differs from raw snapshot")
			}
			before, err := doc.Canonical()
			mustIntegration(t, err)
			note("ast", tree.Diagnostics())
			note("semantic", doc.Diagnostics)
			type row struct {
				ID         groff.NodeID
				Kind       groff.ASTKind
				Path, Text string
			}
			var rows []row
			seen := map[groff.NodeID]bool{}
			tree.Root().Walk(func(n *groff.ASTNode) groff.WalkAction {
				if seen[n.ID()] || tree.Node(n.ID()) != n {
					t.Fatal("unstable/repeated node")
				}
				seen[n.ID()] = true
				if n.Parent() != nil && !seen[n.Parent().ID()] {
					t.Fatal("walk is not preorder")
				}
				span := n.Span()
				if n.Raw() != string(source[span.Start.Offset:span.End.Offset]) {
					t.Fatal("raw extraction mismatch")
				}
				rows = append(rows, row{n.ID(), n.Kind(), n.Path(), n.Text()})
				return groff.WalkContinue
			})
			actual := map[string][]byte{"expected-ast.json": integrationJSON(t, rows)}
			type validation struct {
				Strict      bool
				Exit        int
				Status      string
				Diagnostics []groff.Diagnostic
			}
			validations := []validation{}
			for _, strict := range []bool{false, true} {
				args := []string{"validate", "--format", "json"}
				if strict {
					args = append(args, "--strict")
				}
				var out, stderr bytes.Buffer
				code := cli.Run(args, cli.Environment{In: bytes.NewReader(source), Out: &out, Err: &stderr})
				var v struct {
					Status      string
					Diagnostics []groff.Diagnostic
				}
				mustIntegration(t, json.Unmarshal(out.Bytes(), &v))
				want := o.ValidateCode
				if strict {
					want = o.StrictCode
				}
				if code != want || stderr.Len() != 0 {
					t.Fatalf("validate strict=%v: exit %d want %d: %s", strict, code, want, stderr.String())
				}
				if !strict && v.Status != o.Status {
					t.Fatalf("validate status %s want %s", v.Status, o.Status)
				}
				validations = append(validations, validation{strict, code, v.Status, v.Diagnostics})
			}
			actual["expected-validation.json"] = integrationJSON(t, validations)
			sem, err := conversion.Normalize(doc)
			mustIntegration(t, err)
			actual["expected-semantics.json"] = integrationJSON(t, sem)
			var rendered, canonical bytes.Buffer
			res, err := renderhtml.Render(&rendered, doc, renderhtml.Options{Fragment: true, Assets: renderhtml.OmitFonts})
			mustIntegration(t, err)
			note("html", res.Diagnostics)
			written, err := conversion.WriteMan(&canonical, doc, conversion.Options{})
			mustIntegration(t, err)
			note("canonical", written.Diagnostics)
			actual["expected.html"] = rendered.Bytes()
			actual["expected.1"] = canonical.Bytes()
			if o.SkipLoopReason == "" {
				assertIntegrationLoop(t, doc, &rendered, &canonical)
			} else if len(o.Retain) == 0 {
				t.Fatal("loop exception requires explicit retained content assertions")
			}
			reparse, err := conversion.ImportMan(bytes.NewReader(canonical.Bytes()), conversion.Options{})
			mustIntegration(t, err)
			// Even literal fallback cases must not turn stored controls into live
			// metadata or headings when canonical roff is parsed again.
			if reparse.Document.Metadata != doc.Metadata || !reflect.DeepEqual(integrationHeadings(doc), integrationHeadings(reparse.Document)) {
				t.Fatal("canonical import changed page identity or headings")
			}
			for _, literal := range o.Retain {
				for stage, text := range map[string]string{"semantic": doc.Text(1, man.TextOptions{}), "HTML": html.UnescapeString(rendered.String()), "canonical": reparse.Document.Text(1, man.TextOptions{})} {
					if !strings.Contains(text, literal) {
						t.Errorf("%s dropped %q", stage, literal)
					}
				}
			}
			// Layout summaries stay readable; encoded goldens stay binary and opt-in.
			height := o.PageHeight
			if height == 0 {
				height = 320
			}
			format := o.Snapshot
			if format == "" {
				format = "png"
			}
			rasterOptions := renderimage.Options{Width: 320, Margin: 16, FontSize: 10, PageHeight: height, Format: format}
			plan, err := renderimage.Layout(doc, rasterOptions)
			mustIntegration(t, err)
			assertImageContent(t, doc, rasterOptions, plan.Pages())
			note("image", plan.Result().Diagnostics)
			type page struct {
				Number, Width, Height int
				Text                  []string
			}
			pages := []page{}
			for _, p := range plan.Pages() {
				v := page{Number: p.Number, Width: p.Width, Height: p.Height}
				for _, l := range p.Lines {
					var s strings.Builder
					for _, f := range l.Fragments {
						s.WriteString(f.Source)
					}
					v.Text = append(v.Text, s.String())
				}
				pages = append(pages, v)
			}
			if len(pages) < max(1, o.MinPages) {
				t.Fatalf("got %d pages, need %d", len(pages), o.MinPages)
			}
			actual["expected-layout.json"] = integrationJSON(t, pages)
			_, err = plan.Encode(func(p renderimage.Page, data []byte) error {
				im, _, e := image.Decode(bytes.NewReader(data))
				if e != nil {
					return e
				}
				if im.Bounds() != image.Rect(0, 0, p.Width, p.Height) {
					t.Fatal("encoded dimensions differ")
				}
				if o.Snapshot != "" {
					name := filepath.ToSlash(filepath.Join("testdata/golden/scenarios", strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(path, "testdata/scenarios/"), ".txtar"), "/", "-")+fmt.Sprintf("-%03d.%s", p.Number, format)))
					if *updateIntegration {
						mustIntegration(t, os.WriteFile(name, data, 0644))
					} else {
						want, e := integrationCases.ReadFile(name)
						mustIntegration(t, e)
						if !bytes.Equal(want, data) {
							t.Errorf("binary mismatch %s; refresh with -update-integration", name)
						}
					}
				}
				return nil
			})
			mustIntegration(t, err)
			if o.Server {
				assertIntegrationServer(t, source, o.ServerQuery, o.ServerHeading, o.ServerText)
			}
			after, err := doc.Canonical()
			mustIntegration(t, err)
			if !bytes.Equal(before, after) || tree.Source() != string(source) {
				t.Fatal("pipeline mutated source/model")
			}
			if len(notes) > 0 {
				actual["expected-diagnostics.json"] = integrationJSON(t, notes)
			} else if d, ok := files["expected-diagnostics.json"]; ok && strings.TrimSpace(string(d)) != "{}" {
				t.Error("expected diagnostics but none produced")
			}
			names := make([]string, 0, len(actual))
			for name := range actual {
				names = append(names, name)
			}
			sort.Strings(names)
			if *updateIntegration {
				next := &txtar.Archive{Comment: ar.Comment}
				for _, f := range ar.Files {
					if !strings.HasPrefix(f.Name, "expected") {
						next.Files = append(next.Files, f)
					}
				}
				for _, name := range names {
					next.Files = append(next.Files, txtar.File{Name: name, Data: actual[name]})
				}
				mustIntegration(t, os.WriteFile(path, txtar.Format(next), 0644))
			} else {
				for _, name := range names {
					if !bytes.Equal(files[name], actual[name]) {
						t.Errorf("%s mismatch; inspect and refresh with -update-integration\ngot:\n%s", name, actual[name])
					}
				}
			}
		})
	}
}

func mustIntegration(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func integrationHeadings(d *man.Document) []string {
	var headings []string
	for _, n := range d.Nodes {
		if n.Kind == man.Section || n.Kind == man.Subsection {
			headings = append(headings, string(n.Kind)+":"+n.Heading)
		}
	}
	return headings
}
func integrationJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.MarshalIndent(v, "", "  ")
	mustIntegration(t, e)
	return append(b, '\n')
}
func assertIntegrationLoop(t *testing.T, original *man.Document, rendered, canonical *bytes.Buffer) {
	t.Helper()
	want, e := conversion.Normalize(original)
	mustIntegration(t, e)
	imported, e := conversion.ImportHTML(bytes.NewReader(rendered.Bytes()), conversion.Options{})
	mustIntegration(t, e)
	parsed, e := conversion.ImportMan(bytes.NewReader(canonical.Bytes()), conversion.Options{})
	mustIntegration(t, e)
	var nextHTML, nextRoff bytes.Buffer
	_, e = conversion.WriteMan(&nextRoff, imported.Document, conversion.Options{})
	mustIntegration(t, e)
	fromHTML, e := conversion.ImportMan(&nextRoff, conversion.Options{})
	mustIntegration(t, e)
	_, e = renderhtml.Render(&nextHTML, parsed.Document, renderhtml.Options{Fragment: true, Assets: renderhtml.OmitFonts})
	mustIntegration(t, e)
	final, e := conversion.ImportHTML(bytes.NewReader(nextHTML.Bytes()), conversion.Options{})
	mustIntegration(t, e)
	for _, d := range []*man.Document{imported.Document, parsed.Document, fromHTML.Document, final.Document} {
		got, e := conversion.Normalize(d)
		mustIntegration(t, e)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("semantic loop changed content\nwant %s\ngot %s", integrationJSON(t, want), integrationJSON(t, got))
		}
	}
	var again bytes.Buffer
	_, e = conversion.WriteMan(&again, parsed.Document, conversion.Options{})
	mustIntegration(t, e)
	if !bytes.Equal(canonical.Bytes(), again.Bytes()) {
		t.Fatal("second canonical roff differs")
	}
	var stableHTML bytes.Buffer
	// HTML includes source provenance. Canonical HTML is rendered from stable
	// canonical roff so byte equality does not compare unrelated input offsets.
	again.Reset()
	_, e = conversion.WriteMan(&again, final.Document, conversion.Options{})
	mustIntegration(t, e)
	if !bytes.Equal(canonical.Bytes(), again.Bytes()) {
		t.Fatal("HTML loop changed canonical roff")
	}
	stable, e := conversion.ImportMan(&again, conversion.Options{})
	mustIntegration(t, e)
	_, e = renderhtml.Render(&stableHTML, stable.Document, renderhtml.Options{Fragment: true, Assets: renderhtml.OmitFonts})
	mustIntegration(t, e)
	if !bytes.Equal(nextHTML.Bytes(), stableHTML.Bytes()) {
		a, b := nextHTML.Bytes(), stableHTML.Bytes()
		i := 0
		for i < len(a) && i < len(b) && a[i] == b[i] {
			i++
		}
		t.Fatalf("second canonical HTML differs at %d\nfirst: %s\nsecond: %s", i, a[max(0, i-100):min(len(a), i+180)], b[max(0, i-100):min(len(b), i+180)])
	}
}
func assertIntegrationServer(t *testing.T, source []byte, query, heading, text string) {
	t.Helper()
	if query == "" {
		query = "inspect"
	}
	if heading == "" {
		heading = "OPTIONS"
	}
	if text == "" {
		text = "Display verbose diagnostics"
	}
	idx, e := manlib.Build(context.Background(), []manlib.Root{{Label: "demo", FS: fstest.MapFS{"man1/man-tool.1": &fstest.MapFile{Data: source}}}}, manlib.Limits{})
	mustIntegration(t, e)
	hits, e := idx.Search(context.Background(), query, 10)
	mustIntegration(t, e)
	if len(hits) != 1 || hits[0].Name != "man-tool" {
		t.Fatalf("search lost example: %+v", hits)
	}
	handler, e := manlib.Handler(idx, manlib.HandlerOptions{Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	mustIntegration(t, e)
	for _, tc := range []struct{ url, contains string }{{"/", "man-tool"}, {"/search?q=" + url.QueryEscape(query), "man-tool"}, {"/page/1/man-tool?root=demo", heading}, {"/page/1/man-tool?root=demo&heading=" + url.QueryEscape(heading), text}} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest("GET", "http://localhost"+tc.url, nil))
		if r.Code != 200 || !strings.Contains(r.Body.String(), tc.contains) {
			t.Fatalf("%s status=%d missing %q", tc.url, r.Code, tc.contains)
		}
	}
	r := httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest("GET", "http://localhost/page/1/missing", nil))
	if r.Code != 404 {
		t.Fatalf("missing page returned %d", r.Code)
	}
}
