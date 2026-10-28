package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/convert/*.txtar
var convertCases embed.FS
var updateConvert = flag.Bool("update-convert", false, "explicitly update semantic conversion scenario expectations")

func TestConvertScenarios(t *testing.T) {
	paths, err := fs.Glob(convertCases, "testdata/convert/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/convert/"), func(t *testing.T) {
			raw, err := convertCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate archive file")
				}
				files[f.Name] = f.Data
			}
			var tc struct {
				Equivalent bool     `json:"equivalent"`
				Retained   []string `json:"retained"`
			}
			if err := json.Unmarshal(files["case.json"], &tc); err != nil {
				t.Fatal(err)
			}
			if !tc.Equivalent && len(tc.Retained) == 0 {
				t.Fatal("fallback scenario requires explicit retained-content assertions")
			}
			o := conversion.Options{SourceName: "input.html", HTML: renderhtml.Options{Assets: renderhtml.OmitFonts}}
			var input conversion.Input
			if source, ok := files["input.1"]; ok {
				o.SourceName = "input.1"
				original, e := conversion.ImportMan(bytes.NewReader(source), o)
				if e != nil {
					t.Fatal(e)
				}
				if original.Source != string(source) || original.Document.Source() != string(source) {
					t.Fatal("raw roff changed")
				}
				// Actual exporter bytes, including its styled fragment, enter the importer.
				var exported bytes.Buffer
				_, e = renderhtml.Render(&exported, original.Document, renderhtml.Options{Fragment: true, Assets: renderhtml.OmitFonts})
				if e != nil {
					t.Fatal(e)
				}
				input, e = conversion.ImportHTML(&exported, o)
				if e != nil {
					t.Fatal(e)
				}
				if tc.Equivalent {
					assertConversionSemantics(t, original.Document, input.Document)
				}
			} else {
				source, ok := files["input.html"]
				if !ok {
					t.Fatal("missing input")
				}
				input, err = conversion.ImportHTML(bytes.NewReader(source), o)
				if err != nil {
					t.Fatal(err)
				}
				if input.Source != string(source) || input.Document.Source() != "" {
					t.Fatal("original HTML mixed with roff source")
				}
			}
			originalJSON, err := json.Marshal(input.Document)
			if err != nil {
				t.Fatal(err)
			}
			semantics, err := conversion.Normalize(input.Document)
			if err != nil {
				t.Fatal(err)
			}
			var canonical bytes.Buffer
			res, err := conversion.WriteMan(&canonical, input.Document, o)
			if err != nil {
				t.Fatal(err)
			}
			afterJSON, _ := json.Marshal(input.Document)
			if !bytes.Equal(originalJSON, afterJSON) {
				t.Fatal("canonical writer mutated source model")
			}
			parsed, err := conversion.ImportMan(bytes.NewReader(canonical.Bytes()), conversion.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if len(parsed.Document.Diagnostics) > 0 {
				t.Fatalf("canonical source has diagnostics: %+v\n%s", parsed.Document.Diagnostics, canonical.String())
			}
			if tc.Equivalent {
				if len(res.Diagnostics) > 0 {
					t.Fatalf("supported input has diagnostics: %+v", res.Diagnostics)
				}
				assertConversionSemantics(t, input.Document, parsed.Document)
			} else if len(res.Diagnostics) == 0 {
				t.Fatal("unsupported input lost diagnostics")
			}
			for _, s := range tc.Retained {
				if !strings.Contains(parsed.Document.Text(1, man.TextOptions{}), s) {
					t.Errorf("canonical source dropped %q", s)
				}
			}
			var again bytes.Buffer
			_, err = conversion.WriteMan(&again, parsed.Document, conversion.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(canonical.Bytes(), again.Bytes()) {
				t.Fatalf("second canonicalization unstable\nFIRST\n%s\nSECOND\n%s", canonical.String(), again.String())
			}
			var exported bytes.Buffer
			_, err = renderhtml.Render(&exported, parsed.Document, renderhtml.Options{Assets: renderhtml.OmitFonts})
			if err != nil {
				t.Fatal(err)
			}
			imported, err := conversion.ImportHTML(bytes.NewReader(exported.Bytes()), conversion.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if len(imported.Document.Diagnostics) > 0 {
				t.Fatalf("canonical HTML import diagnosed: %+v", imported.Document.Diagnostics)
			}
			assertConversionSemantics(t, parsed.Document, imported.Document)
			semJSON, _ := json.MarshalIndent(semantics, "", "  ")
			semJSON = append(semJSON, '\n')
			ds := res.Diagnostics
			if ds == nil {
				ds = []groff.Diagnostic{}
			}
			diagJSON, _ := json.MarshalIndent(ds, "", "  ")
			diagJSON = append(diagJSON, '\n')
			actual := map[string][]byte{"expected.1": canonical.Bytes(), "expected.html": exported.Bytes(), "expected-semantics.json": semJSON, "expected-diagnostics.json": diagJSON}
			for name, got := range actual {
				if !*updateConvert && !bytes.Equal(got, files[name]) {
					t.Errorf("%s mismatch; update explicitly with -update-convert\ngot:\n%s", name, got)
				}
			}
			if *updateConvert {
				next := &txtar.Archive{Comment: ar.Comment}
				for _, f := range ar.Files {
					if _, ok := actual[f.Name]; !ok {
						next.Files = append(next.Files, f)
					}
				}
				for _, name := range []string{"expected.1", "expected.html", "expected-semantics.json", "expected-diagnostics.json"} {
					next.Files = append(next.Files, txtar.File{Name: name, Data: actual[name]})
				}
				if err := os.WriteFile(path, txtar.Format(next), 0644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func assertConversionSemantics(t *testing.T, a, b *man.Document) {
	t.Helper()
	x, e := conversion.Normalize(a)
	if e != nil {
		t.Fatal(e)
	}
	y, e := conversion.Normalize(b)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(x, y) {
		j, _ := json.MarshalIndent(x, "", "  ")
		k, _ := json.MarshalIndent(y, "", "  ")
		t.Fatalf("semantic conversion changed structure/text\n%s\nBECAME\n%s", j, k)
	}
}

//go:embed examples/convert.html examples/convert.1
var convertExample embed.FS

func TestConvertCLIExample(t *testing.T) {
	source, err := convertExample.ReadFile("examples/convert.html")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := convertExample.ReadFile("examples/convert.1")
	if err != nil {
		t.Fatal(err)
	}
	var out, diags bytes.Buffer
	code := cli.Run([]string{"convert", "--from", "html", "--to", "roff", "--strict"}, cli.Environment{In: bytes.NewReader(source), Out: &out, Err: &diags})
	if code != 0 || diags.Len() != 0 || !bytes.Equal(out.Bytes(), expected) {
		t.Fatalf("CLI example differs: exit=%d stderr=%s\n%s", code, diags.String(), out.String())
	}
}
