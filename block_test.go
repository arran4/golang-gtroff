package groff

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"golang.org/x/tools/txtar"
)

type blockTestToken struct{ Command, Header, Body, End string }

func (t *blockTestToken) String() string       { return "formatted block" }
func (t *blockTestToken) TokenName() string    { return normalizedBlockCommand(t.Command) }
func (t *blockTestToken) TokenKind() TokenKind { return KindBlock }
func (t *blockTestToken) PackageName() string  { return "man" }

type blockTestHandler struct {
	command, pkg, mode string
	begins, parses     atomic.Int64
}

func (h *blockTestHandler) CanHandle(cmd, pkg string) bool {
	command := h.command
	if command == "" {
		command = ".BOX"
	}
	return normalizedBlockCommand(cmd) == command && (h.pkg == "" || h.pkg == pkg)
}
func (h *blockTestHandler) Begin(header BlockHeader) (BlockRule, error) {
	h.begins.Add(1)
	if h.mode == "begin-error" {
		return BlockRule{}, errBlockCallback
	}
	if h.mode == "invalid-rule" {
		return BlockRule{}, nil
	}
	return BlockRule{Ends: []string{".END"}, Nested: []BlockPair{{Start: ".BOX", Ends: []string{".END"}}}}, nil
}
func (h *blockTestHandler) ParseBlock(source BlockSource) (Token, error) {
	h.parses.Add(1)
	switch h.mode {
	case "parse-error":
		return nil, errBlockCallback
	case "nil":
		return nil, nil
	case "typed-nil":
		return (*blockTestToken)(nil), nil
	}
	token := &blockTestToken{Command: source.Header.Command, Header: source.Header.Line.RawSource, Body: source.BodySource(), End: source.Terminator.RawSource}
	// Slice metadata supplied to a plugin must not alias parser-owned state.
	if len(source.Body) > 0 {
		source.Body[0].RawSource = "mutated by plugin"
	}
	source.Rule.Ends[0] = ".MUTATED"
	source.Rule.Nested[0].Ends[0] = ".MUTATED"
	return token, nil
}

var _ BlockHandler = (*blockTestHandler)(nil)
var _ Node = (*blockTestToken)(nil)
var errBlockCallback = errors.New("block callback rejected syntax")

//go:embed testdata/blocks/*.txtar
var blockCases embed.FS
var updateBlocks = flag.Bool("update-blocks", false, "explicitly refresh bounded block capture expectations")

func TestBlockScenarios(t *testing.T) {
	paths, err := fs.Glob(blockCases, "testdata/blocks/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := blockCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			files := map[string][]byte{}
			for _, file := range archive.Files {
				if _, exists := files[file.Name]; exists {
					t.Fatalf("duplicate %s", file.Name)
				}
				switch file.Name {
				case "input.roff", "options.json", "expected.txt":
				default:
					t.Fatalf("unknown %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, ok := files["input.roff"]
			if !ok {
				t.Fatal("missing input.roff")
			}
			var options struct {
				Custom, Strict, CRLF, NoFinalLF bool
				Depth                           int
			}
			if raw, ok := files["options.json"]; ok {
				decoder := json.NewDecoder(bytes.NewReader(raw))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&options); err != nil {
					t.Fatal(err)
				}
			}
			if options.CRLF {
				input = bytes.ReplaceAll(input, []byte("\n"), []byte("\r\n"))
			}
			if options.NoFinalLF {
				input = bytes.TrimSuffix(bytes.TrimSuffix(input, []byte("\n")), []byte("\r"))
			}
			opts := []Option{WithRegistry(NewRegistry()), WithPackage("man"), WithSourceName("input.1"), WithStrict(options.Strict), WithMaxBlockDepth(options.Depth)}
			if options.Custom {
				opts = append(opts, WithBlockHandler(&blockTestHandler{}))
			}
			doc, docErr := ParseDocument(bytes.NewReader(input), opts...)
			assertDocumentSource(t, doc, string(input))
			var got strings.Builder
			fmt.Fprintf(&got, "document error=%v\n", docErr)
			for _, entry := range doc.Tokens {
				fmt.Fprintf(&got, "token %T [%d,%d) raw=%q\n", entry.Token, entry.Span.Start.Offset, entry.Span.End.Offset, entry.RawSource)
				if entry.Block != nil {
					assertBlockSource(t, *entry.Block, string(input))
					if entry.Block.Source() != entry.RawSource {
						t.Fatal("token/block source mismatch")
					}
					fmt.Fprintf(&got, "  closed=%v header=%q body=%q end=%q\n", entry.Block.Closed, entry.Block.Header.Line.RawSource, entry.Block.BodySource(), entry.Block.Terminator.RawSource)
				}
			}
			for _, d := range doc.Diagnostics {
				fmt.Fprintf(&got, "document %s %d:%d %q\n", d.Severity, d.Position.Line, d.Position.Column, d.Message)
			}
			ast, astErr := ParseAST(bytes.NewReader(input), opts...)
			if ast.Source() != string(input) || ast.Root().Raw() != string(input) {
				t.Fatal("AST lost block source")
			}
			fmt.Fprintf(&got, "ast error=%v\n", astErr)
			for _, node := range ast.Root().FindKind(ASTBlock) {
				fmt.Fprintf(&got, "ast block %s closed=%v raw=%q\n", node.Attributes().Name, node.Attributes().Closed, node.Raw())
				if block, ok := node.BlockSource(); ok {
					assertBlockSource(t, *block, string(input))
					if block.Source() != node.Raw() {
						t.Fatalf("AST block raw mismatch: %q != %q", block.Source(), node.Raw())
					}
					original := block.BodySource()
					if len(block.Body) > 0 {
						block.Body[0].RawSource = "caller mutation"
					}
					block.Rule.Ends[0] = ".CHANGED"
					again, _ := node.BlockSource()
					if again.BodySource() != original || again.Rule.Ends[0] == ".CHANGED" {
						t.Fatal("mutable AST capture")
					}
				}
				token, err := node.Token()
				if err != nil {
					t.Fatal(err)
				}
				if token != nil {
					fmt.Fprintf(&got, "  token %T %s\n", token, mustBlockJSON(t, token))
					if tok, ok := token.(*blockTestToken); ok {
						old := tok.Body
						tok.Body = "mutation"
						again, err := node.Token()
						if err != nil || again.(*blockTestToken).Body != old {
							t.Fatal("mutable block token")
						}
					}
				}
			}
			for _, d := range ast.Diagnostics() {
				fmt.Fprintf(&got, "ast %s %d:%d %q\n", d.Severity, d.Position.Line, d.Position.Column, d.Message)
			}
			if *updateBlocks {
				found := false
				for i := range archive.Files {
					if archive.Files[i].Name == "expected.txt" {
						archive.Files[i].Data = []byte(got.String())
						found = true
					}
				}
				if !found {
					archive.Files = append(archive.Files, txtar.File{Name: "expected.txt", Data: []byte(got.String())})
				}
				if err := os.WriteFile(path, txtar.Format(archive), 0644); err != nil {
					t.Fatal(err)
				}
			} else if got.String() != string(files["expected.txt"]) {
				t.Fatalf("got:\n%s\nwant:\n%s", got.String(), files["expected.txt"])
			}
		})
	}
}
func mustBlockJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func assertBlockSource(t *testing.T, b BlockSource, input string) {
	t.Helper()
	cursor := b.Header.Line.Span.Start.Offset
	lines := append([]BlockLine{b.Header.Line}, b.Body...)
	if b.Tail.RawSource != "" {
		lines = append(lines, b.Tail)
	}
	if b.Closed {
		lines = append(lines, b.Terminator)
	}
	for _, line := range lines {
		if line.Span.Start.Offset != cursor || line.Span.End.Offset > len(input) || input[cursor:line.Span.End.Offset] != line.RawSource {
			t.Fatalf("invalid block line %+v", line)
		}
		for _, position := range []Position{line.Span.Start, line.Span.End} {
			prefix := input[:position.Offset]
			last := strings.LastIndexByte(prefix, '\n')
			if position.Line != strings.Count(prefix, "\n")+1 || position.Column != utf8.RuneCountInString(prefix[last+1:])+1 {
				t.Fatalf("invalid block position %+v", position)
			}
		}
		cursor = line.Span.End.Offset
	}
	if b.Span().End.Offset != cursor || b.Source() != input[b.Span().Start.Offset:cursor] {
		t.Fatal("block source/span mismatch")
	}
}

func TestBlockHandlerFailuresAndPrecedence(t *testing.T) {
	for _, mode := range []string{"begin-error", "invalid-rule", "parse-error", "nil", "typed-nil", "duplicate", "nil-registration", "line-conflict"} {
		t.Run(mode, func(t *testing.T) {
			for _, strict := range []bool{false, true} {
				for _, astMode := range []bool{false, true} {
					h := &blockTestHandler{mode: mode}
					r := NewRegistry()
					r.RegisterBlock(h)
					cause := ErrInvalidHandler
					switch mode {
					case "begin-error", "parse-error":
						cause = errBlockCallback
					case "duplicate":
						r.RegisterBlock(h)
						cause = ErrAmbiguousHandler
					case "nil-registration":
						r.RegisterBlock((*blockTestHandler)(nil))
					case "line-conflict":
						r.RegisterCustom(blockLineHandler{command: ".BOX"})
						cause = ErrAmbiguousHandler
					}
					input := ".BOX\n.B must remain raw\n.END\nsuffix\n"
					opts := []Option{WithRegistry(r), WithStrict(strict), WithPackage("man")}
					if astMode {
						a, err := ParseAST(strings.NewReader(input), opts...)
						if a.Source() != input || len(a.Diagnostics()) == 0 || strict != (err != nil) {
							t.Fatalf("AST failure %s: %v %+v", mode, err, a.Diagnostics())
						}
					} else {
						d, err := ParseDocument(strings.NewReader(input), opts...)
						assertDocumentSource(t, d, input)
						if len(d.Diagnostics) == 0 || strict != (err != nil) {
							t.Fatalf("document failure: %v %+v", err, d.Diagnostics)
						}
						if strict && !errors.Is(err, cause) {
							t.Fatalf("cause lost: want %v got %v", cause, err)
						}
					}
					if (mode == "duplicate" || mode == "nil-registration" || mode == "line-conflict") && (h.begins.Load() != 0 || h.parses.Load() != 0) {
						t.Fatal("callback ran before overlap/registration validation")
					}
					if mode == "begin-error" || mode == "invalid-rule" {
						if h.parses.Load() != 0 {
							t.Fatal("ParseBlock after Begin failure")
						}
					}
				}
			}
		})
	}
	// A custom block overrides built-in structure; a custom line explicitly opts
	// out of it. ParseLine itself continues to use only line handlers.
	for _, astMode := range []bool{false, true} {
		h := &blockTestHandler{command: ".TS"}
		input := ".TS\n.B raw\n.END\n"
		if astMode {
			a, err := ParseAST(strings.NewReader(input), WithRegistry(NewRegistry()), WithBlockHandler(h))
			if err != nil || len(a.Diagnostics()) != 0 || len(a.Root().FindKind(ASTBlock)) != 1 {
				t.Fatal(err, a.Diagnostics())
			}
		} else {
			d, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), WithBlockHandler(h))
			if err != nil || len(d.Tokens) != 1 {
				t.Fatal(err)
			}
			if _, ok := d.Tokens[0].Token.(*blockTestToken); !ok {
				t.Fatalf("builtin won: %T", d.Tokens[0].Token)
			}
		}
		if h.parses.Load() != 1 {
			t.Fatal("custom block not called exactly once")
		}
	}
	r := NewRegistry()
	r.RegisterCustom(blockLineHandler{command: ".TS"})
	d, err := ParseDocument(strings.NewReader(".TS\ntext\n"), WithRegistry(r), WithStrict(true))
	if err != nil || len(d.Tokens) != 2 || d.Tokens[0].Block != nil {
		t.Fatal(err)
	}
	a, err := ParseAST(strings.NewReader(".TS\ntext\n"), WithRegistry(r), WithStrict(true))
	if err != nil || len(a.Root().FindKind(ASTBlock)) != 0 {
		t.Fatal(err)
	}
	r.RegisterBlock(&blockTestHandler{command: ".TS"})
	if _, err := r.ParseLine(".TS", "man"); err != nil {
		t.Fatal("line-only API used document handlers", err)
	}
}

type blockLineHandler struct {
	command string
	calls   *atomic.Int64
}

func (h blockLineHandler) CanHandle(cmd, pkg string) bool {
	return normalizedBlockCommand(cmd) == h.command
}
func (h blockLineHandler) Parse(cmd, args, pkg string) (Token, error) {
	if h.calls != nil {
		h.calls.Add(1)
	}
	return TokenText("custom line"), nil
}

func TestBlockRegistrySnapshotAndConcurrency(t *testing.T) {
	for _, astMode := range []bool{false, true} {
		r := NewRegistry()
		h := &blockTestHandler{}
		r.RegisterBlock(h)
		source := ".BOX\n雪\n.END\n"
		reader := &blockRegistrationReader{Reader: strings.NewReader(source), register: func() { r.RegisterBlock(&blockTestHandler{}) }}
		if astMode {
			a, err := ParseAST(reader, WithRegistry(r), WithStrict(true))
			if err != nil || len(a.Diagnostics()) != 0 {
				t.Fatal(err)
			}
		} else {
			d, err := ParseDocument(reader, WithRegistry(r), WithStrict(true))
			if err != nil || len(d.Tokens) != 1 {
				t.Fatal(err)
			}
		}
		if h.parses.Load() != 1 {
			t.Fatal("parse did not retain entry snapshot")
		}
		if _, err := ParseDocument(strings.NewReader(source), WithRegistry(r), WithStrict(true)); !errors.Is(err, ErrAmbiguousHandler) {
			t.Fatal(err)
		}
	}
	r := NewRegistry()
	h := &blockTestHandler{}
	r.RegisterBlock(h)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				d, err := ParseDocument(strings.NewReader(".BOX\nraw\n.END\n"), WithRegistry(r), WithStrict(true))
				if err != nil || len(d.Tokens) != 1 {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if h.parses.Load() != 160 {
		t.Fatal(h.parses.Load())
	}
}

type blockRegistrationReader struct {
	io.Reader
	register func()
	once     sync.Once
}

func (r *blockRegistrationReader) Read(p []byte) (int, error) {
	r.once.Do(r.register)
	return r.Reader.Read(p)
}

func TestBlockFaultsLimitsAndIncludes(t *testing.T) {
	for _, strict := range []bool{false, true} {
		h := &blockTestHandler{}
		reader := &terminalBlockReader{data: ".BOX\nraw", err: errBlockReader}
		d, err := ParseDocument(reader, WithBlockHandler(h), WithRegistry(NewRegistry()), WithStrict(strict))
		if !errors.Is(err, errBlockReader) || d.Tokens[0].RawSource != reader.data || h.parses.Load() != 0 {
			t.Fatal(err)
		}
		for _, option := range []Option{WithMaxBlockDepth(1), WithMaxLines(1), WithMaxLineLength(6), WithDocumentBytes(12)} {
			input := ".BOX\n.BOX\nlong payload\n.END\n.END\n"
			d, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), WithBlockHandler(h), option, WithStrict(strict))
			var limit *DocumentLimitError
			if !errors.As(err, &limit) {
				t.Fatalf("bound ignored: %v", err)
			}
			assertDocumentSource(t, d, input)
		}
	}
	h := &blockTestHandler{}
	resolver := includeResolverFunc(func(string) (io.Reader, error) { return strings.NewReader(".BOX\nraw\n.END\n"), nil })
	d, err := ParseDocument(strings.NewReader(".so child\n"), WithIncludeResolver(resolver), WithRegistry(NewRegistry()), WithBlockHandler(h), WithStrict(true))
	if err != nil || len(d.Includes) != 1 || d.Tokens[0].Block == nil || h.parses.Load() != 1 {
		t.Fatal(err)
	}
	// Blocks are captured without calling line handlers or executing includes.
	r := NewRegistry()
	r.RegisterBlock(&blockTestHandler{})
	var lineCalls atomic.Int64
	r.RegisterCustom(blockLineHandler{command: ".B", calls: &lineCalls})
	input := ".BOX\n.B keep\n.so forbidden\n.END\n"
	_, err = ParseDocument(strings.NewReader(input), WithRegistry(r), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { t.Fatal("include executed inside block"); return nil, nil })), WithStrict(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseAST(strings.NewReader(input), WithRegistry(r), WithStrict(true)); err != nil {
		t.Fatal(err)
	}
	if lineCalls.Load() != 0 {
		t.Fatal("line callback dispatched inside opaque body")
	}

	for _, option := range []Option{WithMaxBlockDepth(1), WithASTNodes(8), WithMaxLines(2), WithMaxLineLength(6)} {
		a, err := ParseAST(strings.NewReader(".BOX\n.BOX\nlong payload\n.END\n.END\n"), WithRegistry(NewRegistry()), WithBlockHandler(&blockTestHandler{}), option)
		if err == nil {
			t.Fatal("AST capture budget ignored")
		}
		for _, b := range a.Root().FindKind(ASTBlock) {
			if source, ok := b.BlockSource(); ok && source.Source() != b.Raw() {
				t.Fatalf("AST partial block lost source: %q vs %q", source.Source(), b.Raw())
			}
		}
	}
}

func FuzzBlockCapture(f *testing.F) {
	for _, s := range []string{".TS\ncell\n.TE\n", ".TS\n.EQ\nx\n.TE\n", ".BOX\n.BOX\nraw\n.END\n.END", ".de \"X\" \"END\"\n雪\n.END", ".ig\nraw", ".PS\ncircle\n.PF\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 16384 {
			t.Skip()
		}
		opts := []Option{WithRegistry(NewRegistry()), WithBlockHandler(&blockTestHandler{}), WithMaxBlockDepth(8)}
		d, _ := ParseDocument(strings.NewReader(input), opts...)
		assertDocumentSource(t, d, input)
		for _, entry := range d.Tokens {
			if entry.Block != nil {
				assertBlockSource(t, *entry.Block, d.Source())
			}
		}
		a, _ := ParseAST(strings.NewReader(input), opts...)
		if a.Source() != input || a.Root().Raw() != input {
			t.Fatal("AST source changed")
		}
		for _, node := range a.Root().FindKind(ASTBlock) {
			if b, ok := node.BlockSource(); ok {
				assertBlockSource(t, *b, input)
				if b.Source() != node.Raw() {
					t.Fatalf("partial capture changed source: %q vs %q", b.Source(), node.Raw())
				}
			}
		}
	})
}

func TestBlockHeadersAndRuleValidation(t *testing.T) {
	for _, input := range []string{`.de X END extra`, `.am X END extra`, `.ig END extra`, `.de "unterminated`, `.de ""`} {
		d, err := ParseDocument(strings.NewReader(input+"\nbody\n"), WithRegistry(NewRegistry()), WithStrict(true))
		if !errors.Is(err, ErrInvalidBlock) || d.Tokens[0].Block != nil || d.Tokens[0].RawSource != input+"\n" {
			t.Fatalf("invalid header consumed body: %#v %v", d, err)
		}
	}
	d, err := ParseDocument(strings.NewReader(".de \"X\" \"END\"\nraw\n.END arguments\n"), WithRegistry(NewRegistry()), WithStrict(true))
	if err != nil {
		t.Fatal(err)
	}
	def := d.Tokens[0].Token.(*TokenMacroDefinition)
	if def.Name != "X" || def.End != "END" || !reflect.DeepEqual(def.Body, []string{"raw"}) || d.Tokens[0].Block.Terminator.Text() != ".END arguments" {
		t.Fatalf("header values lost: %+v", def)
	}
	for _, rule := range []BlockRule{
		{}, {Ends: []string{"END"}}, {Ends: []string{".END", ".END"}},
		{Ends: []string{".END\n"}}, {Ends: []string{".END"}, Nested: []BlockPair{{Start: ".BOX"}}},
		{Ends: []string{".END"}, Nested: []BlockPair{{Start: ".BOX", Ends: []string{".END"}}, {Start: ".BOX", Ends: []string{".STOP"}}}},
	} {
		if err := validateBlockRule(rule); !errors.Is(err, ErrInvalidHandler) {
			t.Fatalf("invalid rule accepted: %+v %v", rule, err)
		}
	}
	h := &blockTestHandler{pkg: "ms"}
	if _, err := ParseDocument(strings.NewReader(".BOX\n"), WithRegistry(NewRegistry()), WithBlockHandler(h), WithPackage("man")); err != nil || h.begins.Load() != 0 {
		t.Fatal("package-scoped handler leaked", err)
	}
}
