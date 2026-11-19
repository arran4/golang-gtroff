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

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/scriptapi"
)

func TestScriptCoreBuildAndUsageBeforeIO(t *testing.T) {
	for _, tc := range []struct {
		args []string
		exit int
	}{{[]string{"script", "code.tengo"}, 3}, {[]string{"script", "--help"}, 0}, {[]string{"script"}, 2}, {[]string{"script", "-", "-"}, 2}, {[]string{"script", "--max-steps", "0", "code"}, 2}, {[]string{"script", "--in-place", "code"}, 2}, {[]string{"script", "--include-edits", "code"}, 2}, {[]string{"script", "--write-target", "-", "code"}, 2}, {[]string{"script", "--allow-edits", "--in-place", "--write-target", "out", "code"}, 2}} {
		var out, stderr bytes.Buffer
		opens := 0
		code := Run(tc.args, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opens++; return nil, fs.ErrNotExist }})
		if code != tc.exit || opens != 0 {
			t.Fatal(tc.args, code, opens, stderr.String())
		}
	}
}
func TestScriptReceivesSharedSourcesAndExplicitGuards(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".SH ENVIRONMENT\n.so child\n")}, "child": {Data: []byte(".TP\nVALUE\nOld.\n")}}
	for _, dry := range []bool{false, true} {
		var out, stderr bytes.Buffer
		calls, writes := 0, 0
		environment := Environment{In: strings.NewReader("program"), Out: &out, Err: &stderr, OpenRoot: func(root string) (manlib.Root, io.Closer, error) {
			if root != "docs" {
				t.Fatal(root)
			}
			return manlib.Root{FS: fsys}, nil, nil
		}, Script: func(ctx context.Context, r scriptapi.Request, o scriptapi.Options) (*scriptapi.Result, error) {
			calls++
			if r.Name != "<stdin-script>" || string(r.Code) != "program" || len(r.Sources.Files()) != 2 || !o.AllowEdits || !o.Edit.IncludeEdits || o.MaxSteps != 100000 {
				t.Fatal(r, o)
			}
			w, e := edit.New(r.Sources, o.Edit)
			if e != nil {
				return nil, e
			}
			commands, e := edit.ParseCommands([]byte(`{"schemaVersion":1,"commands":[{"selector":"definition","operation":"find-replace-source","find":"Old","text":"New"}]}`))
			if e != nil {
				return nil, e
			}
			p, e := w.Plan(ctx, commands...)
			if e != nil {
				return nil, e
			}
			if e = w.Apply(p); e != nil {
				return nil, e
			}
			return &scriptapi.Result{Working: w, Output: []byte("done\n")}, nil
		}, CommitEdits: func(ctx context.Context, edits []atomicfile.SourceEdit, guards []atomicfile.SourceGuard) error {
			writes++
			if len(edits) != 1 || edits[0].Path != filepath.FromSlash("docs/child") || len(guards) != 2 || !strings.Contains(string(edits[0].Data), "New") {
				t.Fatal(edits, guards)
			}
			return nil
		}}
		args := []string{"script", "--allow-edits", "--include-edits", "--in-place", "-", "docs/main.1"}
		if dry {
			args = append([]string{"script", "--dry-run"}, args[1:]...)
		}
		code := Run(args, environment)
		if code != 0 || calls != 1 || writes != map[bool]int{false: 1, true: 0}[dry] || out.String() != "done\n" {
			t.Fatal(code, calls, writes, out.String(), stderr.String())
		}
	}
}
func TestScriptWriteAuthorizationAndFailureIsolation(t *testing.T) {
	for _, tc := range []struct {
		name, target, requested string
		backendError            error
		writerError             bool
		exit, writes            int
	}{
		{name: "copy", target: "out.1", requested: "out.1", writes: 1},
		{name: "not granted", requested: "out.1", exit: 1},
		{name: "wrong destination", target: "out.1", requested: "other.1", exit: 1},
		{name: "backend failure", target: "out.1", requested: "out.1", backendError: scriptapi.ErrDenied, exit: 1},
		{name: "backend resource", target: "out.1", requested: "out.1", backendError: scriptapi.ErrLimit, exit: 3},
		{name: "stream failure", target: "out.1", requested: "out.1", writerError: true, exit: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr, out bytes.Buffer
			writes := 0
			var writer io.Writer = &out
			if tc.writerError {
				writer = extractFailWriter{errors.New("stream failure")}
			}
			args := []string{"script"}
			if tc.target != "" {
				args = append(args, "--write-target", tc.target)
			}
			args = append(args, "-")
			code := Run(args, Environment{In: strings.NewReader("program"), Out: writer, Err: &stderr, Script: func(ctx context.Context, r scriptapi.Request, o scriptapi.Options) (*scriptapi.Result, error) {
				w, e := edit.Create(ctx, "new.1", man.Metadata{}, o.Edit)
				if e != nil {
					return nil, e
				}
				data, _ := w.Data("new.1")
				return &scriptapi.Result{Created: true, Working: w, Output: []byte("result\n"), Writes: []scriptapi.WriteRequest{{Source: "new.1", Destination: tc.requested, Data: data}}}, tc.backendError
			}, WriteEditedOutput: func(ctx context.Context, dest string, guards []atomicfile.SourceGuard, data []byte) error {
				writes++
				if dest != "out.1" || len(guards) != 0 {
					t.Fatal(dest, guards)
				}
				return nil
			}})
			if code != tc.exit || writes != tc.writes {
				t.Fatal(code, writes, stderr.String())
			}
			if tc.backendError != nil && out.Len() != 0 {
				t.Fatal("failed backend published partial output")
			}
		})
	}
}
func TestScriptInputFailureNeverExecutes(t *testing.T) {
	failure := errors.New("script close failure")
	called := 0
	var stderr bytes.Buffer
	code := Run([]string{"script", "program.tengo"}, Environment{Err: &stderr, Script: func(context.Context, scriptapi.Request, scriptapi.Options) (*scriptapi.Result, error) {
		called++
		return &scriptapi.Result{}, nil
	}, Open: func(string) (io.ReadCloser, error) {
		return scriptCloseReader{Reader: strings.NewReader("program"), err: failure}, nil
	}})
	if code != ExitIO || called != 0 || !strings.Contains(stderr.String(), failure.Error()) {
		t.Fatal(code, called, stderr.String())
	}
	stderr.Reset()
	code = Run([]string{"script", "--max-script-bytes", "1", "-"}, Environment{In: strings.NewReader("too long"), Err: &stderr, Script: func(context.Context, scriptapi.Request, scriptapi.Options) (*scriptapi.Result, error) {
		called++
		return &scriptapi.Result{}, nil
	}})
	if code != ExitIO || called != 0 {
		t.Fatal(code, called, stderr.String())
	}
}

type scriptCloseReader struct {
	io.Reader
	err error
}

func (r scriptCloseReader) Close() error { return r.err }
