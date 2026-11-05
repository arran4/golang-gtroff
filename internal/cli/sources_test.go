package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/source"
)

func TestSourcesSharedLoader(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".so common\n")}, "common": {Data: []byte("child\n")}}
	var opened []string
	env := Environment{In: strings.NewReader(".so common\n"), OpenRoot: func(name string) (manlib.Root, io.Closer, error) {
		opened = append(opened, name)
		return manlib.Root{FS: fsys}, nil, nil
	}}
	set, err := loadSources("docs/main.1", sourceOptions{}, env)
	if err != nil || len(set.Files()) != 2 || len(opened) != 1 || opened[0] != "docs" {
		t.Fatal("named file did not use bounded context", err)
	}
	set, err = loadSources("docs/main.1", sourceOptions{Options: source.Options{NoIncludes: true}}, env)
	if err != nil || len(set.Files()) != 1 || set.Edges()[0].State != "disabled" {
		t.Fatal("no-includes policy ignored", err)
	}
	opened = nil
	set, err = loadSources("-", sourceOptions{}, env)
	if err != nil || len(opened) != 0 || set.Edges()[0].State != "unresolved" {
		t.Fatal("stdin acquired ambient root", err)
	}
	env.In = strings.NewReader(".so common\n")
	set, err = loadSources("-", sourceOptions{root: "docs"}, env)
	if err != nil || len(set.Files()) != 2 || opened[0] != "docs" {
		t.Fatal("explicit stdin root ignored", err)
	}
	opened = nil
	_, err = loadSources("outside/main.1", sourceOptions{root: "docs"}, env)
	if err == nil || len(opened) != 0 {
		t.Fatal("root escape reached filesystem")
	}
}

func TestSourcesBoundaries(t *testing.T) {
	for _, args := range [][]string{
		{"sources", "--format", "yaml"}, {"sources", "--package", "unknown"}, {"sources", "a", "b"},
		{"sources", "--max-bytes", "0"}, {"sources", "--max-nodes", "2"}, {"sources", "--max-files", "0"}, {"sources", "--max-depth", "129"},
	} {
		var out, stderr bytes.Buffer
		code := Run(args, Environment{In: panicReader{}, Out: &out, Err: &stderr, OpenRoot: func(string) (manlib.Root, io.Closer, error) {
			t.Fatal("opened on invalid usage")
			return manlib.Root{}, nil, nil
		}})
		if code != ExitUsage || out.Len() != 0 {
			t.Fatalf("invalid usage code %d: %v", code, args)
		}
	}
	for _, writer := range []io.Writer{errorWriter{}, shortWriter{}} {
		if code := Run([]string{"sources"}, Environment{In: strings.NewReader("text\n"), Out: writer}); code != ExitIO {
			t.Fatal("output failure ignored")
		}
	}
	rootCloser := &closeReader{err: errBoundary}
	var out bytes.Buffer
	code := Run([]string{"sources", "main.1"}, Environment{Out: &out, OpenRoot: func(string) (manlib.Root, io.Closer, error) {
		return manlib.Root{FS: fstest.MapFS{"main.1": {Data: []byte("text\n")}}}, rootCloser, nil
	}})
	if code != ExitIO || !rootCloser.closed {
		t.Fatal("root close failure lost")
	}
	reader := &closeReader{Reader: strings.NewReader("text\n")}
	set, err := loadSources("main.1", sourceOptions{}, Environment{Open: func(string) (io.ReadCloser, error) { return reader, nil }})
	if err != nil || !reader.closed || set.File(1).AST().Source() != "text\n" {
		t.Fatal("fallback stream ownership lost", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = loadSources("-", sourceOptions{}, Environment{Context: ctx, In: panicReader{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
}
