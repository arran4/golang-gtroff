package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/image/*.txtar
var imageCases embed.FS
var updateImages = flag.Bool("update-images", false, "explicitly refresh textual image scenarios and separate binary goldens")

func TestImageScenarios(t *testing.T) {
	paths, e := fs.Glob(imageCases, "testdata/image/*.txtar")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/image/"), func(t *testing.T) {
			raw, e := imageCases.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate archive name")
				}
				files[f.Name] = f.Data
			}
			input, ok := files["input.1"]
			if !ok {
				t.Fatal("missing source")
			}
			var o renderimage.Options
			if e := json.Unmarshal(files["options.json"], &o); e != nil {
				t.Fatal(e)
			}
			tree, e := groff.ParseAST(bytes.NewReader(input), groff.WithPackage("man"), groff.WithSourceName("input.1"))
			if e != nil {
				t.Fatal(e)
			}
			doc, e := man.Build(tree, man.Options{})
			if e != nil {
				t.Fatal(e)
			}
			before, _ := doc.Canonical()
			plan, err := renderimage.Layout(doc, o)
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if wantErr != "" && (err == nil || !strings.Contains(err.Error(), wantErr)) {
				t.Fatalf("want %q got %v", wantErr, err)
			}
			var pages []renderimage.Page
			var diagnostics []groff.Diagnostic
			var names []string
			if plan != nil {
				pages = plan.Pages()
				diagnostics = plan.Result().Diagnostics
			}
			if diagnostics == nil {
				diagnostics = []groff.Diagnostic{}
			}
			if err == nil {
				assertImageContent(t, doc, o, pages)
				format := o.Format
				if format == "" {
					format = "png"
				}
				if format == "jpeg" {
					format = "jpg"
				}
				names, e = renderimage.OutputPaths("page."+format, format, len(pages))
				if e != nil {
					t.Fatal(e)
				}
				_, e = plan.Encode(func(pg renderimage.Page, data []byte) error {
					var decoded image.Image
					var decodeErr error
					if format == "png" {
						decoded, decodeErr = png.Decode(bytes.NewReader(data))
					} else {
						decoded, decodeErr = jpeg.Decode(bytes.NewReader(data))
					}
					if decodeErr != nil {
						t.Fatal(decodeErr)
					}
					if decoded.Bounds() != image.Rect(0, 0, pg.Width, pg.Height) {
						t.Fatal("encoded dimensions changed")
					}
					actual, e := plan.Image(pg.Number - 1)
					if e != nil {
						t.Fatal(e)
					}
					if format == "png" {
						if !bytes.Equal(rgbaPixels(actual), rgbaPixels(decoded)) {
							t.Fatal("PNG pixels differ from layout painting")
						}
					}
					golden := filepath.Join("testdata", "golden", "image", strings.TrimSuffix(filepath.Base(path), ".txtar")+fmt.Sprintf("-%03d.%s", pg.Number, format))
					if *updateImages {
						if e := os.WriteFile(golden, data, 0644); e != nil {
							t.Fatal(e)
						}
					} else {
						want, e := os.ReadFile(golden)
						if e != nil {
							t.Fatal(e)
						}
						if !bytes.Equal(want, data) {
							t.Errorf("binary golden differs: %s; approve refresh explicitly with -update-images", golden)
						}
					}
					// At least one non-background pixel on every content-bearing page.
					if len(pg.Lines) > 0 {
						pixels := rgbaPixels(actual)
						bg := pixels[:4]
						visible := false
						for i := 0; i < len(pixels); i += 4 {
							if !bytes.Equal(bg, pixels[i:i+4]) {
								visible = true
								break
							}
						}
						if !visible {
							t.Fatal("content page is blank")
						}
					}
					return nil
				})
				if e != nil {
					t.Fatal(e)
				}
			} else if plan != nil {
				if _, e := plan.Image(0); e == nil {
					t.Fatal("strict plan permitted painting")
				}
			}
			after, _ := doc.Canonical()
			if !bytes.Equal(before, after) || doc.Source() != string(input) {
				t.Fatal("rendering mutated semantic model or raw source")
			}
			pageJSON, _ := json.MarshalIndent(pages, "", "  ")
			diagJSON, _ := json.MarshalIndent(diagnostics, "", "  ")
			nameJSON, _ := json.MarshalIndent(names, "", "  ")
			actual := map[string][]byte{"expected-layout.json": append(pageJSON, '\n'), "expected-diagnostics.json": append(diagJSON, '\n'), "expected-paths.json": append(nameJSON, '\n')}
			if *updateImages {
				next := &txtar.Archive{Comment: ar.Comment}
				for _, f := range ar.Files {
					if _, ok := actual[f.Name]; !ok {
						next.Files = append(next.Files, f)
					}
				}
				for _, name := range []string{"expected-layout.json", "expected-diagnostics.json", "expected-paths.json"} {
					next.Files = append(next.Files, txtar.File{Name: name, Data: actual[name]})
				}
				if e := os.WriteFile(path, txtar.Format(next), 0644); e != nil {
					t.Fatal(e)
				}
			} else {
				for name, got := range actual {
					if !bytes.Equal(files[name], got) {
						t.Errorf("%s differs; refresh explicitly with -update-images\n%s", name, got)
					}
				}
			}
		})
	}
}
func rgbaPixels(img image.Image) []byte {
	r := img.Bounds()
	out := make([]byte, 0, r.Dx()*r.Dy()*4)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			rr, g, b, a := img.At(x, y).RGBA()
			out = append(out, byte(rr>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return out
}
func assertImageContent(t *testing.T, doc *man.Document, o renderimage.Options, pages []renderimage.Page) {
	t.Helper()
	var selection *man.Selection
	var err error
	switch {
	case o.Section != "":
		selection, err = doc.SelectSection(o.Section, false)
	case o.Node != 0:
		selection, err = doc.SelectNode(o.Node)
	default:
		selection, err = doc.SelectNode(1)
	}
	if err != nil {
		t.Fatal(err)
	}
	actual := map[man.ID]*strings.Builder{}
	for _, pg := range pages {
		for _, l := range pg.Lines {
			if actual[l.Node] == nil {
				actual[l.Node] = &strings.Builder{}
			}
			for _, f := range l.Fragments {
				actual[l.Node].WriteString(f.Source)
				if !f.Bounds.Empty() && !f.Bounds.In(image.Rect(0, 0, pg.Width, pg.Height)) {
					t.Fatal("glyph outside page")
				}
			}
			if l.HardBreak {
				actual[l.Node].WriteByte('\n')
			}
		}
	}
	strip := func(s string) string {
		return strings.Map(func(r rune) rune {
			if strings.ContainsRune(" \t\r\n\f", r) {
				return -1
			}
			return r
		}, s)
	}
	for _, n := range selection.Nodes {
		var want strings.Builder
		rs := n.Runs
		if n.Kind == man.Item {
			rs = n.Tag
		}
		if n.Kind == man.DocumentNode {
			continue
		}
		for _, r := range rs {
			want.WriteString(r.Text)
		}
		if len(rs) == 0 {
			continue
		}
		got := ""
		if actual[n.ID] != nil {
			got = actual[n.ID].String()
		}
		if strip(got) != strip(want.String()) {
			t.Errorf("node %d lost/duplicated visible source: got %q want %q", n.ID, got, want.String())
		}
	}
}

//go:embed examples/man-tool.png examples/man-tool.jpg
var imageExamples embed.FS

func TestImageCLIExamples(t *testing.T) {
	source, e := htmlCases.ReadFile("examples/man-tool.1")
	if e != nil {
		t.Fatal(e)
	}
	for _, format := range []string{"png", "jpg"} {
		t.Run(format, func(t *testing.T) {
			args := []string{"render", "--format", format}
			if format == "jpg" {
				args = append(args, "--theme", "dark")
			}
			var out, diags bytes.Buffer
			code := cli.Run(args, cli.Environment{In: bytes.NewReader(source), Out: &out, Err: &diags})
			if code != 0 || diags.Len() != 0 {
				t.Fatalf("example %d %s", code, diags.String())
			}
			want, e := imageExamples.ReadFile("examples/man-tool." + format)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(out.Bytes(), want) {
				t.Fatal("checked-in image differs from CLI output")
			}
		})
	}
}
