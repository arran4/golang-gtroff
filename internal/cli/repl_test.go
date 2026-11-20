package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/scriptapi"
)

func TestREPLCoreValidationBeforeInput(t *testing.T) {
	for _, args := range [][]string{
		{"script", "--repl"},
		{"script", "--repl", "-"},
		{"script", "--repl", "a", "b"},
		{"script", "--repl", "--max-cells", "0"},
		{"script", "--repl", "--max-session-bytes", "0"},
		{"script", "--repl", "--in-place"},
	} {
		var stderr bytes.Buffer
		opened := 0
		code := Run(args, Environment{Err: &stderr, Open: func(string) (io.ReadCloser, error) { opened++; return nil, errors.New("unexpected") }})
		want := ExitUsage
		if len(args) == 2 {
			want = ExitIO
		}
		if code != want || opened != 0 {
			t.Fatal(args, code, opened, stderr.String())
		}
	}
}
func TestREPLFactoryFailuresAndCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		ctx     context.Context
		factory scriptapi.SessionFactory
		message string
	}{
		{ctx, func(context.Context, scriptapi.Request, scriptapi.SessionOptions) (scriptapi.Session, error) {
			t.Fatal("called canceled factory")
			return nil, nil
		}, "canceled"},
		{nil, func(context.Context, scriptapi.Request, scriptapi.SessionOptions) (scriptapi.Session, error) {
			return nil, nil
		}, "nil script session"},
		{nil, func(context.Context, scriptapi.Request, scriptapi.SessionOptions) (scriptapi.Session, error) {
			return nil, scriptapi.ErrLimit
		}, "limit"},
	} {
		var stderr bytes.Buffer
		code := Run([]string{"script", "--repl", "--prompt=false"}, Environment{Context: tc.ctx, In: strings.NewReader(""), Err: &stderr, ScriptSession: tc.factory})
		if code != ExitIO || !strings.Contains(stderr.String(), tc.message) {
			t.Fatal(code, stderr.String())
		}
	}
}
