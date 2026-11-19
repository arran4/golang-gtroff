package source_test

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/source"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var scenarios embed.FS

func TestSourceScenarios(t *testing.T) {
	paths, err := fs.Glob(scenarios, "testdata/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := scenarios.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]string{}
			virtual := fstest.MapFS{}
			for _, file := range txtar.Parse(data).Files {
				if _, ok := files[file.Name]; ok {
					t.Fatal("duplicate fixture", file.Name)
				}
				files[file.Name] = string(file.Data)
				if strings.HasPrefix(file.Name, "input/") {
					virtual[strings.TrimPrefix(file.Name, "input/")] = &fstest.MapFile{Data: file.Data}
				}
			}
			var config struct {
				Root    string
				Options source.Options
				Error   string
			}
			if err := json.Unmarshal([]byte(files["options.json"]), &config); err != nil {
				t.Fatal(err)
			}
			var set *source.Set
			if config.Root == "" {
				set, err = source.Read(context.Background(), strings.NewReader(files["stdin.1"]), "<stdin>", config.Options)
			} else {
				set, err = source.Load(context.Background(), virtual, config.Root, config.Options)
			}
			switch config.Error {
			case "":
				if err != nil {
					t.Fatal(err)
				}
			case "strict":
				if !errors.Is(err, source.ErrStrict) {
					t.Fatal("strict error missing", err)
				}
			case "limit":
				if !errors.Is(err, source.ErrLimit) {
					t.Fatal("limit error missing", err)
				}
			default:
				t.Fatal("unknown expected error")
			}
			var out strings.Builder
			for _, file := range set.Files() {
				fmt.Fprintf(&out, "file %d %s depth=%d parent=%d\n", file.ID(), file.Name(), file.Depth(), file.ParentEdge())
			}
			for _, edge := range set.Edges() {
				fmt.Fprintf(&out, "edge %d %d:%d %s %s %s -> %d\n", edge.ID, edge.From, edge.Span.Start.Line, edge.Request, edge.Target, edge.State, edge.To)
			}
			physical := map[source.ID]*strings.Builder{}
			for _, line := range set.Lines() {
				if physical[line.Source] == nil {
					physical[line.Source] = new(strings.Builder)
				}
				physical[line.Source].WriteString(set.Node(line).Raw())
				fmt.Fprintf(&out, "line %d:%d\n", line.Source, set.Node(line).Span().Start.Line)
			}
			for _, file := range set.Files() {
				if physical[file.ID()].String() != file.AST().Source() {
					t.Fatal("logical references lost physical bytes")
				}
			}
			if out.String() != files["expected.txt"] {
				t.Fatalf("source graph:\n%s\nwant:\n%s", out.String(), files["expected.txt"])
			}
		})
	}
}
