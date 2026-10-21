package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/fontset"
)

// This deliberately exercises the built executable: real argument parsing,
// process streams, exit statuses and the filesystem adapter in main.
func TestExecutableFlows(t *testing.T) {
	work := t.TempDir()
	binary := filepath.Join(work, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags", "-X main.version=integration-test -X main.commit=abc123 -X main.date=2026-10-21", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v\n%s", err, out)
	}
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "simple.1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "my page.1"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "--page.1"), []byte(".B hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "secret.1"), []byte("must-not-be-included"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "custom.ttf"), fontset.Default().Mono, 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name                           string
		args                           []string
		stdin                          string
		code                           int
		stdoutContains, stderrContains string
		json                           bool
	}{
		{name: "version", args: []string{"version"}, stdoutContains: "gtroff integration-test (commit abc123, built 2026-10-21)"},
		{name: "help", args: []string{"help", "ast"}, stdoutContains: "node:ID"},
		{name: "file with spaces", args: []string{"validate", "--strict", "my page.1"}, stdoutContains: "valid syntax (semantic validation not performed)"},
		{name: "stdin AST", args: []string{"ast", "--select", "section:OPTIONS", "-"}, stdin: string(source), json: true, stdoutContains: "\"rootIds\""},
		{name: "empty stdin", args: []string{"ast"}, json: true, stdoutContains: "\"kind\":\"document\""},
		{name: "strict unknown", args: []string{"validate", "--strict", "--format", "json"}, stdin: ".CUSTOM Snow\n", code: 1, json: true, stdoutContains: "\"status\":\"invalid\""},
		{name: "malformed", args: []string{"validate"}, stdin: ".B \"unterminated\n", code: 1, stdoutContains: "invalid syntax", stderrContains: "<stdin>:1:1: error:"},
		{name: "missing file", args: []string{"validate", "missing.1"}, code: 3, stderrContains: "gtroff: open \"missing.1\":"},
		{name: "ordinary usage", args: []string{"validate", "--package", "absent"}, code: 2, stderrContains: "package must be man, ms or roff"},
		{name: "no matches", args: []string{"ast", "--select", "section:ABSENT"}, stdin: string(source), code: 2, stderrContains: "matched no nodes"},
		{name: "dash file", args: []string{"validate", "--", "--page.1"}, stdoutContains: "valid syntax"},
		{name: "no includes", args: []string{"ast"}, stdin: ".so secret.1\n", json: true, stdoutContains: "include preserved; target was not loaded"},
		{name: "HTML full default fonts", args: []string{"render", "my page.1"}, stdoutContains: "data:font/ttf;base64,"},
		{name: "HTML selected fragment", args: []string{"render", "--section", "OPTIONS", "--fragment", "--assets", "omit-fonts", "my page.1"}, stdoutContains: "<dl class=\"definitions\""},
		{name: "HTML custom font", args: []string{"render", "--font-regular", "custom.ttf", "my page.1"}, stdoutContains: "@font-face"},
		{name: "convert HTML stdin", args: []string{"convert", "--from", "html", "--to", "roff"}, stdin: "<h1>TOOL</h1><p>.danger</p>", stdoutContains: `\&\f[R].danger`},
		{name: "convert roff stdin", args: []string{"convert", "--from", "roff", "--to", "html", "--fragment", "--assets", "omit-fonts"}, stdin: ".TH TOOL 1\n.SH NAME\ntext\n", stdoutContains: "<article"},
		{name: "convert strict hostile", args: []string{"convert", "--from", "html", "--to", "roff", "--strict"}, stdin: "<script>literal</script>", code: 1, stderrContains: "unsupported <script>"},
		{name: "HTML reject format", args: []string{"render", "--format", "png"}, code: 2, stderrContains: "render format must be html"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			command := exec.CommandContext(runCtx, binary, tc.args...)
			command.Dir = work
			command.Stdin = strings.NewReader(tc.stdin)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != tc.code {
				t.Fatalf("exit=%d want=%d stderr=%s", code, tc.code, stderr.String())
			}
			if tc.stdoutContains != "" && !strings.Contains(stdout.String(), tc.stdoutContains) {
				t.Fatalf("stdout missing %q: %s", tc.stdoutContains, stdout.String())
			}
			if tc.stderrContains != "" && !strings.Contains(stderr.String(), tc.stderrContains) {
				t.Fatalf("stderr missing %q: %s", tc.stderrContains, stderr.String())
			}
			if tc.stderrContains == "" && stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}
			if tc.stdoutContains == "" && stdout.Len() != 0 {
				t.Fatalf("failure wrote unexpected stdout: %s", stdout.String())
			}
			if strings.Contains(stdout.String(), "must-not-be-included") {
				t.Fatal("CLI loaded arbitrary include target")
			}
			if tc.json {
				var doc struct {
					SchemaVersion int    `json:"schemaVersion"`
					Source        string `json:"source"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
					t.Fatal(err)
				}
				if doc.SchemaVersion != 1 {
					t.Fatal("wrong schema version")
				}
				if len(tc.args) > 0 && tc.args[0] == "ast" && doc.Source != tc.stdin {
					t.Fatal("executable AST lost source bytes")
				}
			}
		})
	}
	t.Run("HTML atomic output preserves failed destinations and source", func(t *testing.T) {
		execute := func(args []string, input string) (int, string, string) {
			runCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			c := exec.CommandContext(runCtx, binary, args...)
			c.Dir = work
			c.Stdin = strings.NewReader(input)
			var out, stderr bytes.Buffer
			c.Stdout = &out
			c.Stderr = &stderr
			err := c.Run()
			code := 0
			if err != nil {
				var e *exec.ExitError
				if !errors.As(err, &e) {
					t.Fatal(err)
				}
				code = e.ExitCode()
			}
			return code, out.String(), stderr.String()
		}
		if code, out, stderr := execute([]string{"render", "-o", "page.html", "my page.1"}, ""); code != 0 || out != "" || stderr != "" {
			t.Fatalf("atomic output code=%d stderr=%s", code, stderr)
		}
		saved, err := os.ReadFile(filepath.Join(work, "page.html"))
		if err != nil || !bytes.HasPrefix(saved, []byte("<!doctype html>")) {
			t.Fatal("no standalone HTML file")
		}
		if code, _, _ := execute([]string{"render", "--strict", "-o", "page.html"}, ".CUSTOM unsafe\n"); code != 1 {
			t.Fatal("strict render did not fail")
		}
		again, err := os.ReadFile(filepath.Join(work, "page.html"))
		if err != nil || !bytes.Equal(saved, again) {
			t.Fatal("strict failure damaged existing output")
		}
		if code, _, _ := execute([]string{"render", "-o", "my page.1", "my page.1"}, ""); code != 3 {
			t.Fatal("source replacement not rejected")
		}
		if runtime.GOOS != "windows" {
			weird := filepath.Join(work, "<stdin>")
			if err := os.WriteFile(weird, source, 0600); err != nil {
				t.Fatal(err)
			}
			if code, _, _ := execute([]string{"render", "-o", "<stdin>", "<stdin>"}, ""); code != 3 {
				t.Fatal("a real file named <stdin> bypassed source protection")
			}
			retained, err := os.ReadFile(weird)
			if err != nil || !bytes.Equal(retained, source) {
				t.Fatal("named source lost through stdin label collision")
			}
		}
		unchanged, err := os.ReadFile(filepath.Join(work, "my page.1"))
		if err != nil || !bytes.Equal(source, unchanged) {
			t.Fatal("source file was overwritten")
		}
	})
	t.Run("conversion atomic file loop", func(t *testing.T) {
		execute := func(args ...string) (int, string) {
			c := exec.CommandContext(ctx, binary, args...)
			c.Dir = work
			out, err := c.CombinedOutput()
			if err == nil {
				return 0, string(out)
			}
			var e *exec.ExitError
			if !errors.As(err, &e) {
				t.Fatal(err)
			}
			return e.ExitCode(), string(out)
		}
		if code, msg := execute("convert", "--from", "roff", "--to", "html", "-i", "my page.1", "-o", "converted.html"); code != 0 {
			t.Fatalf("export %d %s", code, msg)
		}
		if code, msg := execute("convert", "--from", "html", "--to", "roff", "--strict", "-i", "converted.html", "-o", "canonical.1"); code != 0 {
			t.Fatalf("import %d %s", code, msg)
		}
		saved, err := os.ReadFile(filepath.Join(work, "canonical.1"))
		if err != nil || !bytes.Contains(saved, []byte(".TH ")) {
			t.Fatal("canonical file missing", err)
		}
		if code, msg := execute("convert", "--from", "html", "--to", "roff", "-i", "converted.html", "-o", "converted.html"); code != 3 {
			t.Fatalf("source overwrite %d %s", code, msg)
		}
		if err := os.WriteFile(filepath.Join(work, "hostile.html"), []byte("<script>literal</script>"), 0600); err != nil {
			t.Fatal(err)
		}
		if code, msg := execute("convert", "--from", "html", "--to", "roff", "--strict", "-i", "hostile.html", "-o", "canonical.1"); code != 1 {
			t.Fatalf("strict %d %s", code, msg)
		}
		after, err := os.ReadFile(filepath.Join(work, "canonical.1"))
		if err != nil || !bytes.Equal(saved, after) {
			t.Fatal("strict conversion damaged output")
		}
	})

}
