package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func TestGeneratedManPages(t *testing.T) {
	for _, command := range append([]string{""}, DocumentationCommands()...) {
		t.Run(command, func(t *testing.T) {
			args := []string{"man"}
			if command != "" {
				args = append(args, command)
			}
			var out, stderr bytes.Buffer
			if code := Run(args, Environment{Out: &out, Err: &stderr}); code != 0 || stderr.Len() != 0 {
				t.Fatalf("man exit %d: %s", code, stderr.String())
			}
			tree, err := groff.ParseAST(bytes.NewReader(out.Bytes()), groff.WithPackage("man"), groff.WithStrict(true))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := man.Build(tree, man.Options{Strict: true})
			if err != nil {
				t.Fatal(err)
			}
			text := doc.Text(1, man.TextOptions{})
			if !strings.Contains(text, "Usage: gtroff") || !strings.Contains(text, "SEE ALSO") {
				t.Fatalf("man page omitted help: %s", text)
			}
		})
	}
}
func TestCompletionShells(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			program := shell
			if shell == "powershell" {
				program = "pwsh"
			}
			path, err := exec.LookPath(program)
			if err != nil {
				t.Skip("shell not installed; completion has TXTAR coverage")
			}
			var out bytes.Buffer
			if code := Run([]string{"completion", shell}, Environment{Out: &out}); code != 0 {
				t.Fatal(code)
			}
			dir := t.TempDir()
			file := filepath.Join(dir, "_gtroff")
			if shell == "powershell" {
				file = filepath.Join(dir, "gtroff.ps1")
				// Native completion requires an application resolvable as gtroff.
				// It is never executed; use this test's existing executable.
				exe, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				name := "gtroff"
				if runtime.GOOS == "windows" {
					name += ".exe"
				}
				data, err := os.ReadFile(exe)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(file, out.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			var args []string
			switch shell {
			case "bash":
				args = []string{"-c", `source "$GTROFF_COMPLETION_FILE"; COMP_WORDS=(gtroff re); COMP_CWORD=1; _gtroff; printf '%s\n' "${COMPREPLY[@]}"; COMP_WORDS=(gtroff render --for); COMP_CWORD=2; _gtroff; printf '%s\n' "${COMPREPLY[@]}"`}
			case "zsh":
				args = []string{"-c", `compadd() { print -r -- "$@"; }; _files() { print FILES; }; fpath=("$GTROFF_COMPLETION_DIR" $fpath); autoload -U _gtroff; CURRENT=2; words=(gtroff re); PREFIX=re; _gtroff; CURRENT=3; words=(gtroff render --for); PREFIX=--for; _gtroff`}
			case "fish":
				args = []string{"-c", `source "$GTROFF_COMPLETION_FILE"; complete -C 'gtroff re'; complete -C 'gtroff render --for'`}
			case "powershell":
				args = []string{"-NoProfile", "-NonInteractive", "-Command", `$tokens=$null; $errors=$null; [System.Management.Automation.Language.Parser]::ParseFile($env:GTROFF_COMPLETION_FILE,[ref]$tokens,[ref]$errors) > $null; if ($errors.Count) { throw $errors }; . $env:GTROFF_COMPLETION_FILE; (TabExpansion2 'gtroff re' 9).CompletionMatches.CompletionText; (TabExpansion2 'gtroff render --for' 19).CompletionMatches.CompletionText`}
			}
			cmd := exec.Command(path, args...)
			cmd.Env = append(os.Environ(), "GTROFF_COMPLETION_FILE="+file, "GTROFF_COMPLETION_DIR="+dir)
			cmd.Env = append(cmd.Env, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			result, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(result), "render") || !strings.Contains(string(result), "--format") {
				t.Fatalf("shell completion: %v: %s", err, result)
			}
		})
	}
}
