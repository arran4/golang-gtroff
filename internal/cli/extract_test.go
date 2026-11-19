package cli

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
)

func TestExtractAtomicOutputReceivesEverySource(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".TH TOOL 1\n.SH ENVIRONMENT\n.so child.roff\n")}, "child.roff": {Data: []byte(".TP\n.B HOME\nDirectory.\n")}}
	var out, stderr bytes.Buffer
	writes := 0
	env := Environment{Out: &out, Err: &stderr, OpenRoot: func(root string) (manlib.Root, io.Closer, error) {
		if root != "docs" {
			t.Fatal("unexpected root", root)
		}
		return manlib.Root{FS: fsys}, nil, nil
	}, WriteSourceOutput: func(dest string, sources []string, data []byte) error {
		writes++
		if dest != "out.csv" || len(sources) != 2 || sources[0] != filepath.FromSlash("docs/main.1") || sources[1] != filepath.FromSlash("docs/child.roff") || !strings.Contains(string(data), "HOME") {
			t.Fatalf("output guard contract %s %v %s", dest, sources, data)
		}
		return nil
	}}
	if code := Run([]string{"extract", "--format", "csv", "-o", "out.csv", "definition", "docs/main.1"}, env); code != 0 || writes != 1 || out.Len() != 0 {
		t.Fatal("explicit output failed", code, writes, stderr.String())
	}
	writes = 0
	code := Run([]string{"extract", "--strict", "-o", "out.csv", "definition", "docs/main.1"}, env)
	if code != ExitInvalid || writes != 0 {
		t.Fatal("strict wrote destination", code, stderr.String())
	}
	writes = 0
	code = Run([]string{"extract", "--format", "csv", "--max-output-bytes", "1", "-o", "out.csv", "definition", "docs/main.1"}, env)
	if code != ExitIO || writes != 0 {
		t.Fatal("partial serialization wrote destination", code, stderr.String())
	}
}
func TestExtractOptionsRejectedBeforeSourceIO(t *testing.T) {
	for _, args := range [][]string{{"extract", "--fields", "bogus", "--format", "csv", "definition", "input.1"}, {"extract", "--normalize", "--format", "raw", "definition", "input.1"}, {"extract", "--view", "syntax", "--format", "html", "line", "input.1"}, {"extract", "section[", "input.1"}, {"extract", "--max-records", "0", "definition", "input.1"}} {
		opens := 0
		var out, stderr bytes.Buffer
		code := Run(args, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opens++; return nil, fs.ErrNotExist }})
		if code != ExitUsage || opens != 0 {
			t.Fatalf("%v -> %d opens %d", args, code, opens)
		}
	}
}
func TestExtractWriterFailure(t *testing.T) {
	sentinel := errors.New("write failed")
	var stderr bytes.Buffer
	code := Run([]string{"extract", "--format", "text", "paragraph"}, Environment{In: strings.NewReader("Text.\n"), Out: extractFailWriter{sentinel}, Err: &stderr})
	if code != ExitIO || !strings.Contains(stderr.String(), sentinel.Error()) {
		t.Fatal("writer error lost", code, stderr.String())
	}
}

type extractFailWriter struct{ err error }

func (w extractFailWriter) Write(data []byte) (int, error) { return 0, w.err }

type extractInterruptedReader struct{ data []byte }

func (r *extractInterruptedReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, errors.New("interrupted input")
}
func TestExtractInputFailureRetainsReportButDoesNotWriteFile(t *testing.T) {
	for _, destination := range []string{"-", "out.json"} {
		var out, stderr bytes.Buffer
		writes := 0
		code := Run([]string{"extract", "--format", "json", "-o", destination, "definition"}, Environment{In: &extractInterruptedReader{[]byte(".TP\n.B HOME\nDirectory.\n")}, Out: &out, Err: &stderr, WriteSourceOutput: func(string, []string, []byte) error { writes++; return nil }})
		if code != ExitIO || writes != 0 || !strings.Contains(stderr.String(), "interrupted input") {
			t.Fatal("input failure handling", code, writes, stderr.String())
		}
		if destination == "-" && !strings.Contains(out.String(), "inputError") {
			t.Fatal("input failure report lost")
		}
		if destination != "-" && out.Len() != 0 {
			t.Fatal("named failure leaked content to stdout")
		}
	}
}
