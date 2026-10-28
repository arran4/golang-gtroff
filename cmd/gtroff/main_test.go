package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
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
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "man-tool.1"))
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
		{name: "render reject format", args: []string{"render", "--format", "gif"}, code: 2, stderrContains: "render format must be html"},
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
	t.Run("raster executable pages and preservation", func(t *testing.T) {
		execute := func(input string, args ...string) (int, []byte, string) {
			c := exec.CommandContext(ctx, binary, args...)
			c.Dir = work
			c.Stdin = strings.NewReader(input)
			var out, diags bytes.Buffer
			c.Stdout = &out
			c.Stderr = &diags
			err := c.Run()
			if err == nil {
				return 0, out.Bytes(), diags.String()
			}
			var e *exec.ExitError
			if !errors.As(err, &e) {
				t.Fatal(err)
			}
			return e.ExitCode(), out.Bytes(), diags.String()
		}
		code, data, diags := execute("", "render", "--format", "png", "--section", "OPTIONS", "--single", "my page.1")
		if code != 0 || diags != "" {
			t.Fatalf("PNG stdout %d %s", code, diags)
		}
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		long := ".TH TOOL 1\n.SH CONTENT\n" + strings.Repeat(".PP\nvisible paragraph\n", 25)
		if err := os.WriteFile(filepath.Join(work, "long.1"), []byte(long), 0600); err != nil {
			t.Fatal(err)
		}
		if code, data, msg := execute("", "render", "--format", "png", "--width", "400", "--page-height", "300", "-o", "pages.png", "long.1"); code != 0 || len(data) != 0 || msg != "" {
			t.Fatalf("pages %d %s", code, msg)
		}
		pages, err := filepath.Glob(filepath.Join(work, "pages-*.png"))
		if err != nil || len(pages) < 2 {
			t.Fatal("numbered pages missing", pages, err)
		}
		saved := map[string][]byte{}
		for _, path := range pages {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = png.Decode(bytes.NewReader(data)); err != nil {
				t.Fatal(err)
			}
			saved[path] = data
		}
		if _, err := os.Stat(filepath.Join(work, "pages.png")); !os.IsNotExist(err) {
			t.Fatal("multi-page produced ambiguous base file")
		}
		if code, data, _ := execute(long, "render", "--format", "png", "--width", "400", "--page-height", "300"); code != 2 || len(data) > 0 {
			t.Fatal("multi-page stdout was not refused")
		}
		if code, _, _ := execute(".UNKNOWN literal\n", "render", "--format", "png", "--strict", "-o", "pages.png"); code != 1 {
			t.Fatal("strict image accepted")
		}
		for path, want := range saved {
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatal("strict failure damaged image")
			}
		}
		first := filepath.Join(work, "blocked-001.png")
		if err := os.WriteFile(first, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(work, "blocked-002.png"), 0700); err != nil {
			t.Fatal(err)
		}
		if code, _, _ := execute("", "render", "--format", "png", "--width", "400", "--page-height", "300", "-o", "blocked.png", "long.1"); code != 3 {
			t.Fatal("unsafe second destination accepted")
		}
		got, err := os.ReadFile(first)
		if err != nil || string(got) != "original" {
			t.Fatal("batch preflight damaged first destination")
		}
	})
	t.Run("HTTP executable collection", func(t *testing.T) {
		root := filepath.Join(work, "collection")
		if err := os.MkdirAll(filepath.Join(root, "man1"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "man1", "man-tool.1"), source, 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.CommandContext(ctx, binary, "serve", "--root", root, "--listen", "127.0.0.1:0", "--assets", "omit-fonts")
		stderr, err := command.StderrPipe()
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		command.Stdout = &out
		if err = command.Start(); err != nil {
			t.Fatal(err)
		}
		stopped := false
		defer func() {
			if !stopped {
				_ = command.Process.Kill()
			}
		}()
		ready := make(chan string, 1)
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				select {
				case ready <- scanner.Text():
				default:
				}
			}
		}()
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		var line string
		select {
		case line = <-ready:
		case err := <-done:
			t.Fatal("server exited before ready", err)
		case <-time.After(5 * time.Second):
			t.Fatal("server did not announce readiness")
		}
		var address string
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "http://") {
				address = field
			}
		}
		if address == "" {
			t.Fatal("missing listener address", line)
		}
		client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{}}
		for _, target := range []string{"/search?q=inspect", "/page/1/man-tool", "/page/1/man-tool?heading=OPTIONS"} {
			res, err := client.Get(address + target)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if err != nil || res.StatusCode != 200 || !bytes.Contains(body, []byte("man-tool(1)")) {
				t.Fatalf("HTTP %s %d %s %v", target, res.StatusCode, body, err)
			}
		}
		if runtime.GOOS == "windows" {
			_ = command.Process.Kill()
		} else if err := command.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			stopped = true
			if runtime.GOOS != "windows" && err != nil {
				t.Fatal("graceful process shutdown", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("HTTP process did not stop")
		}
		if out.Len() != 0 {
			t.Fatal("serve wrote stdout")
		}
	})
	t.Run("README quickstart fences", func(t *testing.T) { smokeReadme(t, binary) })

}

// Only explicitly marked shell fences run. Commands are passed as argv, with
// a single optional stdout redirect; no shell or platform command is invoked.
func smokeReadme(t *testing.T, binary string) {
	t.Helper()
	work := t.TempDir()
	if err := os.Mkdir(filepath.Join(work, "examples"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"man-tool.1", "convert.html"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, "examples", name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "readme.md"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, part := range strings.Split(string(readme), "```sh\n")[1:] {
		block, _, ok := strings.Cut(part, "```")
		if !ok {
			t.Fatal("unclosed shell fence")
		}
		if !strings.HasPrefix(block, "# smoke\n") {
			continue
		}
		for _, line := range strings.Split(block, "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			args := strings.Fields(line)
			if args[0] != "./gtroff" {
				t.Fatalf("unsupported smoke command %q", line)
			}
			args = args[1:]
			destination := ""
			for i, arg := range args {
				if arg == ">" {
					if i != len(args)-2 {
						t.Fatal("unsupported redirect")
					}
					destination = args[i+1]
					args = args[:i]
					break
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			command := exec.CommandContext(ctx, binary, args...)
			command.Dir = work
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			cancel()
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("README %s: %v: %s", line, err, stderr.String())
			}
			if destination != "" {
				if stdout.Len() == 0 {
					t.Fatal("redirected command wrote nothing")
				}
				if err := os.WriteFile(filepath.Join(work, destination), stdout.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			count++
		}
	}
	if count < 10 {
		t.Fatalf("expected meaningful quickstart smoke, got %d commands", count)
	}
	for _, name := range []string{"page.html", "options.html", "converted.html", "converted.1", "options.json", "page.png", "page.jpg", "options.png", "gtroff-render.1", "gtroff.bash"} {
		data, err := os.ReadFile(filepath.Join(work, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("README missing output %s: %v", name, err)
		}
		if strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") {
			im, _, err := image.Decode(bytes.NewReader(data))
			if err != nil || im.Bounds().Dx() == 0 {
				t.Fatalf("README invalid image %s: %v", name, err)
			}
		}
		if strings.HasSuffix(name, ".html") && !bytes.Contains(data, []byte("<article")) {
			t.Fatalf("README HTML lost article: %s", name)
		}
	}
	pages, err := filepath.Glob(filepath.Join(work, "pages-*.png"))
	if err != nil || len(pages) < 2 {
		t.Fatal("README pagination did not produce numbered pages", err)
	}
}
