package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/manlib"
)

func TestAnalyzeSharedIncludePolicy(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".TH TOOL 1\n.so common\n")}, "common": {Data: []byte(".nr X 1\n")}}
	for _, disabled := range []bool{false, true} {
		args := []string{"analyze", "--format", "json"}
		if disabled {
			args = append(args, "--no-includes")
		}
		args = append(args, "docs/main.1")
		var out, stderr bytes.Buffer
		code := Run(args, Environment{Out: &out, Err: &stderr, OpenRoot: func(root string) (manlib.Root, io.Closer, error) {
			if root != "docs" {
				t.Fatal(root)
			}
			return manlib.Root{FS: fsys}, nil, nil
		}})
		if code != ExitOK || stderr.Len() != 0 {
			t.Fatalf("analyze code %d: %s", code, stderr.String())
		}
		var report analyze.Report
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if !report.Flags.HasInclude || len(report.Includes) != 1 {
			t.Fatal("include evidence missing")
		}
		if disabled {
			if len(report.Sources) != 1 || report.Includes[0].State != "disabled" {
				t.Fatal("no-includes ignored")
			}
		} else {
			if len(report.Sources) != 2 || report.Includes[0].State != "resolved" || !report.Flags.HasDefinitions {
				t.Fatal("default source policy not shared")
			}
		}
	}
}
func TestAnalyzeRulesAndBoundaryFailures(t *testing.T) {
	for _, args := range [][]string{{"analyze", "a", "b"}, {"analyze", "--format", "yaml"}, {"analyze", "--format", "json", "--details"}, {"analyze", "--max-findings", "0"}, {"analyze", "--max-work", "0"}, {"analyze", "--max-analysis-nodes", "0"}} {
		var out, stderr bytes.Buffer
		code := Run(args, Environment{In: panicReader{}, Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { t.Fatal("invalid options opened a file"); return nil, nil }})
		if code != ExitUsage || out.Len() != 0 {
			t.Fatal("invalid analyze options accepted", args, code)
		}
	}
	for _, config := range []struct {
		input    string
		closeErr error
		want     int
	}{
		{`{"schemaVersion":1,"rules":[{"id":"project.title","match":{"macro":".TH"},"feature":"project-title","reason":"Review title"}]}`, nil, ExitOK},
		{`{"schemaVersion":1,"unknown":true}`, nil, ExitUsage},
		{`{"schemaVersion":1}`, errBoundary, ExitIO},
	} {
		reader := &closeReader{Reader: strings.NewReader(config.input), err: config.closeErr}
		var out, stderr bytes.Buffer
		var input io.Reader = strings.NewReader(".TH TOOL 1\n")
		if config.want != ExitOK {
			input = panicReader{}
		}
		code := Run([]string{"analyze", "--format", "json", "--rules", "rules.json"}, Environment{In: input, Out: &out, Err: &stderr, Open: func(name string) (io.ReadCloser, error) {
			if name != "rules.json" {
				t.Fatal(name)
			}
			return reader, nil
		}})
		if code != config.want || !reader.closed {
			t.Fatal("rules outcome/ownership incorrect", code, stderr.String())
		}
		if code == ExitOK {
			var report analyze.Report
			if err := json.Unmarshal(out.Bytes(), &report); err != nil || !report.HasRule("project.title") {
				t.Fatal("file rules not applied", err)
			}
		}
	}
	for _, format := range []string{"text", "json", "jsonl"} {
		if code := Run([]string{"analyze", "--format", format}, Environment{In: strings.NewReader("text\n"), Out: shortWriter{}}); code != ExitIO {
			t.Fatal("analysis output short write ignored", format)
		}
	}
}
