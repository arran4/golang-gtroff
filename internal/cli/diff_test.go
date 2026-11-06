package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
)

func TestDiffSharedSourceBudgetsAndEarlyFailure(t *testing.T) {
	for _, args := range [][]string{{"diff", "--max-files", "1", "before.1", "after.1"}, {"diff", "missing.1", "after.1"}} {
		fsys := fstest.MapFS{"before.1": {Data: []byte(".TH TOOL 1\n")}, "after.1": {Data: []byte(".TH TOOL 1\n")}}
		opens := 0
		var out, stderr bytes.Buffer
		code := Run(args, Environment{Out: &out, Err: &stderr, OpenRoot: func(string) (manlib.Root, io.Closer, error) { opens++; return manlib.Root{FS: fsys}, nil, nil }})
		if code != ExitIO || opens != 1 || strings.Contains(out.String(), "removed source") {
			t.Fatal("second root opened after fatal/shared cap or invented removals", code, opens, out.String(), stderr.String())
		}
	}
}
func TestDiffUsageAndCancellationBeforeIO(t *testing.T) {
	for _, args := range [][]string{{"diff", "--format", "bogus", "a", "b"}, {"diff", "--max-work", "0", "a", "b"}, {"diff", "-", "-"}, {"diff", "a"}, {"diff", "", "b"}} {
		opens := 0
		var out, stderr bytes.Buffer
		code := Run(args, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opens++; return nil, fs.ErrNotExist }})
		if code != ExitUsage || opens != 0 {
			t.Fatalf("%v -> %d opens %d", args, code, opens)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, stderr bytes.Buffer
	opened := false
	code := Run([]string{"diff", "a", "b"}, Environment{Context: ctx, Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opened = true; return nil, fs.ErrNotExist }})
	if code != ExitIO || opened {
		t.Fatal("canceled diff did input I/O")
	}
}
func TestDiffPropagatesWriterFailure(t *testing.T) {
	sentinel := errors.New("diff writer failed")
	var stderr bytes.Buffer
	code := Run([]string{"diff", "-", "after.1"}, Environment{In: strings.NewReader(".TH TOOL 1\n"), Out: diffFailWriter{sentinel}, Err: &stderr, Open: func(string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(".TH TOOL 1\n")), nil }})
	if code != ExitIO || !strings.Contains(stderr.String(), sentinel.Error()) {
		t.Fatal("writer error lost", code, stderr.String())
	}
}

type diffFailWriter struct{ err error }

func (w diffFailWriter) Write(data []byte) (int, error) { return 0, w.err }
