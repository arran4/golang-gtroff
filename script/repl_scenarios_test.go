package script_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/manlib"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/repl/*.txtar
var replScenarios embed.FS
var updateREPL = flag.Bool("update-repl", false, "explicitly capture native REPL transcripts")

func TestREPLScenarios(t *testing.T) {
	paths, e := fs.Glob(replScenarios, "testdata/repl/*.txtar")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/repl/"), func(t *testing.T) {
			data, e := replScenarios.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			ar := txtar.Parse(data)
			files := map[string][]byte{}
			virtual := fstest.MapFS{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate", f.Name)
				}
				files[f.Name] = f.Data
				if strings.HasPrefix(f.Name, "input/") {
					virtual[strings.TrimPrefix(f.Name, "input/")] = &fstest.MapFile{Data: f.Data}
				}
			}
			var args []string
			if e = json.Unmarshal(files["args.json"], &args); e != nil {
				t.Fatal(e)
			}
			run := func() (int, []byte, []byte) {
				var out, stderr bytes.Buffer
				env := replEnvironment(string(files["session.txt"]), &out, &stderr)
				env.OpenRoot = func(string) (manlib.Root, io.Closer, error) { return manlib.Root{FS: virtual}, nil, nil }
				code := cli.Run(args, env)
				return code, out.Bytes(), stderr.Bytes()
			}
			code, out, stderr := run()
			second, other, errOut := run()
			if code != second || !bytes.Equal(out, other) || !bytes.Equal(stderr, errOut) {
				t.Fatal("nondeterministic native transcript")
			}
			encoded, _ := json.Marshal(code)
			outputs := []txtar.File{{Name: "stdout.quoted", Data: []byte(strconv.Quote(string(out)) + "\n")}, {Name: "stderr.quoted", Data: []byte(strconv.Quote(string(stderr)) + "\n")}, {Name: "exit.txt", Data: append(encoded, '\n')}}
			if *updateREPL {
				var keep []txtar.File
				for _, f := range ar.Files {
					if f.Name == "args.json" || f.Name == "session.txt" || strings.HasPrefix(f.Name, "input/") {
						keep = append(keep, f)
					}
				}
				ar.Files = append(keep, outputs...)
				if e = os.WriteFile(path, txtar.Format(ar), 0644); e != nil {
					t.Fatal(e)
				}
			} else {
				for _, f := range outputs {
					if !bytes.Equal(f.Data, files[f.Name]) {
						t.Errorf("%s mismatch; explicitly -update-repl\ngot %s\nwant %s", f.Name, f.Data, files[f.Name])
					}
				}
			}
		})
	}
}
