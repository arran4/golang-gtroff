package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLintRejectsConfigBeforeInput(t *testing.T) {
	for _, args := range [][]string{{"lint", "--disable", "typo", "missing.1"}, {"lint", "--max-files", "0", "missing.1"}, {"lint", "--fail-on", "bogus", "missing.1"}, {"lint", "-", "-"}} {
		var out, stderr bytes.Buffer
		opened := 0
		code := Run(args, Environment{Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { opened++; return nil, errors.New("unexpected open") }})
		if code != ExitUsage || opened != 0 {
			t.Fatalf("args %v code %d opened %d: %s", args, code, opened, stderr.String())
		}
	}
}
func TestLintPolicyCloseAndCancellation(t *testing.T) {
	closeErr := errors.New("policy close failed")
	var out, stderr bytes.Buffer
	opened := []string{}
	code := Run([]string{"lint", "--rules", "policy.json", "input.1"}, Environment{Out: &out, Err: &stderr, Open: func(path string) (io.ReadCloser, error) {
		opened = append(opened, path)
		return lintPolicyReader{Reader: strings.NewReader(`{"schemaVersion":1}`), err: closeErr}, nil
	}})
	if code != ExitIO || len(opened) != 1 || !strings.Contains(stderr.String(), closeErr.Error()) {
		t.Fatal("policy close ignored", code, opened, stderr.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opened = nil
	code = Run([]string{"lint", "--rules", "policy.json", "input.1"}, Environment{Context: ctx, Out: &out, Err: &stderr, Open: func(path string) (io.ReadCloser, error) {
		opened = append(opened, path)
		return nil, errors.New("unexpected")
	}})
	if code != ExitIO || len(opened) != 0 {
		t.Fatal("canceled command opened files")
	}
}

type lintPolicyReader struct {
	io.Reader
	err error
}

func (r lintPolicyReader) Close() error { return r.err }
