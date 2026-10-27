package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_ProcessReader(t *testing.T) {
	input := ".TH HELLO 1\n.SH NAME\nhello - greeting\n"
	*pkgFlag = "man"
	*strictFlag = true

	var buf bytes.Buffer
	// Redirect stdout for testing
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := processReader("test.1", strings.NewReader(input))
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("processReader error: %v", err)
	}

	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "test.1:1:1: .TH HELLO 1") {
		t.Errorf("expected output to contain span and .TH, got: %s", out)
	}
	if !strings.Contains(out, "test.1:2:1: .SH NAME") {
		t.Errorf("expected output to contain span and .SH, got: %s", out)
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
	input := ".TH TEST 1\nunterminated final line"
	*strictFlag = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := processReader("test.1", strings.NewReader(input))
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("processReader error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "unterminated final line") {
		t.Errorf("expected final unterminated line in output, got: %s", out)
	}
}
