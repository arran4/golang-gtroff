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

	"github.com/arran4/golang-gtroff/lint"
)

func TestLintExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lint: %v %s", err, output)
	}
	good := ".TH TOOL 1\n.SH NAME\ntool \\- example\n.SH SYNOPSIS\n.B tool\n.SH DESCRIPTION\nDescription.\n"
	for name, data := range map[string]string{"main.1": ".TH TOOL 1\n.so body.roff\n", "body.roff": strings.SplitN(good, "\n", 2)[1], "other.1": strings.Replace(good, "TOOL", "OTHER", 1), "refs.1": good + ".MR other 1\n.MR absent 3\n", "policy.json": `{"schemaVersion":1,"policy":{"requiredSections":[]}}`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name                string
		args                []string
		input               string
		docs, sources, exit int
		rule                string
	}{
		{name: "includes", args: []string{"lint", "--format", "json", filepath.Join(dir, "main.1")}, docs: 1, sources: 2},
		{name: "no includes", args: []string{"lint", "--format", "json", "--no-includes", filepath.Join(dir, "main.1")}, docs: 1, sources: 1, rule: "man.required-section"},
		{name: "stdin root", args: []string{"lint", "--format", "json", "--root", dir}, input: ".TH TOOL 1\n.so body.roff\n", docs: 1, sources: 2},
		{name: "stdin isolated", args: []string{"lint", "--format", "json"}, input: good, docs: 1, sources: 1},
		{name: "missing", args: []string{"lint", "--format", "json"}, input: ".TH TOOL 1\n", docs: 1, sources: 1, exit: 1, rule: "man.required-section"},
		{name: "collection", args: []string{"lint", "--format", "json", "--closed-world", filepath.Join(dir, "refs.1"), filepath.Join(dir, "other.1")}, docs: 2, sources: 2, rule: "reference.local-target"},
		{name: "strict warnings", args: []string{"lint", "--format", "json", "--strict"}, input: good + ".SH NAME\ntool \\- repeat\n", docs: 1, sources: 1, exit: 1, rule: "man.duplicate-section"},
		{name: "policy", args: []string{"lint", "--format", "json", "--rules", filepath.Join(dir, "policy.json")}, input: ".TH TOOL 1\n", docs: 1, sources: 1},
		{name: "partial", args: []string{"lint", "--format", "json", "--max-findings", "1"}, input: ".TH TOOL 1\n", docs: 1, sources: 1, exit: 3},
		{name: "collection resource", args: []string{"lint", "--format", "json", "--max-files", "1", filepath.Join(dir, "other.1"), filepath.Join(dir, "other.1")}, docs: 2, sources: 1, exit: 3},
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
				t.Fatalf("exit %d want %d: %s\n%s", exit, tc.exit, stderr.String(), out.String())
			}
			var report lint.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("decode: %v %s", err, out.String())
			}
			sources := 0
			for _, doc := range report.Documents {
				sources += len(doc.Sources)
			}
			if report.SchemaVersion != 1 || report.Command != "lint" || len(report.Documents) != tc.docs || sources != tc.sources {
				t.Fatalf("report contract %+v", report)
			}
			if tc.rule != "" {
				found := false
				for _, finding := range report.Findings {
					if finding.Rule == tc.rule {
						found = true
					}
				}
				if !found {
					t.Fatalf("expected %s", tc.rule)
				}
			}
			if tc.exit == 3 && report.Error == "" && !report.Failed() {
				t.Fatal("resource failure appeared valid")
			}
		})
	}
}
