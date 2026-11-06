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

	"github.com/arran4/golang-gtroff/query"
)

func TestQueryExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build query executable: %v %s", err, out)
	}
	for name, data := range map[string]string{"main.1": ".SH ENVIRONMENT\n.so terms\n", "terms": ".TP\n.B TOOL_HOME\nDirectory.\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name        string
		args        []string
		stdin       string
		count, exit int
	}{
		{name: "includes", args: []string{"query", `section[name="ENVIRONMENT"] definition`, filepath.Join(dir, "main.1")}, count: 1},
		{name: "no includes", args: []string{"query", "--no-includes", `section[name="ENVIRONMENT"] definition`, filepath.Join(dir, "main.1")}},
		{name: "stdin", args: []string{"query", `definition > term`}, stdin: ".TP\n.B TOOL_MODE\nMode.\n", count: 1},
		{name: "syntax", args: []string{"query", "--view", "syntax", `macro[arg.0="TOOL_HOME"]`, filepath.Join(dir, "main.1")}, count: 1},
		{name: "strict", args: []string{"query", "--strict", `line`}, stdin: ".UNKNOWN literal\n", exit: 1},
		{name: "limit", args: []string{"query", "--max-results", "1", `paragraph`}, stdin: "one\n.PP\ntwo\n", count: 1, exit: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := exec.Command(binary, tc.args...)
			command.Stdin = strings.NewReader(tc.stdin)
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
			if exit != tc.exit {
				t.Fatalf("exit %d want %d: %s", exit, tc.exit, stderr.String())
			}
			var report query.Result
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("invalid executable report %s: %v", out.String(), err)
			}
			if len(report.Matches) != tc.count || report.Command != "query" || report.Complete != (tc.exit != 3) {
				t.Fatal("incorrect generated command behavior", out.String())
			}
			if tc.name == "includes" && (report.Matches[0].Source.Name != "terms" || report.Matches[0].Term != "TOOL_HOME") {
				t.Fatal("include ownership lost")
			}
			if tc.name == "limit" && report.Error == "" {
				t.Fatal("partial machine report lacks query error")
			}
			if tc.name == "strict" && report.InputError == "" {
				t.Fatal("strict report lacks input error")
			}
		})
	}
}
