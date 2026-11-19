package main

import (
	"bytes"
	"errors"
	groff "github.com/arran4/golang-gtroff"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_ProcessReader(t *testing.T) {
	input := ".TH HELLO 1\n.SH NAME\nhello - greeting\n"
	var out, diagnostics bytes.Buffer
	err := processReaderTo("test.1", strings.NewReader(input), &out, &diagnostics, groff.WithPackage("man"), groff.WithStrict(true))
	if err != nil || diagnostics.Len() != 0 {
		t.Fatalf("processReader: %v %s", err, diagnostics.String())
	}
	for _, want := range []string{"test.1:1:1: .TH HELLO 1", "test.1:2:1: .SH NAME"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
}

func TestCLI_ParseFileHelper(t *testing.T) {
	tokens, err := ParseFile(filepath.Join("..", "..", "testdata", "simple.1"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	if len(tokens) < 20 {
		t.Errorf("expected at least 20 tokens, got %d", len(tokens))
	}
}

func TestCLI_UnterminatedFinalLine(t *testing.T) {
	var out, diagnostics bytes.Buffer
	err := processReaderTo("test.1", strings.NewReader(".TH TEST 1\nunterminated final line"), &out, &diagnostics)
	if err != nil || !strings.Contains(out.String(), "unterminated final line") {
		t.Fatalf("final line: %s %v", out.String(), err)
	}
}

func TestCLI_PartialInputAndOutputErrors(t *testing.T) {
	failure := errors.New("source read failed")
	var out, diagnostics bytes.Buffer
	err := processReaderTo("partial.1", &lexTerminalReader{data: "snow 雪", err: failure}, &out, &diagnostics)
	if !errors.Is(err, failure) || !strings.Contains(out.String(), "snow 雪") || !strings.Contains(diagnostics.String(), "partial.1:1:7: error:") {
		t.Fatalf("partial input hidden: out=%q diagnostics=%q err=%v", out.String(), diagnostics.String(), err)
	}
	out.Reset()
	diagnostics.Reset()
	err = processReaderTo("strict.1", strings.NewReader(".UNKNOWN arg\nretained tail\n"), &out, &diagnostics, groff.WithStrict(true))
	if !errors.Is(err, groff.ErrNoTokenMatch) || !strings.Contains(out.String(), "retained tail") || !strings.Contains(diagnostics.String(), "strict.1:1:1: error:") {
		t.Fatalf("strict partial source: %q %q %v", out.String(), diagnostics.String(), err)
	}
	writerErr := errors.New("output failed")
	if err := processReaderTo("test.1", strings.NewReader("text\n"), lexErrorWriter{writerErr}, io.Discard); !errors.Is(err, writerErr) {
		t.Fatalf("output error lost: %v", err)
	}
	path := filepath.Join(t.TempDir(), "partial.1")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 64*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	tokens, err := ParseFile(path)
	var limit *groff.DocumentLimitError
	if !errors.As(err, &limit) || len(tokens) == 0 {
		t.Fatalf("legacy file helper discarded resource-failure partials: tokens=%d err=%v", len(tokens), err)
	}
}

type lexTerminalReader struct {
	data string
	err  error
	done bool
}

func (r *lexTerminalReader) Read(p []byte) (int, error) {
	if r.done {
		panic("retried terminal input")
	}
	r.done = true
	return copy(p, r.data), r.err
}

type lexErrorWriter struct{ err error }

func (w lexErrorWriter) Write([]byte) (int, error) { return 0, w.err }
