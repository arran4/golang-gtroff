package groff

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

	"golang.org/x/tools/txtar"
)

//go:embed testdata/ast/*.txtar
var astCases embed.FS
var updateAST = flag.Bool("update-ast", false, "explicitly update AST scenario goldens")

type astScenarioOptions struct {
	Package       string `json:"package"`
	Strict        bool   `json:"strict"`
	Unterminated  bool   `json:"unterminated"`
	SelectSection string `json:"selectSection"`
}

func TestASTScenarios(t *testing.T) {
	var paths []string
	if err := fs.WalkDir(astCases, "testdata/ast", func(p string, d fs.DirEntry, err error) error {
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
		t.Run(strings.TrimPrefix(path, "testdata/ast/"), func(t *testing.T) {
			data, err := astCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ar := txtar.Parse(data)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatalf("duplicate file %s", f.Name)
				}
				files[f.Name] = f.Data
			}
			input, ok := files["input.1"]
			if !ok {
				t.Fatal("missing input.1")
			}
			o := astScenarioOptions{Package: "man"}
			if data, ok := files["options.json"]; ok {
				if err := json.Unmarshal(data, &o); err != nil {
					t.Fatal(err)
				}
			}
			if o.Unterminated {
				input = bytes.TrimSuffix(input, []byte("\n"))
			}
			parse := func() (*AST, error) {
				return ParseAST(bytes.NewReader(input), WithPackage(o.Package), WithStrict(o.Strict), WithSourceName("input.1"))
			}
			a, parseErr := parse()
			if a == nil {
				t.Fatal("no preserved AST")
			}
			expectedError := strings.TrimSpace(string(files["error.txt"]))
			if expectedError == "" && parseErr != nil {
				t.Fatal(parseErr)
			}
			if expectedError != "" && (parseErr == nil || !strings.Contains(parseErr.Error(), expectedError)) {
				t.Fatalf("want error containing %q, got %v", expectedError, parseErr)
			}
			verifyASTIntegrity(t, a, string(input))
			gotJSON, err := astGoldenJSON(a)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON = append(gotJSON, '\n')
			again, _ := parse()
			againJSON, err := astGoldenJSON(again)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bytes.TrimSuffix(gotJSON, []byte("\n")), againJSON) {
				t.Fatal("AST serialization is nondeterministic")
			}
			selected := a.Root()
			if o.SelectSection != "" {
				hits := selected.FindSection(o.SelectSection, false)
				if len(hits) == 0 {
					t.Fatalf("no section %s", o.SelectSection)
				}
				selected = hits[0]
			}
			extract := selected.Text()
			// txtar represents text files with a final newline. Exact extraction
			// boundaries are asserted separately by TestASTTextBoundaries.
			if extract != "" && !strings.HasSuffix(extract, "\n") {
				extract += "\n"
			}
			outputs := []struct {
				name string
				data []byte
			}{
				{"expected-ast.json", gotJSON}, {"expected-extract.txt", []byte(extract)},
			}
			if *updateAST {
				for _, output := range outputs {
					name, data := output.name, output.data
					found := false
					for i := range ar.Files {
						if ar.Files[i].Name == name {
							ar.Files[i].Data = data
							found = true
							break
						}
					}
					if !found {
						ar.Files = append(ar.Files, txtar.File{Name: name, Data: data})
					}
				}
				if err := os.WriteFile(path, txtar.Format(ar), 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				for _, output := range outputs {
					name, got := output.name, output.data
					want, ok := files[name]
					if !ok {
						t.Errorf("missing %s (use -update-ast explicitly)", name)
						continue
					}
					if !bytes.Equal(got, want) {
						t.Errorf("%s mismatch\ngot:\n%s\nwant:\n%s", name, got, want)
					}
				}
			}
		})
	}
}

func verifyASTIntegrity(t *testing.T, a *AST, source string) {
	t.Helper()
	if a.Source() != source || a.Root().Raw() != source {
		t.Fatal("source preservation failed")
	}
	var lines strings.Builder
	lastLineEnd := 0
	visited := 0
	a.Root().Walk(func(n *ASTNode) WalkAction {
		visited++
		if n.ID() != NodeID(visited) || a.Node(n.ID()) != n {
			t.Fatal("identity is not indexed preorder")
		}
		s := n.Span()
		if s.Start.Offset < 0 || s.End.Offset < s.Start.Offset || s.End.Offset > len(source) {
			t.Fatalf("invalid span %+v", s)
		}
		wantStart := advanceSourcePosition(Position{Filename: "input.1", Line: 1, Column: 1}, source[:s.Start.Offset])
		wantEnd := advanceSourcePosition(Position{Filename: "input.1", Line: 1, Column: 1}, source[:s.End.Offset])
		if s.Start != wantStart || s.End != wantEnd {
			t.Fatalf("line/column disagrees with bytes: %+v want %+v .. %+v", s, wantStart, wantEnd)
		}
		children := n.Children()
		for i, c := range children {
			if c.Parent() != n || c.sibling != i || c.Span().Start.Offset < s.Start.Offset || c.Span().End.Offset > s.End.Offset {
				t.Fatalf("broken child relation at %s", c.Path())
			}
			if i > 0 && c.PreviousSibling() != children[i-1] {
				t.Fatal("previous sibling")
			}
			if i+1 < len(children) && c.NextSibling() != children[i+1] {
				t.Fatal("next sibling")
			}
		}
		if n.Kind() == ASTLine {
			if s.Start.Offset != lastLineEnd {
				t.Fatal("missing or repeated physical line")
			}
			lines.WriteString(n.Raw())
			lastLineEnd = s.End.Offset
		}
		if n.Kind() == ASTInline {
			v, ok := n.Inline()
			if !ok || v.Raw != n.Raw() {
				t.Fatal("inline raw/source mismatch")
			}
		}
		return WalkContinue
	})
	if lines.String() != source || lastLineEnd != len(source) {
		t.Fatal("traversal dropped/reordered source")
	}
	physicalLines := strings.Count(source, "\n")
	if len(source) > 0 && !strings.HasSuffix(source, "\n") {
		physicalLines++
	}
	if got := len(a.Root().FindKind(ASTLine)); got != physicalLines {
		t.Fatalf("physical line cardinality: got %d want %d", got, physicalLines)
	}
	if len(a.Root().Descendants())+1 != visited {
		t.Fatal("descendant range")
	}
	if a.Node(0) != nil || a.Node(NodeID(visited+1)) != nil {
		t.Fatal("invalid ID resolves")
	}
}

// One JSON row per node makes individual source spans and links reviewable
// without thousands of lines of repeated indentation in a tiny scenario.
func astGoldenJSON(a *AST) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var table struct {
		Nodes       []json.RawMessage `json:"nodes"`
		Diagnostics json.RawMessage   `json:"diagnostics"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("{\n  \"nodes\": [\n")
	for i, row := range table.Nodes {
		b.WriteString("    ")
		b.Write(row)
		if i+1 < len(table.Nodes) {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ]")
	if len(table.Diagnostics) > 0 {
		b.WriteString(",\n  \"diagnostics\": ")
		b.Write(table.Diagnostics)
	}
	b.WriteString("\n}")
	return b.Bytes(), nil
}
