package script_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/scriptapi"
)

func replEnvironment(input string, out, stderr io.Writer) cli.Environment {
	fsys := fstest.MapFS{"page.1": {Data: []byte(".SH ENVIRONMENT\n.so child\n")}, "child": {Data: []byte(".\\\" keep\r\n.UNKNOWN retained\r\n.TP\r\nVALUE\r\nOld.\r\n\xff\r\n")}}
	return cli.Environment{In: strings.NewReader(input), Out: out, Err: stderr,
		OpenRoot: func(string) (manlib.Root, io.Closer, error) { return manlib.Root{FS: fsys}, nil, nil },
		ScriptSession: func(ctx context.Context, r scriptapi.Request, o scriptapi.SessionOptions) (scriptapi.Session, error) {
			return script.NewSession(ctx, r, o)
		},
	}
}
func TestREPLNativeCellsAndInputRecovery(t *testing.T) {
	code := `n := 0
f := func() {
 n++
 return n
}
doc.emit_json([f(),len(doc.query("definition"))])
n = 8; doc.emit("partial"); fail := 1/0
doc.emit_json(n)
incomplete := [
:cancel
doc.emit("ok")
:quit
`
	var out, stderr bytes.Buffer
	status := cli.Run([]string{"script", "--repl", "--prompt=false", "page.1"}, replEnvironment(code, &out, &stderr))
	if status != 1 || out.String() != "[1,1]\n1\nok" || !strings.Contains(stderr.String(), "divide by zero") {
		t.Fatal(status, out.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "gtroff>") {
		t.Fatal("prompt disabled")
	}
}
func TestREPLGuardedCommitRetryAndBaseline(t *testing.T) {
	code := `doc.edit({selector:"definition",operation:"find-replace-source",find:"Old",text:"New"})
:commit
:pending
blocked := 1
:retry
doc.edit({selector:"definition",operation:"find-replace-source",find:"New",text:"Final"})
:commit
:quit
`
	var out, stderr bytes.Buffer
	env := replEnvironment(code, &out, &stderr)
	calls := 0
	oldDigest := ""
	firstAfter := []byte(nil)
	env.CommitEdits = func(_ context.Context, updates []atomicfile.SourceEdit, guards []atomicfile.SourceGuard) error {
		calls++
		if len(updates) != 1 || len(guards) != 2 || !strings.HasSuffix(updates[0].Path, "child") {
			t.Fatal(updates, guards)
		}
		var digest string
		for _, g := range guards {
			if strings.HasSuffix(g.Path, "child") {
				digest = g.Digest
			}
		}
		if calls == 1 {
			oldDigest = digest
			firstAfter = append([]byte(nil), updates[0].Data...)
			return errors.New("rename failed; recovery: owned-stage")
		}
		if calls == 2 {
			if digest != oldDigest || !bytes.Equal(firstAfter, updates[0].Data) {
				t.Fatal("retry changed original batch")
			}
		}
		if calls == 3 {
			if digest == oldDigest || !strings.Contains(string(updates[0].Data), "Final") {
				t.Fatal("baseline not acknowledged")
			}
		}
		return nil
	}
	status := cli.Run([]string{"script", "--repl", "--prompt=false", "--allow-edits", "--include-edits", "--in-place", "page.1"}, env)
	if status != 3 || calls != 3 || !strings.Contains(stderr.String(), "owned-stage") || !strings.Contains(stderr.String(), "unpublished") || !strings.Contains(out.String(), "bytes [") {
		t.Fatal(status, calls, out.String(), stderr.String())
	}
}

type retryOutput struct {
	bytes.Buffer
	failed bool
}

func (w *retryOutput) Write(p []byte) (int, error) {
	if !w.failed {
		w.failed = true
		if len(p) > 0 {
			w.Buffer.Write(p[:1])
			return 1, errors.New("stream prefix failed")
		}
	}
	return w.Buffer.Write(p)
}
func TestREPLPublicationRetryDiscardAndDryRun(t *testing.T) {
	var out retryOutput
	var stderr bytes.Buffer
	code := "n:=1;doc.emit(\"A\\n\")\nn=8\n:retry\ndoc.emit_json(n)\n:quit\n"
	status := cli.Run([]string{"script", "--repl", "--prompt=false"}, replEnvironment(code, &out, &stderr))
	if status != 3 || out.String() != "AA\n1\n" || !strings.Contains(stderr.String(), "unpublished") {
		t.Fatal(status, out.String(), stderr.String())
	}
	// A failed copy request is retained. Explicit discard permits later cells,
	// while dry-run suppresses authorized copy/commit callbacks altogether.
	for _, dry := range []bool{false, true} {
		var output, diagnostics bytes.Buffer
		code := `doc.write("child","out.1")
:discard
doc.emit("continued")
:commit
:quit
`
		env := replEnvironment(code, &output, &diagnostics)
		writes, commits := 0, 0
		env.WriteEditedOutput = func(context.Context, string, []atomicfile.SourceGuard, []byte) error {
			writes++
			return errors.New("copy failed")
		}
		env.CommitEdits = func(context.Context, []atomicfile.SourceEdit, []atomicfile.SourceGuard) error { commits++; return nil }
		args := []string{"script", "--repl", "--prompt=false", "--write-target", "out.1"}
		if dry {
			args = append(args, "--dry-run")
		}
		args = append(args, "page.1")
		status := cli.Run(args, env)
		if status != 1 && !(!dry && status == 3) || output.String() != "continued" || commits != 0 || writes != map[bool]int{true: 0, false: 1}[dry] {
			t.Fatal(dry, status, writes, commits, output.String(), diagnostics.String())
		}
	}
}
func TestREPLBoundsPromptsAndEOF(t *testing.T) {
	for _, tc := range []struct {
		input    string
		args     []string
		want     int
		contains string
	}{
		{"doc.emit(\"ok\")", nil, 0, "gtroff> "},
		{"f:=func(){\n", nil, 1, "incomplete"},
		{"x:=1\nx=2\n", []string{"--max-cells", "1"}, 3, "limit"},
		{"doc.emit(\"over\")\n", []string{"--max-session-bytes", "3"}, 3, "limit"},
		{":commit\n:quit\n", nil, 1, "authorization"},
		{":help\n:quit\n", nil, 0, "native cells"},
		{"/* multi\nline */\ndoc.emit(\"ok\")\n:quit\n", nil, 0, "gtroff> "},
	} {
		var out, stderr bytes.Buffer
		args := append([]string{"script", "--repl"}, tc.args...)
		status := cli.Run(args, replEnvironment(tc.input, &out, &stderr))
		if status != tc.want || !strings.Contains(stderr.String(), tc.contains) {
			t.Fatal(tc, status, out.String(), stderr.String())
		}
	}
}

type brokenCellReader struct {
	data    string
	invalid bool
}

func (r brokenCellReader) Read(p []byte) (int, error) {
	if r.invalid {
		return len(p) + 1, nil
	}
	return copy(p, r.data), errors.New("cell input failed")
}
func TestREPLInputFailureWithholdsPartialCell(t *testing.T) {
	for _, reader := range []io.Reader{brokenCellReader{data: `doc.emit("must be withheld")`}, brokenCellReader{invalid: true}} {
		var out, stderr bytes.Buffer
		env := replEnvironment("", &out, &stderr)
		env.In = reader
		status := cli.Run([]string{"script", "--repl", "--prompt=false"}, env)
		if status != 3 || out.Len() != 0 || stderr.Len() == 0 {
			t.Fatal(status, out.String(), stderr.String())
		}
	}
}
