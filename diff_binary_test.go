package groff_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/docdiff"
	"github.com/arran4/golang-gtroff/source"
)

func TestDiffExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build diff: %v %s", err, out)
	}
	header := ".TH TOOL 1\n.SH ENVIRONMENT\n"
	for name, data := range map[string]string{"old.1": header + ".TP\n.B HOME\nOld.\n", "new.1": header + ".TP\n.B HOME\nNew.\n", "format.1": ".TH \"TOOL\" \"1\"\n.SH ENVIRONMENT\n.TP 8\n\\fBHOME\\fR\nOld.\n", "before/main.1": header + ".so body.roff\n", "after/main.1": header + ".so body.roff\n", "before/body.roff": ".TP\n.B HOME\nOld.\n", "after/body.roff": ".TP\n.B HOME\nNew.\n"} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name          string
		args          []string
		stdin         string
		exit, changes int
		complete      bool
	}{
		{name: "semantic change", args: []string{"diff", "--format", "json", filepath.Join(dir, "old.1"), filepath.Join(dir, "new.1")}, exit: 1, changes: 1, complete: true},
		{name: "semantic formatting", args: []string{"diff", "--format", "json", filepath.Join(dir, "old.1"), filepath.Join(dir, "format.1")}, complete: true},
		{name: "source formatting", args: []string{"diff", "--format", "json", "--view", "source", filepath.Join(dir, "old.1"), filepath.Join(dir, "format.1")}, exit: 1, changes: 1, complete: true},
		{name: "included change", args: []string{"diff", "--format", "json", filepath.Join(dir, "before/main.1"), filepath.Join(dir, "after/main.1")}, exit: 1, changes: 1, complete: true},
		{name: "disabled includes", args: []string{"diff", "--format", "json", "--no-includes", filepath.Join(dir, "before/main.1"), filepath.Join(dir, "after/main.1")}, complete: true},
		{name: "stdin", args: []string{"diff", "--format", "json", "-", filepath.Join(dir, "new.1")}, stdin: header + ".TP\n.B HOME\nOld.\n", exit: 1, changes: 1, complete: true},
		{name: "shared budget", args: []string{"diff", "--format", "json", "--max-files", "1", filepath.Join(dir, "old.1"), filepath.Join(dir, "new.1")}, exit: 3},
		{name: "strict", args: []string{"diff", "--format", "json", "--strict", "--no-includes", filepath.Join(dir, "before/main.1"), filepath.Join(dir, "after/main.1")}, exit: 1, complete: true},
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
				t.Fatalf("exit %d want %d: %s\n%s", exit, tc.exit, stderr.String(), out.String())
			}
			var report docdiff.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("invalid JSON: %v %s", err, out.String())
			}
			if report.SchemaVersion != 1 || report.Command != "diff" || report.Complete != tc.complete || len(report.Changes) != tc.changes {
				t.Fatalf("report contract %+v", report)
			}
			if tc.name == "included change" && report.Changes[0].Before.Location.Source.ID != 2 {
				t.Fatal("include provenance lost")
			}
		})
	}
}
func TestDiffRealEnvironRevision(t *testing.T) {
	data, err := queryRealCases.ReadFile("testdata/environ_snippet.7")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	revision := strings.Replace(text, "The value can be anything that can be represented as a string.", "The value is a revised, source-attributed string description.", 1)
	if text == revision {
		t.Fatal("real revision trigger missing")
	}
	before, err := source.Read(context.Background(), strings.NewReader(text), "environ-before.7", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := source.Read(context.Background(), strings.NewReader(revision), "environ-after.7", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	report, err := docdiff.Compare(context.Background(), docdiff.Input{Sources: before}, docdiff.Input{Sources: after}, docdiff.Options{})
	if err != nil || len(report.Changes) != 1 || report.Changes[0].Before.Kind != docdiff.Section || report.Changes[0].Before.Identity != "DESCRIPTION" {
		t.Fatalf("real changed prose not found: %v %+v", err, report)
	}
	if report.Changes[0].Before.Location.Span.Start.Filename != "environ-before.7" || report.Changes[0].After.Location.Span.Start.Filename != "environ-after.7" {
		t.Fatal("real revision positions lost")
	}
}
