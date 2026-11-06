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

	"github.com/arran4/golang-gtroff/analyze"
)

func TestAnalyzeExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build analyzer: %v %s", err, out)
	}
	for name, data := range map[string]string{"main.1": ".TH TOOL 1\n.so common\n", "common": ".nr VALUE 1\n", "rules.json": `{"schemaVersion":1,"rules":[{"id":"project.title","match":{"macro":".TH"},"feature":"project-title","reason":"Review title"}]}`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name          string
		args          []string
		input         string
		sources, exit int
		rule          string
	}{
		{name: "default includes", args: []string{"analyze", "--format", "json", filepath.Join(dir, "main.1")}, sources: 2},
		{name: "disabled includes", args: []string{"analyze", "--format", "json", "--no-includes", filepath.Join(dir, "main.1")}, sources: 1},
		{name: "stdin", args: []string{"analyze", "--format", "json"}, input: ".so common\n", sources: 1},
		{name: "stdin root", args: []string{"analyze", "--format", "json", "--root", dir}, input: ".so common\n", sources: 2},
		{name: "custom rules", args: []string{"analyze", "--format", "json", "--rules", filepath.Join(dir, "rules.json"), filepath.Join(dir, "main.1")}, sources: 2, rule: "project.title"},
		{name: "strict", args: []string{"analyze", "--format", "json", "--strict"}, input: ".CUSTOM x\n", sources: 1, exit: 1},
		{name: "bounded", args: []string{"analyze", "--format", "json", "--max-findings", "1"}, input: ".TH TOOL 1\n.SH NAME\ntext\n", sources: 1, exit: 3},
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
			if exit != tc.exit {
				t.Fatalf("exit %d want %d: %s", exit, tc.exit, stderr.String())
			}
			var report analyze.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("invalid JSON report: %v %s", err, out.String())
			}
			if report.Command != "analyze" || report.SchemaVersion != 1 || report.CapabilityVersion != 1 || len(report.Sources) != tc.sources {
				t.Fatal("generated analyzer/report contract incorrect")
			}
			if tc.rule != "" && !report.HasRule(tc.rule) {
				t.Fatal("file-defined question absent")
			}
			if tc.exit == 3 && (report.Complete || report.Error == "") {
				t.Fatal("limit looked successful")
			}
			if tc.exit == 1 && report.InputError == "" {
				t.Fatal("strict input error omitted")
			}
		})
	}
}
