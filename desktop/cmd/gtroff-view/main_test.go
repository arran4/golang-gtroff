package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecutableInfoDoesNotOpenDisplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "gtroff-view")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", path, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(out))
	}
	for _, arg := range []string{"--help", "--version"} {
		cmd := exec.CommandContext(ctx, path, arg)
		cmd.Env = append(os.Environ(), "DISPLAY=")
		out, e := cmd.CombinedOutput()
		if e != nil || len(out) == 0 {
			t.Fatal(arg, e, string(out))
		}
	}
	if runtime.GOOS == "linux" {
		cmd := exec.CommandContext(ctx, path, "-")
		cmd.Env = append(os.Environ(), "DISPLAY=")
		cmd.Stdin = strings.NewReader(".SH NAME\ntext\n")
		var out bytes.Buffer
		cmd.Stderr = &out
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 3 || !strings.Contains(out.String(), "open Shiny window") {
			t.Fatal("missing display did not produce clear backend error", err, out.String())
		}
	}
}
