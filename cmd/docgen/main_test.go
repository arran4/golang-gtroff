package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDocumentationGeneratorProcess(t *testing.T) {
	work := t.TempDir()
	binary := filepath.Join(t.TempDir(), "docgen")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build docgen: %v\n%s", err, out)
	}
	run := func(wantSuccess bool, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = work
		out, err := cmd.CombinedOutput()
		if wantSuccess != (err == nil) {
			t.Fatalf("docgen %v: %v\n%s", args, err, out)
		}
	}
	run(false) // A check may not create the missing documentation tree.
	if entries, err := os.ReadDir(work); err != nil || len(entries) != 0 {
		t.Fatal("check created files", entries, err)
	}
	run(true, "--write")
	run(true)
	first := filepath.Join(work, "docs", "generated", "man", "gtroff.1")
	last := filepath.Join(work, "docs", "generated", "completions", "gtroff.ps1")
	if err := os.WriteFile(first, []byte("original first output"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(last); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(last, 0700); err != nil {
		t.Fatal(err)
	}
	run(false, "--write")
	data, err := os.ReadFile(first)
	if err != nil || string(data) != "original first output" {
		t.Fatalf("late destination failure partially regenerated docs: %q %v", data, err)
	}
}
