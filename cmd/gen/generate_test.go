package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/generation/*.txtar
var generationCases embed.FS

func TestGenerationScenarios(t *testing.T) {
	paths, err := fs.Glob(generationCases, "testdata/generation/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := generationCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for _, f := range txtar.Parse(data).Files {
				if _, duplicate := files[f.Name]; duplicate {
					t.Fatalf("duplicate fixture %s", f.Name)
				}
				switch f.Name {
				case "catalogue.json", "error.txt", "macros.go.template", "parseline.go.template", "catalogue.go.template":
				default:
					t.Fatalf("unknown fixture %s", f.Name)
				}
				files[f.Name] = f.Data
			}
			input, ok := files["catalogue.json"]
			if !ok {
				t.Fatal("missing catalogue.json")
			}
			templates := fstest.MapFS{}
			for _, name := range []string{"macros.go.template", "parseline.go.template", "catalogue.go.template"} {
				data, err := goTemplates.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				if replacement, ok := files[name]; ok {
					data = replacement
				}
				templates[name] = &fstest.MapFile{Data: data}
			}
			dir := t.TempDir()
			paths := testOutputPaths(dir)
			for _, path := range []string{paths.macros, paths.lines, paths.catalogue, paths.support} {
				if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			outputs, err := renderOutputs(input, templates, paths)
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), wantErr) {
					t.Fatalf("want error containing %q, got %v", wantErr, err)
				}
				for _, path := range []string{paths.macros, paths.lines, paths.catalogue, paths.support} {
					assertGenerationFile(t, path, []byte("original"))
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 4 {
					t.Fatalf("render failure staged or wrote files: %v %v", entries, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			again, err := renderOutputs(input, templates, paths)
			if err != nil || len(outputs) != 4 || len(again) != 4 {
				t.Fatalf("incomplete generation: %d %d %v", len(outputs), len(again), err)
			}
			for i, output := range outputs {
				if !bytes.Equal(output.Data, again[i].Data) {
					t.Fatalf("nondeterministic output %s", output.Path)
				}
				if strings.HasSuffix(output.Path, ".go") {
					file, err := parser.ParseFile(token.NewFileSet(), output.Path, output.Data, parser.ParseComments|parser.AllErrors)
					if err != nil || !ast.IsGenerated(file) {
						t.Fatalf("invalid Go or unrecognized generated header: %s %v", output.Path, err)
					}
				}
			}
			if err := atomicfile.WriteGenerated(outputs); err != nil {
				t.Fatal(err)
			}
			if err := checkOutputs(outputs, os.ReadFile); err != nil {
				t.Fatal(err)
			}
			// A handwritten implementation outside the target set survives refresh.
			custom := filepath.Join(dir, "custom.go")
			if err := os.WriteFile(custom, []byte("handwritten implementation"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := atomicfile.WriteGenerated(again); err != nil {
				t.Fatal(err)
			}
			assertGenerationFile(t, custom, []byte("handwritten implementation"))
		})
	}
}

func TestGeneratorCheckAndLateDestinationFailure(t *testing.T) {
	dir := t.TempDir()
	paths := testOutputPaths(dir)
	args := []string{"-macrosloc", paths.macros, "-parselineloc", paths.lines, "-catalogueloc", paths.catalogue, "-supportloc", paths.support}
	if err := run(append(append([]string{}, args...), "-check"), io.Discard); err == nil {
		t.Fatal("check accepted missing output")
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatal("check created output or staging", entries, err)
	}
	if err := run(args, io.Discard); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1600000000, 0)
	for _, path := range []string{paths.macros, paths.lines, paths.catalogue, paths.support} {
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(append(append([]string{}, args...), "-check"), io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.macros, paths.lines, paths.catalogue, paths.support} {
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(stamp) {
			t.Fatalf("check changed mtime: %s %v", path, err)
		}
	}
	if err := os.WriteFile(paths.macros, []byte("stale first output"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(append(append([]string{}, args...), "-check"), io.Discard); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale check: %v", err)
	}
	assertGenerationFile(t, paths.macros, []byte("stale first output"))
	if err := os.Remove(paths.catalogue); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(paths.catalogue, 0700); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(paths.lines)
	if err != nil {
		t.Fatal(err)
	}
	if err := run(args, io.Discard); err == nil {
		t.Fatal("directory destination accepted")
	}
	assertGenerationFile(t, paths.macros, []byte("stale first output"))
	assertGenerationFile(t, paths.lines, before)
	if paths, err := filepath.Glob(filepath.Join(dir, ".gtroff-gen-*")); err != nil || len(paths) != 0 {
		t.Fatal("failed command left staging", paths, err)
	}
}

func TestGeneratedSpecialIdentifierEscaping(t *testing.T) {
	for _, name := range []string{".café", ".[", ".tick`\"\\path"} {
		t.Run(name, func(t *testing.T) {
			entry := RawContent{Type: "Macro", Token: name}
			if name == ".[" {
				entry.Block = &BlockPolicy{Kind: "reference", Ends: []string{".]"}}
			}
			raw, err := json.Marshal([]RawContainer{{Contents: []RawContent{entry}}})
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := renderOutputs(raw, goTemplates, testOutputPaths(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), "parseline.go", outputs[1].Data, parser.AllErrors)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			ast.Inspect(file, func(node ast.Node) bool {
				if clause, ok := node.(*ast.CaseClause); ok {
					for _, value := range clause.List {
						literal, ok := value.(*ast.BasicLit)
						if !ok {
							t.Fatalf("unexpected generated command expression: %T", value)
						}
						decoded, err := strconv.Unquote(literal.Value)
						if err != nil || decoded != name {
							t.Fatalf("command spelling changed: %q %v", decoded, err)
						}
						found = true
					}
				}
				return true
			})
			if !found {
				t.Fatal("new command did not reach generated dispatch")
			}
			for _, output := range outputs[:3] {
				if _, err := parser.ParseFile(token.NewFileSet(), output.Path, output.Data, parser.AllErrors); err != nil {
					t.Fatalf("special identifier generates invalid Go in %s: %v", output.Path, err)
				}
			}
		})
	}
}

func testOutputPaths(dir string) outputPaths {
	return outputPaths{filepath.Join(dir, "macros.go"), filepath.Join(dir, "parseline.go"), filepath.Join(dir, "catalogue.go"), filepath.Join(dir, "support.json")}
}

func assertGenerationFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("generated file %s changed unexpectedly: %v", path, err)
	}
}
