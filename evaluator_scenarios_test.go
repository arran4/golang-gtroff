package groff

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/evaluation/*.txtar
var evaluationCases embed.FS
var updateEvaluation = flag.Bool("update-evaluation", false, "explicitly refresh opt-in evaluation expectations")

func TestEvaluationScenarios(t *testing.T) {
	paths, err := fs.Glob(evaluationCases, "testdata/evaluation/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range paths {
		t.Run(fixture, func(t *testing.T) {
			data, err := evaluationCases.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			files := map[string][]byte{}
			for _, file := range archive.Files {
				if _, ok := files[file.Name]; ok {
					t.Fatalf("duplicate %s", file.Name)
				}
				switch file.Name {
				case "input.roff", "options.json", "error.txt", "expected.txt", "parsed.txt":
				default:
					t.Fatalf("unknown file %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, ok := files["input.roff"]
			if !ok {
				t.Fatal("missing input.roff")
			}
			doc, err := ParseDocument(strings.NewReader(string(input)), WithRegistry(NewRegistry()))
			if err != nil {
				t.Fatalf("fixture parsing: %v", err)
			}
			tokens := make([]Token, len(doc.Tokens))
			before := make([]string, len(tokens))
			var parsed strings.Builder
			for i, dt := range doc.Tokens {
				tokens[i] = dt.Token
				before[i] = dt.Token.String()
				fmt.Fprintf(&parsed, "%T %q\n", dt.Token, before[i])
			}
			env := NewEnvironment()
			if options := files["options.json"]; options != nil {
				if err := json.Unmarshal(options, env); err != nil {
					t.Fatal(err)
				}
			}
			output, evalErr := EvaluateContext(context.Background(), tokens, env)
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && evalErr != nil || wantErr != "" && (evalErr == nil || !strings.Contains(evalErr.Error(), wantErr)) {
				t.Fatalf("want error %q; got %v", wantErr, evalErr)
			}
			if doc.Source() != string(input) {
				t.Fatal("evaluation changed document source")
			}
			for i, tok := range tokens {
				if tok.String() != before[i] {
					t.Fatalf("evaluation mutated token %d", i)
				}
			}
			var got strings.Builder
			for _, tok := range output {
				fmt.Fprintf(&got, "%T %q\n", tok, tok.String())
			}
			keys := make([]string, 0, len(env.Registers))
			for key := range env.Registers {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(&got, "register %s=%d\n", key, env.Registers[key])
			}
			keys = nil
			for key := range env.Strings {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(&got, "string %s=%q\n", key, env.Strings[key])
			}
			for _, d := range env.Diagnostics {
				fmt.Fprintf(&got, "diagnostic token=%d path=%q %s\n", d.TokenIndex, d.Expansion, d.Message)
			}
			if evalErr != nil {
				fmt.Fprintf(&got, "error %v\n", evalErr)
			}
			for _, part := range []struct{ name, text string }{{"parsed.txt", parsed.String()}, {"expected.txt", got.String()}} {
				if *updateEvaluation {
					found := false
					for i, file := range archive.Files {
						if file.Name == part.name {
							archive.Files[i].Data = []byte(part.text)
							found = true
						}
					}
					if !found {
						archive.Files = append(archive.Files, txtar.File{Name: part.name, Data: []byte(part.text)})
					}
				} else if part.text != string(files[part.name]) {
					t.Fatalf("%s got:\n%s\nwant:\n%s", part.name, part.text, files[part.name])
				}
			}
			if *updateEvaluation {
				if err := os.WriteFile(fixture, txtar.Format(archive), 0644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
