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

	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func TestExtractExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build extract: %v %s", err, out)
	}
	main := filepath.Join(dir, "main.1")
	child := filepath.Join(dir, "child.roff")
	page := ".TH TOOL 1\n.SH ENVIRONMENT\n.so child.roff\n"
	body := ".TP\n.B TOOL_HOME\nDirectory.\n"
	if err := os.WriteFile(main, []byte(page), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		args          []string
		input         string
		exit, records int
		output        string
	}{
		{name: "includes", args: []string{"extract", "--format", "json", "definition", main}, records: 1},
		{name: "disabled", args: []string{"extract", "--format", "json", "--no-includes", "definition", main}},
		{name: "stdin", args: []string{"extract", "--format", "json", "definition"}, input: body, records: 1},
		{name: "stdin root", args: []string{"extract", "--format", "json", "--root", dir, "definition"}, input: page, records: 1},
		{name: "atomic output", args: []string{"extract", "--format", "raw", "-o", filepath.Join(dir, "out.roff"), "definition", main}, output: filepath.Join(dir, "out.roff")},
		{name: "root protected", args: []string{"extract", "--format", "raw", "-o", main, "definition", main}, exit: 3},
		{name: "include protected", args: []string{"extract", "--format", "raw", "-o", child, "definition", main}, exit: 3},
		{name: "strict no write", args: []string{"extract", "--strict", "--format", "raw", "-o", filepath.Join(dir, "strict.roff"), "section", main}, exit: 1},
		{name: "partial json", args: []string{"extract", "--max-records", "1", "--format", "json", "definition"}, input: body + body, records: 1, exit: 3},
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
				t.Fatalf("exit %d want %d: %s %s", exit, tc.exit, stderr.String(), out.String())
			}
			if out.Len() > 0 {
				var report extract.Report
				if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Command != "extract" || len(report.Records) != tc.records {
					t.Fatalf("JSON contract: %v %s", err, out.String())
				}
			}
			if tc.output != "" {
				data, err := os.ReadFile(tc.output)
				if err != nil || string(data) != body || out.Len() != 0 {
					t.Fatal("atomic content", string(data), err)
				}
			}
			for path, want := range map[string]string{main: page, child: body} {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != want {
					t.Fatal("original changed", path, err)
				}
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dir, "strict.roff")); !os.IsNotExist(err) {
		t.Fatal("strict created file")
	}
}
func TestExtractRealEnvironSnippet(t *testing.T) {
	input, err := queryRealCases.ReadFile("testdata/environ_snippet.7")
	if err != nil {
		t.Fatal(err)
	}
	set, err := source.Read(context.Background(), bytes.NewReader(input), "environ_snippet.7", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	selector, err := query.ParseSelector(`section[name="DESCRIPTION"] paragraph[normalized-text*="environment variable"]`)
	if err != nil {
		t.Fatal(err)
	}
	report, err := extract.Select(context.Background(), set, selector, extract.Options{})
	if err != nil || len(report.Records) == 0 {
		t.Fatal("real content absent", err)
	}
	for _, record := range report.Records {
		if record.Raw != string(input[record.Span.Start.Offset:record.Span.End.Offset]) || record.Source.Name != "environ_snippet.7" || len(record.Headings) == 0 {
			t.Fatal("real content provenance lost")
		}
	}
}
