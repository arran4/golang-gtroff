package groff_test

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/hierarchy/*.txtar
var hierarchyCases embed.FS

func TestHierarchyScenarios(t *testing.T) {
	paths, err := fs.Glob(hierarchyCases, "testdata/hierarchy/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := hierarchyCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]string{}
			for _, file := range txtar.Parse(data).Files {
				files[file.Name] = string(file.Data)
			}
			var options struct{ StrictError bool }
			if err := json.Unmarshal([]byte(files["options.json"]), &options); err != nil {
				t.Fatal(err)
			}
			source := files["input.1"]
			tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"), groff.WithSourceName("input.1"))
			if err != nil {
				t.Fatal(err)
			}
			document, err := man.Build(tree, man.Options{})
			if err != nil {
				t.Fatal(err)
			}
			strict, err := man.Build(tree, man.Options{Strict: true})
			if (err != nil) != options.StrictError {
				t.Fatalf("strict error = %v, want failure %v", err, options.StrictError)
			}
			if strict.Syntax() != tree || document.Syntax() != tree {
				t.Fatal("projection lost syntax owner")
			}
			if err := document.Validate(); err != nil {
				t.Fatal(err)
			}
			var outline strings.Builder
			for _, node := range document.Nodes {
				depth := 0
				for parent := node.Parent; parent != 0; parent = document.Node(parent).Parent {
					depth++
				}
				outline.WriteString(strings.Repeat("  ", depth) + string(node.Kind))
				if node.Heading != "" {
					outline.WriteString(" " + node.Heading)
				}
				for _, run := range node.Tag {
					outline.WriteString(" " + run.Text)
				}
				outline.WriteByte('\n')
				lines, err := document.SourceNodes(node.ID)
				if err != nil {
					t.Fatal(err)
				}
				var raw strings.Builder
				for _, line := range lines {
					if line.Kind() != groff.ASTLine || tree.Node(line.ID()) != line {
						t.Fatal("source identity changed")
					}
					if line.Raw() != source[line.Span().Start.Offset:line.Span().End.Offset] {
						t.Fatal("source span mismatch")
					}
					raw.WriteString(line.Raw())
				}
				if raw.String() != document.Raw(node.ID) {
					t.Fatalf("%s source lines do not cover its raw span", node.Kind)
				}
				if node.ID == 1 && raw.String() != source {
					t.Fatal("projection discarded source")
				}
			}
			if outline.String() != files["expected-outline.txt"] {
				t.Fatalf("outline:\n%s\nwant:\n%s", outline.String(), files["expected-outline.txt"])
			}
		})
	}
}
