//go:build !js

package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
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
	// Real persistent process, original guards across explicit commits, exact
	// untouched bytes and absence of automatic writes after ordinary cells.
	reset()
	replArgs := []string{"script", "--repl", "--prompt=false", "--allow-edits", "--include-edits", "--in-place", main}
	transformCell := `doc.edit({selector:"definition",operation:"find-replace-source",find:"Old",text:"New"})` + "\n"
	run(replArgs, transformCell+":quit\n", 0)
	check(child, originalChild)
	run(replArgs, transformCell+":commit\n"+`doc.edit({selector:"definition",operation:"find-replace-source",find:"New",text:"Final"})`+"\n:commit\n:quit\n", 0)
	check(child, strings.ReplaceAll(originalChild, "Old", "Final"))
	reset()
	run(append([]string{"script", "--repl", "--prompt=false", "--dry-run"}, replArgs[3:]...), transformCell+":commit\n:quit\n", 0)
	check(child, originalChild)
	out, _ = run([]string{"script", "--repl", "--prompt=false", main}, "n:=1;fn:=func(){n++;return n}\ndoc.emit_json(fn())\nn=9;bad:=1/0\ndoc.emit_json(n)\n:quit\n", 1)
	if out != "2\n2\n" {
		t.Fatal("REPL globals/rollback", out)
	}
	out, _ = run([]string{"script", "--repl", "--prompt=false", real}, `doc.emit(doc.extract("section[name=DESCRIPTION] paragraph[text*=\"array of pointers\"]","text"))`+"\n:quit\n", 0)
	if !strings.Contains(out, "array of pointers") {
		t.Fatal("real REPL fixture", out)
	}
	check(real, string(fixture))
	run([]string{"script", "--repl", "-"}, "", 2)

	sessionExample, e := os.ReadFile(filepath.Join("..", "examples", "session.tengo"))
	if e != nil {
		t.Fatal(e)
	}
	out, _ = run([]string{"script", "--repl", "--prompt=false", main}, string(sessionExample), 0)
	if !strings.Contains(out, "TOOL_HOME") {
		t.Fatal("documented session example", out)
	}

	// Exercise actual disk drift between cells, inspect the retained candidate,
	// restore the authorized original, then retry the same failed commit batch.
	reset()
	processCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	live := exec.CommandContext(processCtx, binary, replArgs...)
	stdinPipe, e := live.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	stdoutPipe, e := live.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	var liveErrors bytes.Buffer
	live.Stderr = &liveErrors
	if e = live.Start(); e != nil {
		t.Fatal(e)
	}
	if _, e = io.WriteString(stdinPipe, transformCell+`doc.emit("ready\n")`+"\n"); e != nil {
		t.Fatal(e)
	}
	liveOut := bufio.NewReader(stdoutPipe)
	if line, e := liveOut.ReadString('\n'); e != nil || line != "ready\n" {
		t.Fatal(line, e)
	}
	external := strings.ReplaceAll(originalChild, "Old", "EXT")
	write(child, external)
	if _, e = io.WriteString(stdinPipe, ":commit\n:pending\n"); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 5; i++ {
		if _, e = liveOut.ReadString('\n'); e != nil {
			t.Fatal(e, liveErrors.String())
		}
	}
	check(child, external)
	write(child, originalChild)
	if _, e = io.WriteString(stdinPipe, ":retry\n:quit\n"); e != nil {
		t.Fatal(e)
	}
	if e = stdinPipe.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = io.ReadAll(liveOut); e != nil {
		t.Fatal(e)
	}
	exitErr := live.Wait()
	if exit, ok := exitErr.(*exec.ExitError); !ok || exit.ExitCode() != 3 {
		t.Fatal(exitErr, liveErrors.String())
	}
	if !strings.Contains(liveErrors.String(), "changed") && !strings.Contains(liveErrors.String(), "stale") {
		t.Fatal(liveErrors.String())
	}
	check(child, strings.ReplaceAll(originalChild, "Old", "New"))
	check(main, originalRoot)

}
