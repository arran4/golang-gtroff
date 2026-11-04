package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/man/*.txtar examples/man-tool.1 examples/man-tool.semantic.json
var manCases embed.FS
var updateMan = flag.Bool("update-man", false, "explicitly update semantic man scenario/example goldens")

func TestManScenarios(t *testing.T) {
	var paths []string
	if err := fs.WalkDir(manCases, "testdata/man", func(p string, d fs.DirEntry, err error) error {
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
		t.Run(strings.TrimPrefix(path, "testdata/man/"), func(t *testing.T) {
			data, err := manCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ar := txtar.Parse(data)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate scenario file")
				}
				files[f.Name] = f.Data
			}
			input, ok := files["input.1"]
			if !ok {
				t.Fatal("missing input.1")
			}
			var options struct {
				Strict    bool   `json:"strict"`
				Section   string `json:"section"`
				Normalize bool   `json:"normalize"`
				ASTBlock  string `json:"astBlock"`
				ASTLine   string `json:"astLine"`
				Raw       bool   `json:"raw"`
			}
			if data, ok := files["options.json"]; ok {
				if err := json.Unmarshal(data, &options); err != nil {
					t.Fatal(err)
				}
			}
			var tree *groff.AST
			parse := func() (*man.Document, error) {
				var err error
				tree, err = groff.ParseAST(bytes.NewReader(input), groff.WithPackage("man"), groff.WithSourceName("input.1"))
				if err != nil {
					t.Fatal(err)
				}
				return man.Build(tree, man.Options{Strict: options.Strict})
			}
			doc, buildErr := parse()
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && buildErr != nil {
				t.Fatal(buildErr)
			}
			if wantErr != "" && (buildErr == nil || !strings.Contains(buildErr.Error(), wantErr)) {
				t.Fatalf("expected %q, got %v", wantErr, buildErr)
			}
			if err := doc.Validate(); err != nil {
				t.Fatal(err)
			}
			if doc.Source() != string(input) || doc.Raw(1) != string(input) {
				t.Fatal("raw source not independent of semantics")
			}
			gotJSON := manGolden(t, doc)
			again, _ := parse()
			if !bytes.Equal(gotJSON, manGolden(t, again)) {
				t.Fatal("semantic JSON nondeterministic")
			}
			var text string
			selectors := 0
			for _, selector := range []string{options.Section, options.ASTBlock, options.ASTLine} {
				if selector != "" {
					selectors++
				}
			}
			if selectors > 1 {
				t.Fatal("scenario extraction selectors are mutually exclusive")
			}
			textOptions := man.TextOptions{Raw: options.Raw, Normalize: options.Normalize}
			if options.ASTBlock != "" || options.ASTLine != "" {
				hits := tree.Root().Find(func(n *groff.ASTNode) bool {
					return options.ASTBlock != "" && n.Kind() == groff.ASTBlock && n.Attributes().Name == options.ASTBlock ||
						options.ASTLine != "" && n.Kind() == groff.ASTLine && strings.TrimSuffix(strings.TrimSuffix(n.Raw(), "\n"), "\r") == options.ASTLine
				})
				if len(hits) != 1 {
					t.Fatalf("AST extraction selector matched %d nodes", len(hits))
				}
				selection, err := doc.SelectAST(hits[0].ID())
				if err != nil {
					t.Fatal(err)
				}
				text = selection.Text(textOptions)
			} else if options.Section != "" {
				selection, err := doc.SelectSection(options.Section, false)
				if err != nil {
					t.Fatal(err)
				}
				text = selection.Text(textOptions)
			} else {
				text = doc.Text(1, textOptions)
			}
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			canonical, err := doc.Canonical()
			if err != nil {
				t.Fatal(err)
			}
			canonical = append(canonical, '\n')
			// Actual JSON model round-trip retains canonical semantics, not source bytes.
			raw, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			var roundTrip man.Document
			if err := json.Unmarshal(raw, &roundTrip); err != nil {
				t.Fatal(err)
			}
			canonicalAgain, err := roundTrip.Canonical()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bytes.TrimSuffix(canonical, []byte("\n")), canonicalAgain) {
				t.Fatal("canonical semantic JSON round-trip changed content")
			}
			for _, n := range doc.Nodes {
				for _, id := range n.Children {
					child := doc.Node(id)
					if child == nil || child.Parent != n.ID || child.Origin.Span.Start.Offset < n.Origin.Span.Start.Offset || child.Origin.Span.End.Offset > n.Origin.Span.End.Offset {
						t.Fatal("semantic ancestry/source range disagreement")
					}
				}
			}
			outputs := []struct {
				name string
				data []byte
			}{{"expected-man.json", gotJSON}, {"expected-text.txt", []byte(text)}, {"expected-canonical.json", canonical}}
			if *updateMan {
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
						t.Errorf("missing %s; update explicitly with -update-man", v.name)
						continue
					}
					if !bytes.Equal(v.data, want) {
						t.Errorf("%s mismatch\ngot:\n%s\nwant:\n%s", v.name, v.data, want)
					}
				}
			}
			if strings.HasSuffix(path, "synthetic-page.txtar") {
				source, err := manCases.ReadFile("examples/man-tool.1")
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(input, source) {
					t.Fatal("representative example source differs from scenario")
				}
				if *updateMan {
					if err := os.WriteFile("examples/man-tool.semantic.json", gotJSON, 0644); err != nil {
						t.Fatal(err)
					}
				} else {
					expected, err := manCases.ReadFile("examples/man-tool.semantic.json")
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(expected, gotJSON) {
						t.Fatal("example semantic JSON is stale")
					}
				}
			}
		})
	}
}
func manGolden(t *testing.T, d *man.Document) []byte {
	t.Helper()
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		Version        int               `json:"version"`
		Metadata       json.RawMessage   `json:"metadata"`
		MetadataOrigin json.RawMessage   `json:"metadataOrigin"`
		Nodes          []json.RawMessage `json:"nodes"`
		Diagnostics    json.RawMessage   `json:"diagnostics"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	b.WriteString("{\n  \"version\": 1,\n  \"metadata\": ")
	b.Write(value.Metadata)
	if len(value.MetadataOrigin) > 0 {
		b.WriteString(",\n  \"metadataOrigin\": ")
		b.Write(value.MetadataOrigin)
	}
	b.WriteString(",\n  \"nodes\": [\n")
	for i, row := range value.Nodes {
		b.WriteString("    ")
		b.Write(row)
		if i+1 < len(value.Nodes) {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ]")
	if len(value.Diagnostics) > 0 {
		b.WriteString(",\n  \"diagnostics\": ")
		b.Write(value.Diagnostics)
	}
	b.WriteString("\n}\n")
	return b.Bytes()
}
