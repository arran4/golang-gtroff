package script_test

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/source"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/scenarios/*.txtar
var scriptScenarios embed.FS
var updateScript = flag.Bool("update-script", false, "explicitly capture script scenario outputs")

func TestScriptScenarios(t *testing.T) {
	paths, e := fs.Glob(scriptScenarios, "testdata/scenarios/*.txtar")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/scenarios/"), func(t *testing.T) {
			data, e := scriptScenarios.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			archive := txtar.Parse(data)
			files := map[string][]byte{}
			virtual := fstest.MapFS{}
			for _, file := range archive.Files {
				if _, ok := files[file.Name]; ok {
					t.Fatal("duplicate", file.Name)
				}
				files[file.Name] = file.Data
				if strings.HasPrefix(file.Name, "input/") {
					virtual[strings.TrimPrefix(file.Name, "input/")] = &fstest.MapFile{Data: file.Data}
				}
			}
			var config struct {
				AllowEdits, AllowWrites, Strict, IncludeEdits, NoIncludes bool
				MaxSteps, MaxHostCalls, MaxOutputBytes                    int
			}
			if b := files["options.json"]; len(b) > 0 {
				if e = json.Unmarshal(b, &config); e != nil {
					t.Fatal(e)
				}
			}
			options := script.Options{AllowEdits: config.AllowEdits, AllowWriteRequests: config.AllowWrites, Edit: edit.Options{Strict: config.Strict, IncludeEdits: config.IncludeEdits, Source: source.Options{NoIncludes: config.NoIncludes}}, MaxSteps: config.MaxSteps, MaxHostCalls: config.MaxHostCalls, MaxOutputBytes: config.MaxOutputBytes}
			run := func() map[string][]byte {
				var set *source.Set
				var inputError error
				if _, exists := virtual["page.1"]; exists {
					set, inputError = source.Load(context.Background(), virtual, "page.1", options.Edit.Source)
				}
				result, err := script.Run(context.Background(), script.Request{Name: "program.tengo", Code: files["program.tengo"], Sources: set, InputError: inputError}, options)
				output := map[string][]byte{"stdout.txt": {}, "error.txt": {}}
				if err != nil {
					output["error.txt"] = []byte(err.Error() + "\n")
				}
				if result != nil {
					output["stdout.txt"] = result.Output
					summary := struct {
						Calls, Steps, Constants int
						Diagnostics             []edit.Diagnostic
						SourceDiagnostics       []source.Diagnostic
						Writes                  []script.WriteRequest
					}{result.Stats.HostCalls, result.Stats.Steps, result.Stats.Constants, result.Diagnostics, result.SourceDiagnostics, result.Writes}
					encoded, e := json.MarshalIndent(summary, "", "  ")
					if e != nil {
						t.Fatal(e)
					}
					output["report.json"] = append(encoded, '\n')
					if result.Working != nil {
						for _, name := range result.Working.Names() {
							b, e := result.Working.Data(name)
							if e != nil {
								t.Fatal(e)
							}
							output["after/"+name] = b
						}
					}
				}
				return output
			}
			first, second := run(), run()
			for name, b := range first {
				if !bytes.Equal(b, second[name]) {
					t.Fatal("nondeterministic", name)
				}
			}
			if *updateScript {
				var inputFiles []txtar.File
				for _, file := range archive.Files {
					if file.Name == "program.tengo" || file.Name == "options.json" || strings.HasPrefix(file.Name, "input/") {
						inputFiles = append(inputFiles, file)
					}
				}
				archive.Files = inputFiles
				names := make([]string, 0, len(first))
				for name := range first {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					archive.Files = append(archive.Files, txtar.File{Name: name, Data: first[name]})
				}
				if e = os.WriteFile(path, txtar.Format(archive), 0644); e != nil {
					t.Fatal(e)
				}
			} else {
				for name, b := range first {
					expected, ok := files[name]
					if !ok || !bytes.Equal(b, expected) {
						t.Errorf("%s mismatch; explicitly -update-script\ngot %s\nwant %s", name, b, expected)
					}
				}
				for name := range files {
					if strings.HasPrefix(name, "after/") {
						if _, ok := first[name]; !ok {
							t.Error("unexpected previous source", name)
						}
					}
				}
			}
		})
	}
}
