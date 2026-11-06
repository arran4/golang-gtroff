//go:build !js

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSelfContainedScriptExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, b)
	}
	main, child, program := filepath.Join(dir, "main.1"), filepath.Join(dir, "child"), filepath.Join(dir, "program.tengo")
	originalRoot := ".SH ENVIRONMENT\n.so child\n"
	originalChild := ".\\\" comment\r\n.UNKNOWN retained\r\n.TP\r\n.B TOOL_HOME\r\nOld directory.\r\n\xff\r\n"
	write := func(name, data string) {
		t.Helper()
		if e := os.WriteFile(name, []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
	check := func(name, want string) {
		t.Helper()
		b, e := os.ReadFile(name)
		if e != nil || string(b) != want {
			t.Fatalf("%s: %q want %q %v", name, b, want, e)
		}
	}
	run := func(args []string, input string, want int) (string, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = dir
		command.Stdin = strings.NewReader(input)
		var out, stderr bytes.Buffer
		command.Stdout = &out
		command.Stderr = &stderr
		e := command.Run()
		code := 0
		if e != nil {
			if exit, ok := e.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		if code != want {
			t.Fatalf("%v: %d want %d: %s %s", args, code, want, out.String(), stderr.String())
		}
		return out.String(), stderr.String()
	}
	reset := func() { write(main, originalRoot); write(child, originalChild) }
	reset()
	write(program, `for match in doc.query("definition"){doc.emit_json({term:match.term,file:match.source})}`)
	out, _ := run([]string{"script", program, main}, "", 0)
	if out != "{\"file\":\"child\",\"term\":\"TOOL_HOME\"}\n" {
		t.Fatal(out)
	}
	check(main, originalRoot)
	check(child, originalChild)
	out, _ = run([]string{"script", "--no-includes", program, main}, "", 0)
	if out != "" {
		t.Fatal(out)
	}
	out, _ = run([]string{"script", "-", main}, `doc.emit(doc.extract("definition","raw"))`, 0)
	if !strings.Contains(out, "Old directory.\r\n") {
		t.Fatal(out)
	}
	transform := `doc.edit({selector:"definition",operation:"find-replace-source",find:"Old",text:"New"});doc.emit(doc.preview("diff"))`
	write(program, transform)
	run([]string{"script", program, main}, "", 1)
	run([]string{"script", "--allow-edits", program, main}, "", 1)
	run([]string{"script", "--allow-edits", "--include-edits", program, main}, "", 0)
	check(main, originalRoot)
	check(child, originalChild)
	run([]string{"script", "--allow-edits", "--include-edits", "--in-place", "--dry-run", program, main}, "", 0)
	check(child, originalChild)
	run([]string{"script", "--allow-edits", "--include-edits", "--in-place", program, main}, "", 0)
	check(main, originalRoot)
	check(child, strings.ReplaceAll(originalChild, "Old", "New"))
	reset()
	target := filepath.Join(dir, "out.1")
	write(program, `doc.write("child",`+strconv.Quote(target)+`)`)
	run([]string{"script", program, main}, "", 1)
	run([]string{"script", "--write-target", target, "--dry-run", program, main}, "", 0)
	if _, e := os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("dry run wrote", e)
	}
	run([]string{"script", "--write-target", target, program, main}, "", 0)
	check(target, originalChild)
	check(main, originalRoot)
	check(child, originalChild)
	write(program, `doc.write("child",`+strconv.Quote(child)+`)`)
	run([]string{"script", "--write-target", child, program, main}, "", 3)
	check(child, originalChild)
	write(program, `doc.emit("must stay staged");doc.write(`+strconv.Quote(target)+`);f:=1;f()`)
	out, _ = run([]string{"script", "--write-target", target, program, main}, "", 1)
	if out != "" {
		t.Fatal("failed VM published output", out)
	}
	check(target, originalChild)
	write(program, `doc.write(`+strconv.Quote(program)+`)`)
	run([]string{"script", "--write-target", program, program, main}, "", 3)
	check(program, `doc.write(`+strconv.Quote(program)+`)`)
	run([]string{"script", "--max-steps", "10", "-"}, `for {}`, 3)
	run([]string{"script", "-", "-"}, "", 2)
	run([]string{"script", "-"}, `module:=import("../source")`, 1)
	// The optional binary also serves ordinary core commands without a helper.
	out, _ = run([]string{"query", "--format", "text", "definition", main}, "", 0)
	if !strings.Contains(out, "TOOL_HOME") {
		t.Fatal(out)
	}
	fixture, e := os.ReadFile(filepath.Join("..", "..", "testdata", "environ_snippet.7"))
	if e != nil {
		t.Fatal(e)
	}
	real := filepath.Join(dir, "environ.7")
	write(real, string(fixture))
	write(program, `doc.emit(doc.extract("section[name=DESCRIPTION] paragraph[text*=\"array of pointers\"]","text"))`)
	out, _ = run([]string{"script", program, real}, "", 0)
	if !strings.Contains(out, "array of pointers") {
		t.Fatal("real fixture query lost", out)
	}
	check(real, string(fixture))
	for _, example := range []string{"inspect", "functional", "transform"} {
		code, e := os.ReadFile(filepath.Join("..", "examples", example+".tengo"))
		if e != nil {
			t.Fatal(e)
		}
		write(program, string(code))
		page, e := os.ReadFile(filepath.Join("..", "examples", "environment.1"))
		if e != nil {
			t.Fatal(e)
		}
		write(main, string(page))
		args := []string{"script"}
		if example == "transform" {
			args = append(args, "--allow-edits", "--in-place", "--dry-run")
		}
		args = append(args, program, main)
		out, _ = run(args, "", 0)
		if out == "" {
			t.Fatal("documented example produced no output", example)
		}
		check(main, string(page))
	}
	created := filepath.Join(dir, "new.1")
	code, e := os.ReadFile(filepath.Join("..", "examples", "create.tengo"))
	if e != nil {
		t.Fatal(e)
	}
	run([]string{"script", "--allow-edits", "--write-target", "new.1", "-"}, string(code), 0)
	b, e := os.ReadFile(created)
	if e != nil || !bytes.Contains(b, []byte("NEW")) {
		t.Fatal(e, string(b))
	}
}
