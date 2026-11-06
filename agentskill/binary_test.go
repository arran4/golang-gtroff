//go:build !js

package agentskill_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/agentskill"
)

func TestSkillsExecutable(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gtroff")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "../cmd/gtroff")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(b))
	}
	run := func(args []string, want int) []byte {
		t.Helper()
		c := exec.Command(binary, args...)
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
			t.Fatal(args, code, stderr.String(), out.String())
		}
		return out.Bytes()
	}
	b := run([]string{"skills", "list", "--format", "json"}, 0)
	var listed struct {
		SchemaVersion int
		Skills        []agentskill.Info
	}
	if e := json.Unmarshal(b, &listed); e != nil || listed.SchemaVersion != 1 || len(listed.Skills) != 1 {
		t.Fatal(e, string(b))
	}
	if !bytes.Contains(run([]string{"skills", "show", agentskill.Name}, 0), []byte("name: "+agentskill.Name)) {
		t.Fatal("missing entrypoint")
	}
	target := filepath.Join(dir, "chosen-parent")
	b = run([]string{"skills", "export", "--dir", target, "--format", "json", agentskill.Name}, 0)
	var receipt agentskill.Receipt
	if e := json.Unmarshal(b, &receipt); e != nil {
		t.Fatal(e)
	}
	bundle, e := agentskill.Lookup(agentskill.Name)
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range bundle.Files() {
		actual, e := os.ReadFile(filepath.Join(target, agentskill.Name, filepath.FromSlash(file.Path)))
		if e != nil || !bytes.Equal(actual, file.Data) {
			t.Fatal(file.Path, e)
		}
	}
	custom := filepath.Join(target, agentskill.Name, "SKILL.md")
	if e := os.WriteFile(custom, []byte("local customization"), 0600); e != nil {
		t.Fatal(e)
	}
	run([]string{"skills", "export", "--dir", target, agentskill.Name}, 1)
	saved, e := os.ReadFile(custom)
	if e != nil || string(saved) != "local customization" {
		t.Fatal("overwrote local skill", e)
	}
	traversal := filepath.Join(dir, "must-not-exist")
	run([]string{"skills", "export", "--dir", traversal, "../escape"}, 2)
	if _, e := os.Stat(traversal); !os.IsNotExist(e) {
		t.Fatal("created traversal destination", e)
	}
	run([]string{"skills", "export", agentskill.Name}, 2)
	for _, leaf := range []string{"list", "show", "export"} {
		if !strings.Contains(string(run([]string{"skills", leaf, "--help"}, 0)), "gtroff skills "+leaf) {
			t.Fatal(leaf)
		}
	}
}
