package groff

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

type registryTestToken struct {
	Command, Args, Package string
}

func (t *registryTestToken) String() string       { return ".CUSTOM " + t.Package + " " + t.Args }
func (t *registryTestToken) TokenName() string    { return t.Command }
func (t *registryTestToken) TokenKind() TokenKind { return KindRequest }
func (t *registryTestToken) PackageName() string  { return t.Package }

var _ Node = (*registryTestToken)(nil)

var errRegistryTestSyntax = errors.New("custom syntax rejected")

type registryTestHandler struct {
	mode  string
	calls atomic.Int64
}

func (h *registryTestHandler) CanHandle(cmd, pkg string) bool {
	wantPackage := "man"
	if h.mode == "ms" {
		wantPackage = "ms"
	}
	return (cmd == ".LOCAL" || cmd == "'LOCAL") && pkg == wantPackage
}

func (h *registryTestHandler) Parse(cmd, args, pkg string) (Token, error) {
	h.calls.Add(1)
	switch h.mode {
	case "nil-token":
		return nil, nil
	case "typed-nil":
		return (*registryTestToken)(nil), nil
	case "reject":
		return nil, errRegistryTestSyntax
	}
	return &registryTestToken{Command: cmd, Args: args, Package: pkg}, nil
}

var _ CustomHandler = (*registryTestHandler)(nil)

//go:embed testdata/registry/*.txtar
var registryCases embed.FS

func TestRegistryScenarios(t *testing.T) {
	paths, err := fs.Glob(registryCases, "testdata/registry/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := registryCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for _, f := range txtar.Parse(data).Files {
				if _, duplicate := files[f.Name]; duplicate {
					t.Fatalf("duplicate fixture %q", f.Name)
				}
				switch f.Name {
				case "input.1", "options.json", "expected.txt", "error.txt", "extract.txt":
				default:
					t.Fatalf("unknown fixture %q", f.Name)
				}
				files[f.Name] = f.Data
			}
			for _, name := range []string{"input.1", "options.json", "expected.txt", "extract.txt"} {
				if _, ok := files[name]; !ok {
					t.Fatalf("missing fixture %s", name)
				}
			}
			var options struct {
				Package  string   `json:"package"`
				Handlers []string `json:"handlers"`
			}
			if err := json.Unmarshal(files["options.json"], &options); err != nil {
				t.Fatal(err)
			}
			for _, strict := range []bool{false, true} {
				t.Run(fmt.Sprintf("strict=%v", strict), func(t *testing.T) {
					registry := NewRegistry()
					for _, mode := range options.Handlers {
						registry.RegisterCustom(&registryTestHandler{mode: mode})
					}
					opts := []Option{WithRegistry(registry), WithPackage(options.Package), WithStrict(strict)}
					doc, docErr := ParseDocument(bytes.NewReader(files["input.1"]), opts...)
					a, astErr := ParseAST(bytes.NewReader(files["input.1"]), opts...)
					wantErr := strings.TrimSpace(string(files["error.txt"]))
					for api, err := range map[string]error{"document": docErr, "AST": astErr} {
						if strict && wantErr != "" {
							if err == nil || !strings.Contains(err.Error(), wantErr) {
								t.Errorf("%s: want error containing %q, got %v", api, wantErr, err)
							}
						} else if err != nil {
							t.Errorf("%s: %v", api, err)
						}
					}
					if a == nil || a.Source() != string(files["input.1"]) {
						t.Fatal("AST did not preserve original source")
					}
					if a.Root().Text() != string(files["extract.txt"]) {
						t.Errorf("extraction: %q, want %q", a.Root().Text(), files["extract.txt"])
					}
					var astTokens strings.Builder
					for _, node := range a.Root().FindKind(ASTLine) {
						tok, err := node.Token()
						if err != nil || tok == nil {
							t.Fatalf("AST typed token: %v, %v", tok, err)
						}
						fmt.Fprintf(&astTokens, "%s | %s\n", tok.String(), strings.TrimSuffix(node.Raw(), "\n"))
						if custom, ok := tok.(*registryTestToken); ok {
							attrs := node.Attributes()
							if attrs.TokenKind != KindRequest || attrs.Package != custom.Package {
								t.Fatalf("custom token metadata missing: %+v", attrs)
							}
							original := custom.Args
							custom.Args = "caller mutation"
							again, err := node.Token()
							if err != nil || again.(*registryTestToken).Args != original {
								t.Fatalf("custom token snapshot was mutable: %v %v", again, err)
							}
						}
					}
					if astTokens.String() != string(files["expected.txt"]) {
						t.Errorf("AST tokens:\n%s\nwant:\n%s", astTokens.String(), files["expected.txt"])
					}
					if doc != nil {
						var tokens strings.Builder
						for _, tok := range doc.Tokens {
							fmt.Fprintf(&tokens, "%s | %s\n", tok.Token.String(), tok.RawLine)
						}
						if tokens.String() != string(files["expected.txt"]) {
							t.Errorf("document tokens:\n%s\nwant:\n%s", tokens.String(), files["expected.txt"])
						}
						if wantErr != "" && len(doc.Diagnostics) != 1 {
							// Unknown permissive controls have no legacy diagnostic.
							if wantErr != ErrNoTokenMatch.Error() {
								t.Errorf("missing document error diagnostic: %v", doc.Diagnostics)
							}
						}
					}
					if wantErr != "" && (len(a.Diagnostics()) != 1 || !strings.Contains(a.Diagnostics()[0].Message, wantErr)) {
						t.Errorf("AST diagnostics: %v", a.Diagnostics())
					}
					if wantErr == "" && (len(a.Diagnostics()) != 0 || len(doc.Diagnostics) != 0) {
						t.Fatal("unexpected diagnostics")
					}
				})
			}
		})
	}
}

func TestCustomHandlerOptionAndIncludes(t *testing.T) {
	h := &registryTestHandler{}
	registry := NewRegistry()
	doc, err := ParseDocument(strings.NewReader(".LOCAL root\n.so child\n"), WithStrict(true), WithPackage("man"), WithRegistry(registry), WithCustomHandler(h), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) {
		return strings.NewReader("'LOCAL included\n"), nil
	})))
	if err != nil || len(doc.Tokens) != 2 || h.calls.Load() != 2 {
		t.Fatalf("local handler/included source: doc=%+v calls=%d err=%v", doc, h.calls.Load(), err)
	}
	ast, err := ParseAST(strings.NewReader(".LOCAL AST\n"), WithStrict(true), WithPackage("man"), WithRegistry(registry), WithCustomHandler(h))
	if err != nil || len(ast.Diagnostics()) != 0 || h.calls.Load() != 3 {
		t.Fatalf("AST local handler: calls=%d err=%v", h.calls.Load(), err)
	}
	for _, r := range []*Registry{registry, nil} {
		_, err := r.ParseLine(".LOCAL independent", "man")
		if !errors.Is(err, ErrNoTokenMatch) {
			t.Fatalf("local handler polluted another registry: %v", err)
		}
	}
	_, err = ParseLineWithPackage(".LOCAL global", "man")
	if !errors.Is(err, ErrNoTokenMatch) {
		t.Fatalf("local handler polluted legacy default: %v", err)
	}
}

func TestRegistryErrorsAndGeneratedFallback(t *testing.T) {
	for _, tc := range []struct {
		name     string
		handlers []CustomHandler
		want     error
	}{
		{"duplicate", []CustomHandler{&registryTestHandler{}, &registryTestHandler{}}, ErrAmbiguousHandler},
		{"nil handler", []CustomHandler{nil}, ErrInvalidHandler},
		{"typed nil handler", []CustomHandler{(*registryTestHandler)(nil)}, ErrInvalidHandler},
		{"nil token", []CustomHandler{&registryTestHandler{mode: "nil-token"}}, ErrInvalidHandler},
		{"typed nil token", []CustomHandler{&registryTestHandler{mode: "typed-nil"}}, ErrInvalidHandler},
		{"handler syntax error", []CustomHandler{&registryTestHandler{mode: "reject"}}, errRegistryTestSyntax},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r Registry
			for _, h := range tc.handlers {
				r.RegisterCustom(h)
			}
			for _, permissive := range []bool{false, true} {
				_, handled, err := r.DispatchCustom(".LOCAL", "argument", "man")
				if !handled || !errors.Is(err, tc.want) {
					t.Fatalf("dispatch: handled=%v err=%v want=%v", handled, err, tc.want)
				}
				_, err = parseLineWithRegistry(".LOCAL argument", "man", permissive, &r)
				if !errors.Is(err, tc.want) {
					t.Fatalf("line: got %v, want %v", err, tc.want)
				}
			}
			if errors.Is(tc.want, ErrAmbiguousHandler) {
				for _, h := range tc.handlers {
					if h.(*registryTestHandler).calls.Load() != 0 {
						t.Error("ambiguous dispatch executed a Parse callback")
					}
				}
			}
		})
	}
	r := NewRegistry()
	r.RegisterCustom(&registryTestHandler{})
	tok, err := r.ParseLine(".TH TOOL 1", "man")
	if _, ok := tok.(*TokenMacroTH); err != nil || !ok {
		t.Fatalf("generated fallback: %T %v", tok, err)
	}
	_, err = r.ParseLine(".UNREGISTERED", "man")
	if !errors.Is(err, ErrNoTokenMatch) {
		t.Fatal(err)
	}
	tok, err = r.ParseLinePermissive("'UNREGISTERED  raw", "man")
	if _, ok := tok.(*TokenUnknownControl); err != nil || !ok || tok.String() != "'UNREGISTERED  raw" {
		t.Fatalf("unknown preservation: %T %v", tok, err)
	}
	r.RegisterCustom(registryTitleHandler{&registryTestHandler{}})
	tok, err = r.ParseLine(".TH OVERRIDE 1", "man")
	if custom, ok := tok.(*registryTestToken); err != nil || !ok || custom.Args != "OVERRIDE 1" {
		t.Fatalf("custom precedence over generated binder: %T %v", tok, err)
	}
}

type registryTitleHandler struct{ *registryTestHandler }

func (registryTitleHandler) CanHandle(cmd, pkg string) bool { return cmd == ".TH" && pkg == "man" }

type registryReadHook struct {
	io.Reader
	first func()
}

func (r *registryReadHook) Read(p []byte) (int, error) {
	if r.first != nil {
		r.first()
		r.first = nil
	}
	return r.Reader.Read(p)
}

func TestRegistrySnapshotAtParseEntry(t *testing.T) {
	for _, ast := range []bool{false, true} {
		t.Run(fmt.Sprintf("AST=%v", ast), func(t *testing.T) {
			r := NewRegistry()
			h := &registryTestHandler{}
			reader := &registryReadHook{Reader: strings.NewReader(".LOCAL first\n.LOCAL second\n"), first: func() { r.RegisterCustom(h) }}
			opts := []Option{WithRegistry(r), WithPackage("man")}
			if ast {
				a, err := ParseAST(reader, opts...)
				if err != nil || len(a.Diagnostics()) != 2 {
					t.Fatalf("snapshot AST: diagnostics=%v err=%v", a.Diagnostics(), err)
				}
			} else {
				doc, err := ParseDocument(reader, opts...)
				if err != nil || len(doc.Tokens) != 2 {
					t.Fatalf("snapshot document: doc=%+v err=%v", doc, err)
				}
				for _, tok := range doc.Tokens {
					if _, ok := tok.Token.(*TokenUnknownControl); !ok {
						t.Fatalf("registration changed an active parse: %T", tok.Token)
					}
				}
			}
			if h.calls.Load() != 0 {
				t.Fatal("late registration ran during existing parse")
			}
			if _, err := r.ParseLine(".LOCAL next", "man"); err != nil || h.calls.Load() != 1 {
				t.Fatalf("next parse did not see registration: calls=%d err=%v", h.calls.Load(), err)
			}
		})
	}
}

func TestRegistryConcurrentDispatchAndRegistration(t *testing.T) {
	r := NewRegistry()
	h := &registryTestHandler{}
	r.RegisterCustom(h)
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; i < 100; i++ {
			r.RegisterCustom(registryNeverHandler{})
		}
	})
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			for j := 0; j < 100; j++ {
				tok, err := r.ParseLine(".LOCAL parallel", "man")
				if err != nil || tok.String() != ".CUSTOM man parallel" {
					t.Errorf("parallel dispatch: %v %v", tok, err)
				}
			}
		})
	}
	wg.Wait()
	if h.calls.Load() != 800 {
		t.Fatalf("lost dispatches: %d", h.calls.Load())
	}
}

func TestRegistryIsolatedConcurrentParsers(t *testing.T) {
	var wg sync.WaitGroup
	for _, reject := range []bool{false, true} {
		wg.Go(func() {
			r := NewRegistry()
			h := &registryTestHandler{}
			if reject {
				h.mode = "reject"
			}
			r.RegisterCustom(h)
			for i := 0; i < 20; i++ {
				opts := []Option{WithRegistry(r), WithStrict(true), WithPackage("man")}
				_, docErr := ParseDocument(strings.NewReader(".LOCAL value\n"), opts...)
				_, astErr := ParseAST(strings.NewReader(".LOCAL value\n"), opts...)
				if reject != (docErr != nil) || reject != (astErr != nil) {
					t.Errorf("isolated parser reject=%v: document=%v AST=%v", reject, docErr, astErr)
				}
			}
			if h.calls.Load() != 40 {
				t.Errorf("handler calls=%d, want 40", h.calls.Load())
			}
		})
	}
	wg.Wait()
}

// Exercise the compatibility global API in its own process so registration
// cannot change another scenario, example or fuzz target's parsing behavior.
func TestRegistryLegacyGlobal(t *testing.T) {
	for _, arg := range os.Args {
		if arg != "registry-global-helper" {
			continue
		}
		h := &registryTestHandler{}
		RegisterCustomHandler(registryGlobalHandler{h})
		if tok, err := ParseLine(".GLOBAL line"); err != nil || tok.(*registryTestToken).Args != "line" {
			t.Fatalf("legacy line registration: %v %v", tok, err)
		}
		if _, err := ParseLineWithPackage(".GLOBAL package", "man"); err != nil {
			t.Fatal(err)
		}
		if _, err := ParseDocument(strings.NewReader(".GLOBAL document\n"), WithStrict(true)); err != nil {
			t.Fatal(err)
		}
		if _, err := ParseAST(strings.NewReader(".GLOBAL AST\n"), WithStrict(true)); err != nil {
			t.Fatal(err)
		}
		if h.calls.Load() != 4 {
			t.Fatalf("global compatibility calls=%d", h.calls.Load())
		}
		if _, err := ParseDocument(strings.NewReader(".GLOBAL isolated\n"), WithStrict(true), WithRegistry(NewRegistry())); !errors.Is(err, ErrNoTokenMatch) {
			t.Fatalf("explicit empty registry inherited global registration: %v", err)
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestRegistryLegacyGlobal$", "--", "registry-global-helper")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("global API subprocess: %v\n%s", err, out)
	}
}

type registryGlobalHandler struct{ *registryTestHandler }

func (registryGlobalHandler) CanHandle(cmd, pkg string) bool { return cmd == ".GLOBAL" }

type registryNeverHandler struct{}

func (registryNeverHandler) CanHandle(string, string) bool { return false }
func (registryNeverHandler) Parse(string, string, string) (Token, error) {
	panic("unmatched handler must never parse")
}
