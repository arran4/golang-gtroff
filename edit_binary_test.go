package groff_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEditExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build edit: %v %s", e, b)
	}
	run := func(args []string, input string, want int) string {
		t.Helper()
		c := exec.Command(binary, args...)
		c.Stdin = strings.NewReader(input)
		var out, stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr
		e := c.Run()
		code := 0
		if e != nil {
			if exit, ok := e.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		if code != want {
			t.Fatalf("%v: exit %d want %d: %s %s", args, code, want, out.String(), stderr.String())
		}
		return out.String()
	}
	original := ".\\\" comment\r\n.UNKNOWN retained\r\n.SH ENVIRONMENT\r\n.TP\r\n.B HOME\r\nOld directory.\r\n\xff\r\n"
	main := filepath.Join(dir, "main.1")
	copyName := filepath.Join(dir, "updated.1")
	if e := os.WriteFile(main, []byte(original), 0600); e != nil {
		t.Fatal(e)
	}
	command := []string{"edit", "--select", "definition", "--operation", "find-replace-source", "--find", "Old", "--text", "New"}
	run(append(append([]string{}, command...), main), "", 0)
	assertBytes := func(path, want string) {
		t.Helper()
		b, e := os.ReadFile(path)
		if e != nil || string(b) != want {
			t.Fatalf("%s: %q want %q: %v", path, b, want, e)
		}
	}
	assertBytes(main, original)
	run(append(append([]string{}, command...), "--dry-run", "--in-place", main), "", 0)
	assertBytes(main, original)
	run(append(append([]string{}, command...), "-o", copyName, main), "", 0)
	assertBytes(copyName, strings.ReplaceAll(original, "Old", "New"))
	assertBytes(main, original)
	run(append(append([]string{}, command...), "-o", main, main), "", 3)
	assertBytes(main, original)
	run(append(append([]string{}, command...), "--in-place", main), "", 0)
	assertBytes(main, strings.ReplaceAll(original, "Old", "New"))
	run([]string{"edit", "--strict", "--select", "section", "--operation", "delete"}, ".SH A\na\n.SH B\nb\n", 1)
	run([]string{"edit", "--nth", "0", "--operation", "delete"}, "text\n", 2)
	raw := run([]string{"edit", "--operation", "replace-source", "--text", "new\r\n", "--format", "raw"}, "old\r\n", 0)
	if raw != "new\r\n" {
		t.Fatalf("raw stdout: %q", raw)
	}
	// Actual included-file ownership, guarded commit and repeat occurrences.
	child := filepath.Join(dir, "child")
	root := ".SH ENVIRONMENT\n.so child\n.so child\n"
	body := ".TP\nVALUE\nOld body.\n"
	if e := os.WriteFile(main, []byte(root), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(child, []byte(body), 0600); e != nil {
		t.Fatal(e)
	}
	run(append(append([]string{}, command...), "--in-place", main), "", 1)
	assertBytes(main, root)
	assertBytes(child, body)
	run(append(append([]string{}, command...), "--in-place", "--include-edits", main), "", 0)
	assertBytes(main, root)
	assertBytes(child, strings.ReplaceAll(body, "Old", "New"))
	// A controlled modification of the existing provenance-labelled real fixture.
	fixture, e := os.ReadFile("testdata/environ_snippet.7")
	if e != nil {
		t.Fatal(e)
	}
	real := filepath.Join(dir, "environ.7")
	if e = os.WriteFile(real, fixture, 0600); e != nil {
		t.Fatal(e)
	}
	modified := run([]string{"edit", "--select", `section[name="DESCRIPTION"]`, "--operation", "find-replace-source", "--find", "array of pointers", "--text", "controlled array of pointers", "--format", "raw", real}, "", 0)
	if !strings.Contains(modified, "controlled array of pointers") {
		t.Fatal("real fixture selection lost")
	}
	assertBytes(real, string(fixture))
	created := filepath.Join(dir, "new.1")
	run([]string{"edit", "--create", "--title", "NEW", "--section", "1", "--operation", "append-section", "--heading", "NAME", "--text", "demo", "-o", created}, "", 0)
	b, e := os.ReadFile(created)
	if e != nil || !bytes.Contains(b, []byte("NEW")) {
		t.Fatal(e, string(b))
	}
}
