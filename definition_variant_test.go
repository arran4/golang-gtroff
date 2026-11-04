package groff

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDefinitionVariantsCaptureAndEvaluation(t *testing.T) {
	for _, request := range []string{".de1", ".dei", ".dei1", ".am1", ".ami", ".ami1"} {
		t.Run(request, func(t *testing.T) {
			input := request + " NAME\n.nr SIDE 9\n.so forbidden\n.B preserved\n..\nafter\n"
			var lineCalls atomic.Int64
			r := NewRegistry()
			r.RegisterCustom(blockLineHandler{command: ".nr", calls: &lineCalls})
			resolver := includeResolverFunc(func(string) (io.Reader, error) { t.Fatal("body include executed"); return nil, nil })
			d, err := ParseDocument(strings.NewReader(input), WithRegistry(r), WithIncludeResolver(resolver), WithStrict(true))
			if err != nil || len(d.Tokens) != 2 || len(d.Diagnostics) != 0 {
				t.Fatalf("capture: %+v %v", d, err)
			}
			assertDocumentSource(t, d, input)
			v, ok := d.Tokens[0].Token.(*TokenMacroDefinitionVariant)
			if !ok || v.Request != request || v.Name != "NAME" || v.End != "" || v.Delimiter != ".." || len(v.Body) != 3 {
				t.Fatalf("variant metadata: %#v", d.Tokens[0].Token)
			}
			a, err := ParseAST(strings.NewReader(input), WithRegistry(r), WithStrict(true))
			if err != nil || len(a.Root().FindKind(ASTBlock)) != 1 {
				t.Fatal(err)
			}
			b, ok := a.Root().FindKind(ASTBlock)[0].BlockSource()
			if !ok || b.Source() != d.Tokens[0].RawSource {
				t.Fatal("AST/document variant disagree")
			}
			if lineCalls.Load() != 0 {
				t.Fatal("body line dispatched")
			}
			env := NewEnvironment()
			before := v.String()
			out, err := EvaluateContext(context.Background(), []Token{v, d.Tokens[1].Token}, env)
			if err != nil || len(out) != 2 || out[0] != v || out[1].String() != "after" || v.String() != before {
				t.Fatalf("evaluation: %v %v", out, err)
			}
			if len(env.Registers) != 0 || len(env.Macros) != 0 || len(env.Strings) != 0 || len(env.Diagnostics) != 1 {
				t.Fatalf("variant executed: %+v", env)
			}
		})
	}
}

func TestIndirectDelimiterResolution(t *testing.T) {
	for _, request := range []string{".dei", ".dei1", ".ami", ".ami1"} {
		input := request + " NAME REF\n.de1 nested\nraw\n..\n.STOP args\nafter\n"
		for _, astMode := range []bool{false, true} {
			var calls atomic.Int64
			resolver := BlockNameResolverFunc(func(h BlockHeader, reference string) (string, error) {
				calls.Add(1)
				if reference != "REF" || h.Command != request || h.Line.Span.Start.Line != 1 || h.Args != "NAME REF" {
					t.Fatalf("resolver context: %+v %q", h, reference)
				}
				return "STOP", nil
			})
			opts := []Option{WithRegistry(NewRegistry()), WithPackage("man"), WithBlockNameResolver(resolver), WithStrict(true)}
			if astMode {
				a, err := ParseAST(strings.NewReader(input), opts...)
				if err != nil {
					t.Fatal(err)
				}
				first := a.Root().FindKind(ASTBlock)[0]
				source, _ := first.BlockSource()
				if !source.Closed || source.Terminator.Text() != ".STOP args" {
					t.Fatal("resolved AST boundary")
				}
			} else {
				d, err := ParseDocument(strings.NewReader(input), opts...)
				if err != nil || len(d.Tokens) != 2 {
					t.Fatal(err)
				}
				v := d.Tokens[0].Token.(*TokenMacroDefinitionVariant)
				if v.Name != "NAME" || v.End != "REF" || v.Delimiter != ".STOP" {
					t.Fatalf("reference replaced: %+v", v)
				}
			}
			if calls.Load() != 1 {
				t.Fatalf("resolver called %d times", calls.Load())
			}
		}
	}
	// Resolver results are final names. An explicit empty value selects '..'.
	input := ".dei NAME REF\nraw\n..\n"
	d, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), WithBlockNameResolver(BlockNameResolverFunc(func(BlockHeader, string) (string, error) { return "", nil })), WithStrict(true))
	if err != nil || d.Tokens[0].Token.(*TokenMacroDefinitionVariant).Delimiter != ".." {
		t.Fatal(err)
	}
}

func TestUnresolvedIndirectDelimiterIsOpaque(t *testing.T) {
	failure := errors.New("resolver failed")
	for _, mode := range []string{"absent", "nil", "error", "space", "too-long"} {
		t.Run(mode, func(t *testing.T) {
			input := ".ds REF STOP\n.dei NAME REF\n.nr SIDE 9\n.REF\n.STOP\n..\n.de1 NEVER END\nraw\nsuffix\n"
			for _, strict := range []bool{false, true} {
				opts := []Option{WithRegistry(NewRegistry()), WithStrict(strict)}
				switch mode {
				case "nil":
					opts = append(opts, WithBlockNameResolver(BlockNameResolverFunc(nil)))
				case "error":
					opts = append(opts, WithBlockNameResolver(BlockNameResolverFunc(func(BlockHeader, string) (string, error) { return "", failure })))
				case "space":
					opts = append(opts, WithBlockNameResolver(BlockNameResolverFunc(func(BlockHeader, string) (string, error) { return "BAD NAME", nil })))
				case "too-long":
					opts = append(opts, WithBlockNameResolver(BlockNameResolverFunc(func(BlockHeader, string) (string, error) { return strings.Repeat("X", 65537), nil })))
				}
				d, err := ParseDocument(strings.NewReader(input), opts...)
				if strict != (err != nil) || len(d.Diagnostics) != 1 || len(d.Tokens) != 2 || d.Tokens[1].Block == nil {
					t.Fatalf("unresolved: %+v %v", d, err)
				}
				if strict && !errors.Is(err, ErrUnresolvedBlockDelimiter) {
					t.Fatal("unresolved cause lost", err)
				}
				if strict && mode == "error" && !errors.Is(err, failure) {
					t.Fatal("resolver cause lost", err)
				}
				assertDocumentSource(t, d, input)
				source := d.Tokens[1].Block
				if source.Closed || source.Rule.UnresolvedEnd != "REF" || source.Source() != strings.TrimPrefix(input, ".ds REF STOP\n") {
					t.Fatal("guessed indirect delimiter")
				}
				a, err := ParseAST(strings.NewReader(input), opts...)
				if strict != (err != nil) || a.Source() != input || len(a.Diagnostics()) != 1 {
					t.Fatalf("AST unresolved: %v %+v", err, a.Diagnostics())
				}
				if len(a.Root().FindKind(ASTBlock)) != 1 {
					t.Fatal("unresolved body acquired syntax structure")
				}
				node := a.Root().FindKind(ASTBlock)[0]
				b, _ := node.BlockSource()
				if node.Attributes().Closed || node.Attributes().UnresolvedEnd != "REF" || b.Source() != source.Source() {
					t.Fatal("AST guessed boundary")
				}
			}
		})
	}
}

func TestVariantResolverScopeAndBounds(t *testing.T) {
	var calls atomic.Int64
	resolver := BlockNameResolverFunc(func(BlockHeader, string) (string, error) { calls.Add(1); return "STOP", nil })
	r := NewRegistry()
	r.RegisterCustom(blockLineHandler{command: ".dei"})
	if _, err := ParseDocument(strings.NewReader(".dei NAME REF\n"), WithRegistry(r), WithBlockNameResolver(resolver), WithStrict(true)); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("resolver invoked for line override")
	}
	for _, input := range []string{".de1 X\nraw\n..\n", ".dei X\nraw\n..\n", ".de OUT END\n.dei REF UNKNOWN\n.END\n"} {
		if _, err := ParseAST(strings.NewReader(input), WithRegistry(NewRegistry()), WithBlockNameResolver(resolver), WithStrict(true)); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("resolver invoked for direct/default delimiter or opaque nested header")
	}
	for _, option := range []Option{WithDocumentBytes(24), WithMaxLines(2), WithMaxLineLength(12)} {
		input := ".dei X REF\nlong payload line\n.STOP\nremaining\n"
		d, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), option)
		var limit *DocumentLimitError
		if !errors.As(err, &limit) {
			t.Fatalf("unresolved capture budget ignored: %v", err)
		}
		assertDocumentSource(t, d, input)
	}
	reader := &terminalBlockReader{data: ".dei X REF\nraw", err: errBlockReader}
	d, err := ParseDocument(reader, WithRegistry(NewRegistry()))
	if !errors.Is(err, errBlockReader) || d.Source() != reader.data {
		t.Fatal(err)
	}
	// Explicit includes inherit the supplied delimiter resolver, but the body
	// itself is never interpreted as an include or definition.
	d, err = ParseDocument(strings.NewReader(".so child\n"), WithRegistry(NewRegistry()), WithBlockNameResolver(resolver), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return strings.NewReader(".dei X REF\nraw\n.STOP\n"), nil })), WithStrict(true))
	if err != nil || len(d.Includes) != 1 || d.Tokens[0].Block == nil || calls.Load() != 1 {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := ParseDocument(strings.NewReader(".dei X REF\nraw\n.STOP\n"), WithRegistry(NewRegistry()), WithBlockNameResolver(resolver), WithStrict(true)); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 81 {
		t.Fatal(calls.Load())
	}
}

func TestVariantHeaderErrors(t *testing.T) {
	for _, request := range []string{".de1", ".dei", ".dei1", ".am1", ".ami", ".ami1"} {
		for _, args := range []string{"", `X END extra`, `"unterminated`, `""`} {
			d, err := ParseDocument(strings.NewReader(request+" "+args+"\nbody\n"), WithRegistry(NewRegistry()), WithStrict(true))
			if !errors.Is(err, ErrInvalidBlock) || d.Tokens[0].Block != nil {
				t.Fatalf("invalid header: %s %q: %v", request, args, err)
			}
		}
	}
	variant := &TokenMacroDefinitionVariant{Request: ".de1", Name: "X", Delimiter: "..", Body: []string{"one"}}
	data := mustBlockJSON(t, variant)
	want := `{"Request":".de1","Name":"X","End":"","Delimiter":"..","Body":["one"]}`
	if string(data) != want {
		t.Fatalf("variant inspection %s", data)
	}
	if !reflect.DeepEqual(variant.Body, []string{"one"}) {
		t.Fatal("token mutated")
	}
	if text := variant.String(); text != ".de1 X\none\n.." {
		t.Fatal(fmt.Sprintf("variant String: %q", text))
	}
}

func FuzzDefinitionVariantCapture(f *testing.F) {
	f.Add(uint8(0), ".nr SIDE 9\n..\nafter", "STOP", false)
	f.Add(uint8(1), "raw\n.STOP\nafter", "STOP", true)
	f.Add(uint8(3), "raw\n..\n", "", true)
	f.Add(uint8(5), ".BAD\n", "BAD NAME", true)
	requests := []string{".de1", ".dei", ".dei1", ".am1", ".ami", ".ami1"}
	f.Fuzz(func(t *testing.T, kind uint8, body, resolved string, enable bool) {
		if len(body) > 8192 || len(resolved) > 1024 {
			t.Skip()
		}
		input := requests[int(kind)%len(requests)] + " NAME REF\n" + body
		opts := []Option{WithRegistry(NewRegistry()), WithMaxBlockDepth(8)}
		if enable {
			opts = append(opts, WithBlockNameResolver(BlockNameResolverFunc(func(BlockHeader, string) (string, error) { return resolved, nil })))
		}
		d, _ := ParseDocument(strings.NewReader(input), opts...)
		assertDocumentSource(t, d, input)
		for _, entry := range d.Tokens {
			if entry.Block != nil {
				assertBlockSource(t, *entry.Block, d.Source())
			}
		}
		a, _ := ParseAST(strings.NewReader(input), opts...)
		if a.Source() != input || a.Root().Raw() != input {
			t.Fatal("variant AST source changed")
		}
		for _, node := range a.Root().FindKind(ASTBlock) {
			if block, ok := node.BlockSource(); ok {
				assertBlockSource(t, *block, input)
				if block.Source() != node.Raw() {
					t.Fatal("variant boundary lost source")
				}
			}
		}
	})
}
