package groff_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourcesExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	for name, data := range map[string]string{"main.1": ".so child\n", "child": ".TP\n.B TOOL_HOME\nDirectory.\n", "broken.1": ".so missing\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name        string
		args        []string
		input       string
		files, exit int
		state       string
	}{
		{name: "default", args: []string{"sources", filepath.Join(dir, "main.1")}, files: 2, state: "resolved"},
		{name: "disabled", args: []string{"sources", "--no-includes", filepath.Join(dir, "main.1")}, files: 1, state: "disabled"},
		{name: "stdin", args: []string{"sources"}, input: ".so child\n", files: 1, state: "unresolved"},
		{name: "stdin root", args: []string{"sources", "--root", dir}, input: ".so child\n", files: 2, state: "resolved"},
		{name: "strict missing", args: []string{"sources", "--strict", filepath.Join(dir, "broken.1")}, files: 1, exit: 1, state: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := exec.Command(binary, tc.args...)
			command.Stdin = strings.NewReader(tc.input)
			var out, stderr bytes.Buffer
			command.Stdout = &out
			command.Stderr = &stderr
			err := command.Run()
			exit := 0
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					exit = e.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			if exit != tc.exit || stderr.Len() != 0 {
				t.Fatalf("exit %d stderr %s", exit, stderr.String())
			}
			var report struct {
				SchemaVersion int
				Command       string
				SourceSet     struct {
					Files []struct {
						Name   string
						Source string
					}
					Edges []struct{ State string }
				}
			}
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.SchemaVersion != 1 || report.Command != "sources" || len(report.SourceSet.Files) != tc.files || len(report.SourceSet.Edges) != 1 || report.SourceSet.Edges[0].State != tc.state {
				t.Fatalf("incorrect assembled command behavior: %s", out.String())
			}
		})
	}
}
