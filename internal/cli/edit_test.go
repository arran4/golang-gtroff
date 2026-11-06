package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/manlib"
)

func TestEditExplicitWritesAndDryRun(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".SH ENVIRONMENT\n.so child\n")}, "child": {Data: []byte(".TP\nVALUE\nold\n")}}
	for _, dry := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "preview"}[dry], func(t *testing.T) {
			var out, stderr bytes.Buffer
			calls := 0
			env := Environment{Out: &out, Err: &stderr, OpenRoot: func(string) (manlib.Root, io.Closer, error) { return manlib.Root{FS: fsys}, nil, nil }, CommitEdits: func(ctx context.Context, edits []atomicfile.SourceEdit, guards []atomicfile.SourceGuard) error {
				calls++
				if len(edits) != 1 || edits[0].Path != filepath.FromSlash("docs/child") || len(guards) != 2 || !strings.Contains(string(edits[0].Data), "new") {
					t.Fatalf("commit contract: %+v %+v", edits, guards)
				}
				return nil
			}}
			args := []string{"edit", "--include-edits", "--in-place", "--select", "definition", "--operation", "find-replace-source", "--find", "old", "--text", "new", "docs/main.1"}
			if dry {
				args = append([]string{"edit", "--dry-run"}, args[1:]...)
			}
			code := Run(args, env)
			if code != 0 || calls != map[bool]int{false: 1, true: 0}[dry] {
				t.Fatal(code, calls, stderr.String())
			}
			if !strings.Contains(out.String(), `"child"`) {
				t.Fatal(out.String())
			}
		})
	}
}
func TestEditCopyCreateAndFailurePolicies(t *testing.T) {
	for _, tc := range []struct {
		name         string
		extra        []string
		exit, writes int
		reader       string
	}{{"copy", []string{"-o", "out.1"}, 0, 1, ".SH A\nold\n"}, {"dry copy", []string{"-o", "out.1", "--dry-run"}, 0, 0, "old\n"}, {"invalid replacement", []string{"-o", "out.1", "--text", ".SH \"broken"}, 1, 0, "old\n"}, {"truncated input", []string{"--max-bytes", "1", "-o", "out.1"}, 3, 0, "old\n"}, {"source strict", []string{"--strict", "-o", "out.1"}, 1, 0, ".UNKNOWN\nold\n"}} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			writes := 0
			args := []string{"edit", "--operation", "replace-source", "--text", "new\n"}
			args = append(args, tc.extra...)
			code := Run(args, Environment{In: strings.NewReader(tc.reader), Out: &out, Err: &stderr, WriteEditedOutput: func(ctx context.Context, dest string, guards []atomicfile.SourceGuard, b []byte) error {
				writes++
				if len(guards) != 0 || dest != "out.1" || string(b) != "new\n" {
					t.Fatal(dest, guards, string(b))
				}
				return nil
			}})
			if code != tc.exit || writes != tc.writes {
				t.Fatal(code, writes, stderr.String())
			}
		})
	}
	var out, stderr bytes.Buffer
	opened, writes := 0, 0
	code := Run([]string{"edit", "--create", "--title", "TOOL", "--section", "1", "--operation", "append-section", "--heading", "NAME", "--text", "tool - demo", "-o", "new.1"}, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opened++; return nil, fs.ErrNotExist }, WriteEditedOutput: func(ctx context.Context, d string, g []atomicfile.SourceGuard, b []byte) error {
		writes++
		if !strings.Contains(string(b), "TOOL") || len(g) != 0 {
			t.Fatal(string(b), g)
		}
		return nil
	}})
	if code != 0 || opened != 0 || writes != 1 {
		t.Fatal(code, opened, writes, stderr.String())
	}
}
func TestEditUsageBeforeIO(t *testing.T) {
	for _, args := range [][]string{{"--operation", "delete", "--nth", "0"}, {"--operation", "delete", "--first", "0"}, {"--operation", "delete", "--range", "0:1"}, {"--operation", "delete", "--range", "2:1"}, {"--operation", "delete", "--max-nodes", "1"}, {"--operation", "delete", "--select", "section["}, {"--operation", "execute"}, {"--operation", "move-before"}, {"--create", "--in-place", "--operation", "delete"}, {"--operation", "delete", "--in-place", "-o", "out"}, {"--commands", "commands.json", "--operation", "delete"}} {
		opens := 0
		var out, stderr bytes.Buffer
		args = append([]string{"edit"}, args...)
		args = append(args, "page.1")
		code := Run(args, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opens++; return nil, fs.ErrNotExist }})
		if code != ExitUsage || opens != 0 {
			t.Fatalf("%v -> %d opens %d: %s", args, code, opens, stderr.String())
		}
	}
}
func TestEditWriterAndCommitFailure(t *testing.T) {
	failure := errors.New("write failed")
	for _, previewFails := range []bool{false, true} {
		var stderr bytes.Buffer
		var out io.Writer = &bytes.Buffer{}
		if previewFails {
			out = extractFailWriter{failure}
		}
		writes := 0
		code := Run([]string{"edit", "--operation", "replace-source", "--text", "new", "-o", "out"}, Environment{In: strings.NewReader("old\n"), Out: out, Err: &stderr, WriteEditedOutput: func(context.Context, string, []atomicfile.SourceGuard, []byte) error { writes++; return failure }})
		if code != ExitIO || writes != map[bool]int{false: 1, true: 0}[previewFails] || !strings.Contains(stderr.String(), failure.Error()) {
			t.Fatal(code, writes, stderr.String())
		}
	}
}
func TestEditCommandsSequentialAndUnchangedMalformed(t *testing.T) {
	var out, stderr bytes.Buffer
	script := `{"schemaVersion":1,"commands":[{"selector":"source[source-id=1]","operation":"append-section","heading":"NEW","text":"added"},{"selector":"section[name=NEW]","operation":"find-replace-source","find":"added","text":"changed"}]}`
	code := Run([]string{"edit", "--commands", "commands.json", "--format", "raw"}, Environment{In: strings.NewReader(".TH \"unclosed\n.SH OLD\noriginal\n"), Out: &out, Err: &stderr, Open: func(p string) (io.ReadCloser, error) {
		if p != "commands.json" {
			t.Fatal(p)
		}
		return io.NopCloser(strings.NewReader(script)), nil
	}})
	if code != 0 || !strings.HasPrefix(out.String(), ".TH \"unclosed\n") || !strings.Contains(out.String(), "changed") {
		t.Fatal(code, stderr.String(), out.String())
	}
}
