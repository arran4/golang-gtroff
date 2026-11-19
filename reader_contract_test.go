package groff

import (
	"bytes"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/tools/txtar"
)

var errInputContract = errors.New("input fixture failed")

type contractReader struct {
	mode  string
	data  []byte
	calls int
	done  bool
	empty int
}

func (r *contractReader) Read(p []byte) (int, error) {
	r.calls++
	if r.done {
		panic("parser retried a terminal reader")
	}
	if len(r.data) == 0 && strings.HasPrefix(r.mode, "prefix-") {
		r.mode = strings.TrimPrefix(r.mode, "prefix-")
	}
	if r.mode == "zero" {
		r.empty++
		if r.empty > 100 {
			panic("empty read guard did not stop")
		}
		return 0, nil
	}
	if r.mode == "negative" {
		r.done = true
		return -1, errInputContract
	}
	if r.mode == "oversized" {
		r.done = true
		return len(p) + 1, errInputContract
	}
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		if r.mode == "data-error" {
			r.done = true
			return n, errInputContract
		}
		if r.mode == "data-eof" && len(r.data) == 0 {
			r.done = true
			return n, io.EOF
		}
		return n, nil
	}
	r.done = true
	return 0, io.EOF
}

type nilContractReader struct{}

func (*nilContractReader) Read([]byte) (int, error) { panic("typed-nil reader called") }
func (*nilContractReader) Close() error             { panic("typed-nil closer called") }

//go:embed testdata/readers/*.txtar
var readerCases embed.FS
var updateReaders = flag.Bool("update-readers", false, "explicitly refresh reader-boundary expectations")

func TestReaderContractScenarios(t *testing.T) {
	paths, err := fs.Glob(readerCases, "testdata/readers/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := readerCases.ReadFile(path)
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
				case "input.roff", "input.hex", "options.json", "expected.txt":
				default:
					t.Fatalf("unknown %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, plain := files["input.roff"]
			encoded, binary := files["input.hex"]
			if plain == binary {
				t.Fatal("exactly one input file required")
			}
			if binary {
				input, err = hex.DecodeString(strings.TrimSpace(string(encoded)))
				if err != nil {
					t.Fatal(err)
				}
			}
			var options struct {
				Mode                   string
				Strict, NoFinalLF      bool
				ASTBytes, Nodes, Lines int
			}
			decoder := json.NewDecoder(bytes.NewReader(files["options.json"]))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&options); err != nil {
				t.Fatal(err)
			}
			if options.NoFinalLF {
				input = bytes.TrimSuffix(input, []byte("\n"))
			}
			opts := []Option{WithRegistry(NewRegistry()), WithSourceName("input.1"), WithStrict(options.Strict), WithASTBytes(options.ASTBytes), WithASTNodes(options.Nodes)}
			if options.Lines != 0 {
				opts = append(opts, WithMaxLines(options.Lines))
			}
			newReader := func() (io.Reader, *contractReader) {
				observed := &contractReader{mode: options.Mode, data: append([]byte(nil), input...)}
				switch options.Mode {
				case "nil":
					return nil, observed
				case "typed-nil":
					return (*nilContractReader)(nil), observed
				}
				return observed, observed
			}
			var got strings.Builder
			r, observed := newReader()
			a, astErr := ParseAST(r, opts...)
			if a == nil {
				t.Fatal("nil partial AST")
			}
			verifyReaderASTSource(t, a)
			if !bytes.HasPrefix(input, []byte(a.Source())) {
				t.Fatal("AST invented input bytes")
			}
			fmt.Fprintf(&got, "ast source=%q complete=%v calls=%d\nerror=%q readerCause=%v noProgress=%v invalidCount=%v\n", a.Source(), a.SourceComplete(), observed.calls, errorString(astErr), errors.Is(astErr, errInputContract), errors.Is(astErr, io.ErrNoProgress), errors.Is(astErr, ErrInvalidReadCount))
			var inputErr *ASTInputError
			if errors.As(astErr, &inputErr) {
				fmt.Fprintf(&got, "input position=%d/%d:%d\n", inputErr.Position.Offset, inputErr.Position.Line, inputErr.Position.Column)
			}
			var limit *ASTSourceLimitError
			if errors.As(astErr, &limit) {
				fmt.Fprintf(&got, "source limit=%d\n", limit.Limit)
			}
			for _, d := range a.Diagnostics() {
				fmt.Fprintf(&got, "ast %s %d:%d %q\n", d.Severity, d.Position.Line, d.Position.Column, d.Message)
			}
			// Byte options above apply only to AST; document limits keep their existing API.
			r, observed = newReader()
			doc, docErr := ParseDocument(r, opts...)
			assertDocumentSource(t, doc, string(input))
			fmt.Fprintf(&got, "document source=%q complete=%v calls=%d\nerror=%q readerCause=%v noProgress=%v invalidCount=%v\n", doc.Source(), doc.SourceComplete, observed.calls, errorString(docErr), errors.Is(docErr, errInputContract), errors.Is(docErr, io.ErrNoProgress), errors.Is(docErr, ErrInvalidReadCount))
			for _, d := range doc.Diagnostics {
				fmt.Fprintf(&got, "document %s %d:%d %q\n", d.Severity, d.Position.Line, d.Position.Column, d.Message)
			}
			if *updateReaders {
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

// Reader/resource failures can coalesce unparsed physical lines into an opaque
// tail. Verify exact traversal and positions without assuming one node per line.
func verifyReaderASTSource(t *testing.T, a *AST) {
	t.Helper()
	source := a.Source()
	if a.Root().Raw() != source {
		t.Fatal("root/source mismatch")
	}
	cursor := 0
	for _, node := range a.Root().Descendants() {
		span := node.Span()
		if span.Start.Offset < 0 || span.End.Offset < span.Start.Offset || span.End.Offset > len(source) {
			t.Fatalf("bad span %+v", span)
		}
		for _, pos := range []Position{span.Start, span.End} {
			prefix := source[:pos.Offset]
			last := strings.LastIndexByte(prefix, '\n')
			if pos.Line != strings.Count(prefix, "\n")+1 || pos.Column != utf8.RuneCountInString(prefix[last+1:])+1 {
				t.Fatalf("reader span disagrees with source: %+v", pos)
			}
		}
		if node.Kind() == ASTLine {
			if span.Start.Offset != cursor || node.Raw() != source[cursor:span.End.Offset] {
				t.Fatal("source traversal has a gap or overlap")
			}
			if strings.Contains(strings.TrimSuffix(node.Raw(), "\n"), "\n") && !node.Attributes().Opaque {
				t.Fatal("multiple physical lines not marked opaque")
			}
			cursor = span.End.Offset
		}
	}
	if cursor != len(source) {
		t.Fatal("reader bytes missing from AST traversal")
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestASTInputErrorComposesWithTreeLimits(t *testing.T) {
	input := ".TS\nraw payload\n.TE\n"
	for _, option := range []Option{WithASTBytes(3), WithASTNodes(1), WithMaxLines(1), WithMaxLineLength(3)} {
		a, err := ParseAST(&contractReader{mode: "data-error", data: []byte(input)}, option)
		if !errors.Is(err, errInputContract) {
			t.Fatalf("reader cause lost with resource failure: %v", err)
		}
		var inputErr *ASTInputError
		if !errors.As(err, &inputErr) || inputErr.Position.Offset != len(a.Source()) {
			t.Fatalf("input position: %v", err)
		}
		verifyReaderASTSource(t, a)
	}
}
func TestReaderProgressAndTerminalCache(t *testing.T) {
	for _, mode := range []string{"negative", "oversized", "data-error", "data-eof", "zero"} {
		raw := &contractReader{mode: mode, data: []byte("bytes")}
		guarded := &checkedSourceReader{reader: raw}
		p := make([]byte, 8)
		var terminal error
		for terminal == nil {
			_, terminal = guarded.Read(p)
		}
		calls := raw.calls
		for i := 0; i < 3; i++ {
			n, err := guarded.Read(p)
			if n != 0 || err != terminal || raw.calls != calls {
				t.Fatal("terminal reader retried")
			}
		}
	}
	// Intermittent empty reads are not EOF and do not exhaust a lifetime quota.
	source := &intermittentContractReader{remaining: 180}
	a, err := ParseAST(source)
	if err != nil || len(a.Source()) != 180 || !a.SourceComplete() {
		t.Fatal("progress did not reset empty-read count", err)
	}
	// Zero-length requests do not call the source or consume progress allowance.
	raw := &contractReader{mode: "zero"}
	guarded := &checkedSourceReader{reader: raw}
	for i := 0; i < 200; i++ {
		if n, err := guarded.Read(nil); n != 0 || err != nil {
			t.Fatal(err)
		}
	}
	if raw.calls != 0 {
		t.Fatal("empty buffer reached reader")
	}
}

type intermittentContractReader struct{ remaining, empties int }

func (r *intermittentContractReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if r.empties < 99 {
		r.empties++
		return 0, nil
	}
	r.empties = 0
	r.remaining--
	p[0] = 'x'
	return 1, nil
}
func TestTypedNilIncludeReaderRejectedBeforeClose(t *testing.T) {
	for _, strict := range []bool{false, true} {
		d, err := ParseDocument(strings.NewReader(".so child\nsuffix\n"), WithRegistry(NewRegistry()), WithStrict(strict), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return (*nilContractReader)(nil), nil })))
		if strict != (err != nil) || len(d.Diagnostics) != 1 || len(d.Includes) != 0 {
			t.Fatalf("typed nil include: %+v %v", d, err)
		}
		if !strings.Contains(d.Diagnostics[0].Message, "nil reader") {
			t.Fatal(d.Diagnostics)
		}
		assertDocumentSource(t, d, ".so child\nsuffix\n")
	}
}
func TestASTObservedEOFFacts(t *testing.T) {
	for _, source := range []string{"", "雪", "line\r\n", ".de X\nbody\n..\n"} {
		for _, mode := range []string{"plain", "data-eof"} {
			a, err := ParseAST(&contractReader{mode: mode, data: []byte(source)})
			if err != nil || a.Source() != source || !a.SourceComplete() {
				t.Fatalf("EOF %s: %q %v", mode, a.Source(), err)
			}
		}
	}
	for _, mode := range []string{"plain", "data-eof"} {
		a, err := ParseAST(&contractReader{mode: mode, data: []byte("abcd")}, WithASTBytes(3))
		var limit *ASTSourceLimitError
		if !errors.As(err, &limit) || a.Source() != "abcd" || a.SourceComplete() != (mode == "data-eof") {
			t.Fatalf("limit EOF fact: %s %v", mode, err)
		}
	}
	maxInt := int(^uint(0) >> 1)
	a, err := ParseAST(strings.NewReader("still present"), WithASTBytes(maxInt))
	if err != nil || a.Source() != "still present" || !a.SourceComplete() {
		t.Fatal(err)
	}
}

func FuzzReaderContract(f *testing.F) {
	for i, source := range []string{".TH T 1\n", ".TS\n雪", ".de X\nbody\n..\n", "\xff\x00\n"} {
		f.Add(source, uint8(i), uint8(8))
	}
	f.Fuzz(func(t *testing.T, source string, mode, limit uint8) {
		if len(source) > 8192 {
			t.Skip()
		}
		modes := []string{"plain", "data-error", "data-eof", "prefix-zero", "prefix-negative", "prefix-oversized", "negative", "oversized"}
		kind := modes[int(mode)%len(modes)]
		r := &contractReader{mode: kind, data: []byte(source)}
		a, err := ParseAST(r, WithASTBytes(int(limit)+1), WithASTNodes(1000))
		if a == nil || !strings.HasPrefix(source, a.Source()) || a.Root().Raw() != a.Source() {
			t.Fatal("AST reader source changed")
		}
		verifyReaderASTSource(t, a)
		if err != nil && a.SourceComplete() && kind != "data-eof" && kind != "plain" {
			t.Fatal("failure claimed EOF")
		}
		d, _ := ParseDocument(&contractReader{mode: kind, data: []byte(source)}, WithRegistry(NewRegistry()), WithDocumentBytes(int(limit)+1))
		assertDocumentSource(t, d, source)
	})
}
