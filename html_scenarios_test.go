package groff_test

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
	xhtml "golang.org/x/net/html"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/render/html/*.txtar examples/man-tool.1 examples/man-tool.html
var htmlCases embed.FS
var updateHTML = flag.Bool("update-html", false, "explicitly update HTML scenario and CLI example goldens")
var fontDataPattern = regexp.MustCompile(`data:font/(?:ttf|otf);base64,([A-Za-z0-9+/=]+)`)

func TestHTMLScenarios(t *testing.T) {
	var paths []string
	if err := fs.WalkDir(htmlCases, "testdata/render/html", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".txtar") {
			paths = append(paths, p)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		path := path
		t.Run(strings.TrimPrefix(path, "testdata/render/html/"), func(t *testing.T) {
			raw, err := htmlCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate file")
				}
				files[f.Name] = f.Data
			}
			input, ok := files["input.1"]
			if !ok {
				t.Fatal("missing input.1")
			}
			var o renderhtml.Options
			if data, ok := files["options.json"]; ok {
				if err := json.Unmarshal(data, &o); err != nil {
					t.Fatal(err)
				}
			}
			tree, err := groff.ParseAST(bytes.NewReader(input), groff.WithPackage("man"), groff.WithSourceName("input.1"))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := man.Build(tree, man.Options{})
			if err != nil {
				t.Fatal(err)
			}
			before, err := doc.Canonical()
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			res, renderErr := renderhtml.Render(&output, doc, o)
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && renderErr != nil {
				t.Fatal(renderErr)
			}
			if wantErr != "" && (renderErr == nil || !strings.Contains(renderErr.Error(), wantErr)) {
				t.Fatalf("want error %q, got %v", wantErr, renderErr)
			}
			if doc.Source() != string(input) {
				t.Fatal("rendering changed raw source")
			}
			after, err := doc.Canonical()
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("rendering changed semantics")
			}
			var again bytes.Buffer
			againRes, againErr := renderhtml.Render(&again, doc, o)
			if !bytes.Equal(again.Bytes(), output.Bytes()) || (renderErr == nil) != (againErr == nil) || len(res.Diagnostics) != len(againRes.Diagnostics) {
				t.Fatal("HTML export nondeterministic")
			}
			if output.Len() > 0 {
				verifyHTML(t, output.String())
			}
			actual := normalizeHTMLFonts(t, output.Bytes(), o.Assets)
			var diagnostics bytes.Buffer
			for _, d := range res.Diagnostics {
				fmt.Fprintf(&diagnostics, "%s:%d:%d: %s: %s\n", d.Position.Filename, d.Position.Line, d.Position.Column, d.Severity, d.Message)
			}
			outputs := []struct {
				name string
				data []byte
			}{{"expected.html", actual}, {"expected-diagnostics.txt", diagnostics.Bytes()}}
			if *updateHTML {
				for _, v := range outputs {
					found := false
					for i := range ar.Files {
						if ar.Files[i].Name == v.name {
							ar.Files[i].Data = v.data
							found = true
							break
						}
					}
					if !found {
						ar.Files = append(ar.Files, txtar.File{Name: v.name, Data: v.data})
					}
				}
				if err := os.WriteFile(path, txtar.Format(ar), 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				for _, v := range outputs {
					want, ok := files[v.name]
					if !ok {
						t.Errorf("missing %s (refresh explicitly with -update-html)", v.name)
						continue
					}
					if !bytes.Equal(v.data, want) {
						t.Errorf("%s differs\ngot:\n%s\nwant:\n%s", v.name, v.data, want)
					}
				}
			}
			if strings.HasSuffix(path, "complete-page.txtar") {
				source, err := htmlCases.ReadFile("examples/man-tool.1")
				if err != nil || !bytes.Equal(input, source) {
					t.Fatal("example/scenario source mismatch")
				}
				var stdout, stderr bytes.Buffer
				code := cli.Run([]string{"render", "--format", "html", "examples/man-tool.1"}, cli.Environment{Out: &stdout, Err: &stderr, Open: func(path string) (io.ReadCloser, error) {
					if path != "examples/man-tool.1" {
						t.Fatal("unexpected file request")
					}
					return io.NopCloser(bytes.NewReader(source)), nil
				}})
				if code != 0 || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), output.Bytes()) {
					t.Fatal("CLI example differs from API scenario")
				}
				if *updateHTML {
					if err := os.WriteFile("examples/man-tool.html", stdout.Bytes(), 0644); err != nil {
						t.Fatal(err)
					}
				} else {
					expected, err := htmlCases.ReadFile("examples/man-tool.html")
					if err != nil || !bytes.Equal(expected, stdout.Bytes()) {
						t.Fatal("CLI-generated example is stale")
					}
				}
			}
		})
	}
}

// Keep TXTAR readable while asserting actual font payloads independently. Only
// embedded TTF bytes are replaced, with exact face name, SHA256 and byte length.
func normalizeHTMLFonts(t *testing.T, data []byte, policy renderhtml.AssetPolicy) []byte {
	t.Helper()
	fonts := fontset.Default()
	faces := [][]byte{fonts.Regular, fonts.Bold, fonts.Italic, fonts.Mono}
	names := []string{"regular", "bold", "italic", "mono"}
	index := 0
	out := fontDataPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		parts := fontDataPattern.FindSubmatch(match)
		decoded, err := base64.StdEncoding.DecodeString(string(parts[1]))
		if err != nil || index >= 4 || !bytes.Equal(decoded, faces[index]) {
			t.Fatal("embedded default face bytes differ")
		}
		value := []byte(fmt.Sprintf("data:font/ttf;base64,{{font:%s sha256:%x bytes:%d}}", names[index], sha256.Sum256(decoded), len(decoded)))
		index++
		return value
	})
	if len(data) > 0 && policy != renderhtml.OmitFonts && index != 4 {
		t.Fatal("default fonts were not embedded")
	}
	if policy == renderhtml.OmitFonts && index != 0 {
		t.Fatal("omit-fonts policy ignored")
	}
	return out
}
func verifyHTML(t *testing.T, output string) {
	t.Helper()
	root, err := xhtml.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	var links []string
	stack := []*xhtml.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "script", "iframe", "object", "embed", "img", "link", "base", "form":
				t.Fatalf("unexpected active/resource tag %s", n.Data)
			}
			if n.Data == "dt" || n.Data == "dd" {
				if n.Parent == nil || n.Parent.Data != "dl" {
					t.Fatal("definition structure repaired by HTML parser")
				}
			}
			for _, a := range n.Attr {
				if strings.HasPrefix(strings.ToLower(a.Key), "on") {
					t.Fatal("event attribute emitted")
				}
				if a.Key == "id" {
					if ids[a.Val] {
						t.Fatal("duplicate anchor ID")
					}
					ids[a.Val] = true
				}
				if a.Key == "href" {
					if !renderhtml.SafeURL(a.Val) {
						t.Fatal("unsafe href emitted")
					}
					if strings.HasPrefix(a.Val, "#") {
						links = append(links, a.Val[1:])
					}
				}
			}
		}
		for c := n.LastChild; c != nil; c = c.PrevSibling {
			stack = append(stack, c)
		}
	}
	for _, target := range links {
		if !ids[target] {
			t.Fatalf("dangling anchor %s", target)
		}
	}
}
