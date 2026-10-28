package groff

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/include/*.txtar
var includeCases embed.FS

type includeResolverFunc func(string) (io.Reader, error)

func (f includeResolverFunc) Resolve(name string) (io.Reader, error) { return f(name) }

type observedInclude struct {
	io.Reader
	closed   int
	closeErr error
}

func (r *observedInclude) Close() error {
	r.closed++
	return r.closeErr
}

func TestIncludeScenarios(t *testing.T) {
	paths, err := fs.Glob(includeCases, "testdata/include/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := includeCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := make(map[string][]byte)
			tree := make(fstest.MapFS)
			for _, f := range txtar.Parse(data).Files {
				if _, exists := files[f.Name]; exists {
					t.Fatalf("duplicate fixture %q", f.Name)
				}
				files[f.Name] = f.Data
				if strings.HasPrefix(f.Name, "tree/") {
					tree[strings.TrimPrefix(f.Name, "tree/")] = &fstest.MapFile{Data: f.Data}
				}
			}
			for _, name := range []string{"input.1", "expected.txt"} {
				if _, ok := files[name]; !ok {
					t.Fatalf("missing %s", name)
				}
			}
			for _, mode := range []string{"syntax-only", "permissive", "strict"} {
				t.Run(mode, func(t *testing.T) {
					var opened []*observedInclude
					resolver := includeResolverFunc(func(name string) (io.Reader, error) {
						data, err := fs.ReadFile(tree, name)
						if err != nil {
							return nil, err
						}
						r := &observedInclude{Reader: bytes.NewReader(data)}
						opened = append(opened, r)
						return r, nil
					})
					opts := []Option{WithSourceName("input.1"), WithMaxIncludeDepth(2)}
					if mode != "syntax-only" {
						opts = append(opts, WithIncludeResolver(resolver))
					}
					if mode == "strict" {
						opts = append(opts, WithStrict(true))
					}
					doc, err := ParseDocument(bytes.NewReader(files["input.1"]), opts...)
					for i, r := range opened {
						if r.closed != 1 {
							t.Errorf("include reader %d closed %d times", i, r.closed)
						}
					}
					if mode == "syntax-only" {
						if err != nil || len(opened) != 0 {
							t.Fatalf("syntax-only: %v; opened %d files", err, len(opened))
						}
						var raw strings.Builder
						for _, token := range doc.Tokens {
							fmt.Fprintln(&raw, token.RawLine)
						}
						if raw.String() != string(files["input.1"]) {
							t.Fatalf("syntax-only lost directive: %q", raw.String())
						}
						return
					}
					wantErr := strings.TrimSpace(string(files["error.txt"]))
					if mode == "strict" && wantErr != "" {
						if err == nil || !strings.Contains(err.Error(), wantErr) {
							t.Fatalf("want error containing %q, got %v", wantErr, err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					var got strings.Builder
					for _, token := range doc.Tokens {
						fmt.Fprintf(&got, "%s:%d %s\n", token.Span.Start.Filename, token.Span.Start.Line, token.RawLine)
					}
					if got.String() != string(files["expected.txt"]) {
						t.Errorf("tokens:\n%s\nwant:\n%s", got.String(), files["expected.txt"])
					}
					if wantErr == "" && len(doc.Diagnostics) != 0 {
						t.Fatalf("unexpected diagnostics: %v", doc.Diagnostics)
					}
					if wantErr != "" && (len(doc.Diagnostics) != 1 || !strings.Contains(doc.Diagnostics[0].Message, wantErr)) {
						t.Fatalf("want one diagnostic containing %q, got %v", wantErr, doc.Diagnostics)
					}
					if wantErr != "" {
						d := doc.Diagnostics[0]
						got := fmt.Sprintf("%s:%d %s\n", d.Position.Filename, d.Position.Line, d.Severity)
						if got != string(files["diagnostic.txt"]) {
							t.Errorf("diagnostic position/severity: %q, want %q", got, files["diagnostic.txt"])
						}
					}
				})
			}
		})
	}
}

func TestIncludeReaderOwnership(t *testing.T) {
	closeErr := errors.New("include close failed")
	readErr := errors.New("include read failed")
	for _, tc := range []struct {
		name  string
		input io.Reader
		close error
		opts  []Option
		want  error
		text  string
	}{
		{name: "success", input: strings.NewReader("child\n")},
		{name: "strict parse failure", input: strings.NewReader(".de\n"), text: "unterminated macro definition"},
		{name: "read failure", input: includeErrorReader{readErr}, want: readErr},
		{name: "line limit inherited", input: strings.NewReader("oversized\n"), opts: []Option{WithMaxLineLength(8)}, text: "line length budget"},
		{name: "line count inherited", input: strings.NewReader("one\ntwo\n"), opts: []Option{WithMaxLines(1)}, text: "line budget"},
		{name: "close failure", input: strings.NewReader("child\n"), close: closeErr, want: closeErr},
		{name: "read and close failure", input: includeErrorReader{readErr}, close: closeErr, want: readErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := &observedInclude{Reader: tc.input, closeErr: tc.close}
			root := &observedInclude{Reader: strings.NewReader(".so x\n")}
			opts := append([]Option{WithStrict(true), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return child, nil }))}, tc.opts...)
			_, err := ParseDocument(root, opts...)
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
			if tc.close != nil && !errors.Is(err, tc.close) {
				t.Errorf("lost close error: %v", err)
			}
			if tc.text != "" && (err == nil || !strings.Contains(err.Error(), tc.text)) {
				t.Errorf("want error containing %q, got %v", tc.text, err)
			}
			if tc.want == nil && tc.text == "" && err != nil {
				t.Fatal(err)
			}
			if child.closed != 1 || root.closed != 0 {
				t.Fatalf("closed child=%d, caller-owned input=%d", child.closed, root.closed)
			}
		})
	}
}

type includeErrorReader struct{ err error }

func (r includeErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestIncludeBorrowedReader(t *testing.T) {
	child := &observedInclude{Reader: strings.NewReader("borrowed\n")}
	resolver := includeResolverFunc(func(string) (io.Reader, error) {
		return struct{ io.Reader }{child}, nil
	})
	doc, err := ParseDocument(strings.NewReader(".so child\n"), WithStrict(true), WithIncludeResolver(resolver))
	if err != nil || len(doc.Tokens) != 1 || doc.Tokens[0].RawLine != "borrowed" || child.closed != 0 {
		t.Fatalf("borrowed reader: doc=%+v closed=%d err=%v", doc, child.closed, err)
	}
}

func TestIncludeCustomResolverBounds(t *testing.T) {
	for _, limit := range []int{1, 0, -1, 1000} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			calls, closed := 0, 0
			resolver := includeResolverFunc(func(string) (io.Reader, error) {
				calls++
				return &countedInclude{Reader: strings.NewReader(fmt.Sprintf(".so unique%d\n", calls)), closed: &closed}, nil
			})
			_, err := ParseDocument(strings.NewReader(".so start\n"), WithStrict(true), WithIncludeResolver(resolver), WithMaxIncludeDepth(limit))
			want := limit
			if want <= 0 {
				want = 10
			}
			if want > 128 {
				want = 128
			}
			if err == nil || !strings.Contains(err.Error(), "maximum include depth") || calls != want || closed != want {
				t.Fatalf("calls=%d closed=%d want=%d error=%v", calls, closed, want, err)
			}
		})
	}
	_, err := ParseDocument(strings.NewReader(".so nil\n"), WithStrict(true), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return nil, nil })))
	if err == nil || !strings.Contains(err.Error(), "nil reader") {
		t.Fatalf("nil resolver reader: %v", err)
	}
}

type countedInclude struct {
	io.Reader
	closed *int
}

func (r *countedInclude) Close() error { *r.closed++; return nil }

func TestDirectoryIncludeContainment(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	writeIncludeFile(t, filepath.Join(base, "allowed.1"), "allowed\n")
	writeIncludeFile(t, filepath.Join(outside, "outside.1"), "outside\n")
	r := NewSafeDirectoryIncludeResolver(base, 2)
	for _, name := range []string{"../outside.1", filepath.Join(outside, "outside.1"), "missing.1", "."} {
		if reader, err := r.Resolve(name); err == nil {
			reader.(io.Closer).Close()
			t.Errorf("accepted denied/missing include %q", name)
		}
	}
	t.Run("symlink", func(t *testing.T) {
		relative, err := filepath.Rel(base, filepath.Join(outside, "outside.1"))
		if err != nil {
			t.Fatal(err)
		}
		for i, target := range []string{filepath.Join(outside, "outside.1"), relative} {
			name := fmt.Sprintf("escape%d.1", i)
			if err := os.Symlink(target, filepath.Join(base, name)); err != nil {
				t.Skipf("OS cannot create symlink: %v", err)
			}
			if reader, err := r.Resolve(name); err == nil {
				reader.(io.Closer).Close()
				t.Errorf("symlink %q escaped include root", target)
			}
		}
		if err := os.Symlink("allowed.1", filepath.Join(base, "alias.1")); err != nil {
			t.Fatal(err)
		}
		reader := openInclude(t, r, "alias.1")
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil || string(data) != "allowed\n" {
			t.Fatalf("contained symlink: %q, %v", data, err)
		}
		if other, err := r.Resolve("allowed.1"); err == nil {
			other.(io.Closer).Close()
			t.Fatal("symlink alias bypassed cycle detection")
		}
	})
}

func TestDirectoryIncludeLifecycle(t *testing.T) {
	base := t.TempDir()
	writeIncludeFile(t, filepath.Join(base, "a"), "A\n")
	writeIncludeFile(t, filepath.Join(base, "b"), "B\n")
	r := NewSafeDirectoryIncludeResolver(base, 1)
	reader := openInclude(t, r, "a")
	if other, err := r.Resolve("./a"); err == nil || !strings.Contains(err.Error(), "cycle") {
		if err == nil {
			other.(io.Closer).Close()
		}
		t.Fatalf("active cycle: %v", err)
	}
	if other, err := r.Resolve("b"); err == nil || !strings.Contains(err.Error(), "depth") {
		if err == nil {
			other.(io.Closer).Close()
		}
		t.Fatalf("active depth: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("repeat Close: %v", err)
	}
	for i := 0; i < 2; i++ {
		doc, err := ParseDocument(strings.NewReader(".so a\n.mso b\n.so a\n"), WithStrict(true), WithIncludeResolver(r), WithMaxIncludeDepth(1))
		if err != nil || len(doc.Tokens) != 3 || len(doc.Diagnostics) != 0 {
			t.Fatalf("sequential includes/reused resolver: doc=%+v err=%v", doc, err)
		}
	}
	// Simultaneous Close calls are safe, and a subsequent Resolve can proceed.
	reader = openInclude(t, r, "a")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() { reader.Close() })
	}
	wg.Wait()
	reader = openInclude(t, r, "a")
	reader.Close()
	writeIncludeFile(t, filepath.Join(base, "bad"), ".de\n")
	_, err := ParseDocument(strings.NewReader(".so bad\n"), WithStrict(true), WithIncludeResolver(r))
	if err == nil {
		t.Fatal("accepted unterminated included definition")
	}
	reader = openInclude(t, r, "bad")
	reader.Close()
}

func TestDirectoryIncludeConcurrentResolve(t *testing.T) {
	base := t.TempDir()
	r := NewSafeDirectoryIncludeResolver(base, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		name := fmt.Sprint(i)
		writeIncludeFile(t, filepath.Join(base, name), name)
		wg.Go(func() {
			reader, err := r.Resolve(name)
			if err != nil {
				t.Error(err)
				return
			}
			defer reader.(io.Closer).Close()
			data, err := io.ReadAll(reader)
			if err != nil || string(data) != name {
				t.Errorf("concurrent include %q: data=%q err=%v", name, data, err)
			}
		})
	}
	wg.Wait()
}

func openInclude(t *testing.T, r IncludeResolver, name string) io.ReadCloser {
	t.Helper()
	reader, err := r.Resolve(name)
	if err != nil {
		t.Fatal(err)
	}
	closer, ok := reader.(io.ReadCloser)
	if !ok {
		t.Fatal("directory resolver did not return an owned reader")
	}
	return closer
}

func writeIncludeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
