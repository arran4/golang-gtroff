package app

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/viewer"
)

type closer struct {
	io.Reader
	closed *bool
}

func (c closer) Close() error { *c.closed = true; return nil }
func TestCLIInputClosesBeforeDisplay(t *testing.T) {
	closed, shown := false, false
	var diags bytes.Buffer
	code := Run([]string{"--section", "OPTIONS", "source.1"}, Environment{Err: &diags, Open: func(path string) (io.ReadCloser, error) {
		if path != "source.1" {
			t.Fatal(path)
		}
		return closer{strings.NewReader(".TH TOOL 1\n.SH NAME\nunselected\n.SH OPTIONS\nselected\n"), &closed}, nil
	}, Show: func(s *viewer.Session) error {
		shown = true
		if !closed {
			t.Fatal("input still open")
		}
		if s.Controller.State().Pages != 1 {
			t.Fatal("selection")
		}
		_, e := s.Frame()
		return e
	}})
	if code != 0 || !shown || !closed || diags.Len() != 0 {
		t.Fatal(code, shown, closed, diags.String())
	}
}
func TestCLIHelpVersionAndFailuresBeforeIO(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}} {
		var out bytes.Buffer
		code := Run(args, Environment{Out: &out, BackendError: errors.New("no backend"), Open: func(string) (io.ReadCloser, error) { t.Fatal("opened on help/version"); return nil, nil }})
		if code != 0 || out.Len() == 0 {
			t.Fatal(code)
		}
	}
	for _, args := range [][]string{{}, {"--window-width", "2147483647", "file"}, {"--font-size", "NaN", "file"}, {"--page-height", "-1", "file"}} {
		if code := Run(args, Environment{Open: func(string) (io.ReadCloser, error) { t.Fatal("opened invalid input"); return nil, nil }}); code != 2 {
			t.Fatal(args, code)
		}
	}
}
func TestCLIDiagnosticsAndMissingDisplay(t *testing.T) {
	var diags bytes.Buffer
	shown := false
	code := Run([]string{"-"}, Environment{In: strings.NewReader(".UNKNOWN visible\n"), Err: &diags, Show: func(s *viewer.Session) error {
		shown = true
		if len(s.Diagnostics()) == 0 {
			t.Fatal("no diagnostics")
		}
		return nil
	}})
	if code != 0 || !shown || !strings.Contains(diags.String(), "warning") {
		t.Fatal(code, diags.String())
	}
	shown = false
	code = Run([]string{"--strict", "-"}, Environment{In: strings.NewReader(".UNKNOWN visible\n"), Show: func(*viewer.Session) error { shown = true; return nil }})
	if code != 1 || shown {
		t.Fatal("strict opened", code)
	}
	code = Run([]string{"-"}, Environment{BackendError: errors.New("unsupported platform"), Show: func(*viewer.Session) error { t.Fatal("display called"); return nil }})
	if code != 3 {
		t.Fatal(code)
	}
}

func TestMissingSelectionDoesNotOpenWindow(t *testing.T) {
	shown := false
	code := Run([]string{"--section", "ABSENT", "-"}, Environment{In: strings.NewReader(".SH NAME\ntext\n"), Show: func(*viewer.Session) error { shown = true; return nil }})
	if code != 2 || shown {
		t.Fatal(code, shown)
	}
}
