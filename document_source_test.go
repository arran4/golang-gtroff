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
	"testing"
	"unicode/utf8"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/document/*.txtar
var documentSourceCases embed.FS
var updateDocument = flag.Bool("update-document", false, "explicitly refresh document source/span expectations")

func TestDocumentSourceScenarios(t *testing.T) {
	paths, err := fs.Glob(documentSourceCases, "testdata/document/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := documentSourceCases.ReadFile(path)
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
				case "input.1", "options.json", "error.txt", "expected.txt":
				default:
					t.Fatalf("unknown fixture %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, ok := files["input.1"]
			if !ok {
				t.Fatal("missing input.1")
			}
			var options struct {
				Strict, CRLF, NoFinalLF     bool
				MaxLines, MaxLine, MaxBytes int
			}
			if data, ok := files["options.json"]; ok {
				if err := json.Unmarshal(data, &options); err != nil {
					t.Fatal(err)
				}
			}
			if options.CRLF {
				input = bytes.ReplaceAll(input, []byte("\n"), []byte("\r\n"))
			}
			if options.NoFinalLF {
				input = bytes.TrimSuffix(bytes.TrimSuffix(input, []byte("\n")), []byte("\r"))
			}
			opts := []Option{WithRegistry(NewRegistry()), WithPackage("man"), WithSourceName("input.1"), WithStrict(options.Strict)}
			if options.MaxLines > 0 {
				opts = append(opts, WithMaxLines(options.MaxLines))
			}
			if options.MaxLine > 0 {
				opts = append(opts, WithMaxLineLength(options.MaxLine))
			}
			if options.MaxBytes > 0 {
				opts = append(opts, WithDocumentBytes(options.MaxBytes))
			}
			doc, parseErr := ParseDocument(bytes.NewReader(input), opts...)
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && parseErr != nil {
				t.Fatal(parseErr)
			}
			if wantErr != "" && (parseErr == nil || !strings.Contains(parseErr.Error(), wantErr)) {
				t.Fatalf("want error containing %q, got %v", wantErr, parseErr)
			}
			assertDocumentSource(t, doc, string(input))
			var got strings.Builder
			fmt.Fprintf(&got, "sourceComplete=%v bytes=%d\n", doc.SourceComplete, len(doc.Source()))
			for _, token := range doc.Tokens {
				fmt.Fprintf(&got, "%s [%d,%d) %d:%d-%d:%d raw=%q canonical=%q\n", reflect.TypeOf(token.Token), token.Span.Start.Offset, token.Span.End.Offset, token.Span.Start.Line, token.Span.Start.Column, token.Span.End.Line, token.Span.End.Column, token.RawSource, token.Token.String())
			}
			for _, diagnostic := range doc.Diagnostics {
				fmt.Fprintf(&got, "%s %d:%d %s\n", diagnostic.Severity, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Message)
			}
			if *updateDocument {
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
				t.Fatalf("document:\n%s\nwant:\n%s", got.String(), files["expected.txt"])
			}
		})
	}
}

func assertDocumentSource(t *testing.T, doc *Document, input string) {
	t.Helper()
	if doc == nil {
		t.Fatal("nil partial document")
	}
	if !strings.HasPrefix(input, doc.Source()) {
		t.Fatalf("source bytes changed: %q vs %q", doc.Source(), input)
	}
	if doc.SourceComplete && doc.Source() != input {
		t.Fatal("complete source lost input")
	}
	var combined strings.Builder
	for _, token := range doc.Tokens {
		if token.Token == nil {
			t.Fatal("nil token")
		}
		span := token.Span
		if span.Start.Offset != combined.Len() || span.End.Offset > len(doc.Source()) || span.End.Offset-span.Start.Offset != len(token.RawSource) {
			t.Fatalf("inconsistent source span: %+v raw=%q", span, token.RawSource)
		}
		if doc.Source()[span.Start.Offset:span.End.Offset] != token.RawSource {
			t.Fatal("token source differs from document source")
		}
		for _, pos := range []Position{span.Start, span.End} {
			prefix := doc.Source()[:pos.Offset]
			line := strings.Count(prefix, "\n") + 1
			last := strings.LastIndexByte(prefix, '\n')
			column := utf8.RuneCountInString(prefix[last+1:]) + 1
			if pos.Filename != doc.SourceName || pos.Line != line || pos.Column != column {
				t.Fatalf("wrong source position %+v; want %s:%d:%d", pos, doc.SourceName, line, column)
			}
		}
		combined.WriteString(token.RawSource)
	}
	if combined.String() != doc.Source() {
		t.Fatal("tokens dropped captured physical source")
	}
}

func TestDocumentReaderFailuresAndBufferedTail(t *testing.T) {
	failure := errors.New("terminal input failure")
	for _, input := range []string{"", "雪", "line\n", ".so child", ".TS\nbody", ".de X\nbody", ".ig\nbody", ".B \"bad\nretained tail\n"} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/strict=%v", input, strict), func(t *testing.T) {
				reader := &terminalBlockReader{data: input, err: failure}
				calls := 0
				resolver := includeResolverFunc(func(string) (io.Reader, error) { calls++; return strings.NewReader("unexpected"), nil })
				doc, err := ParseDocument(reader, WithStrict(strict), WithSourceName("fault.1"), WithIncludeResolver(resolver))
				if !errors.Is(err, failure) {
					t.Fatalf("reader error identity lost: %v", err)
				}
				var inputErr *DocumentInputError
				if !errors.As(err, &inputErr) {
					t.Fatalf("input failure not classified: %v", err)
				}
				assertDocumentSource(t, doc, input)
				if doc.Source() != input || doc.SourceComplete || len(doc.Diagnostics) == 0 {
					t.Fatalf("partial source/diagnostics lost: %+v", doc)
				}
				if input == ".so child" && calls != 0 {
					t.Fatal("executed incomplete include header after terminal failure")
				}
			})
		}
	}
}

func TestDocumentLimitsAndExactBoundaries(t *testing.T) {
	for _, input := range []string{"one\n", "one", "雪\r\n"} {
		doc, err := ParseDocument(strings.NewReader(input), WithMaxLines(1), WithDocumentBytes(len(input)))
		if err != nil || !doc.SourceComplete {
			t.Fatalf("exact boundary rejected: %q %v", input, err)
		}
		assertDocumentSource(t, doc, input)
	}
	for _, option := range []Option{WithDocumentBytes(4), WithMaxLines(1), WithMaxLineLength(3)} {
		doc, err := ParseDocument(strings.NewReader("long input\nnext\n"), option)
		var inputErr *DocumentInputError
		var limit *DocumentLimitError
		if !errors.As(err, &inputErr) || !errors.As(err, &limit) {
			t.Fatalf("resource failure not classified: %v", err)
		}
		assertDocumentSource(t, doc, "long input\nnext\n")
	}
	reader := &countingDocumentReader{remaining: 1 << 20}
	doc, err := ParseDocument(reader, WithMaxLineLength(16))
	if err == nil || reader.read > 4096 || len(doc.Source()) != reader.read {
		t.Fatalf("line bound did not stop reader: read=%d err=%v", reader.read, err)
	}
	reader = &countingDocumentReader{remaining: 1 << 20}
	doc, err = ParseDocument(reader, WithDocumentBytes(16), WithMaxLineLength(0))
	if err == nil || reader.read != 17 || len(doc.Source()) != 17 {
		t.Fatalf("byte bound did not stop reader: read=%d err=%v", reader.read, err)
	}
}

type countingDocumentReader struct{ remaining, read int }

func (r *countingDocumentReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = 'x'
	}
	r.remaining -= len(p)
	r.read += len(p)
	return len(p), nil
}

func TestDocumentIncludedSourcesAndSharedBudget(t *testing.T) {
	input := ".so child\nparent tail\n"
	childSource := ".de X\r\n雪\r\n..\r\n"
	child := &observedInclude{Reader: strings.NewReader(childSource)}
	doc, err := ParseDocument(strings.NewReader(input), WithSourceName("parent"), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return child, nil })))
	if err != nil || len(doc.Includes) != 1 || child.closed != 1 || doc.Source() != input {
		t.Fatalf("include source/lifetime: doc=%+v err=%v", doc, err)
	}
	included := doc.Includes[0]
	if included.Directive.Start.Offset != 0 || included.Directive.End.Offset != len(".so child\n") || included.Document.SourceName != "child" {
		t.Fatalf("include provenance: %+v", included)
	}
	assertDocumentSource(t, included.Document, childSource)
	if included.Document.Source() != childSource {
		t.Fatal("included CRLF source changed")
	}
	large := &countingDocumentReader{remaining: 1 << 20}
	limited, err := ParseDocument(strings.NewReader(".so child\n"), WithDocumentBytes(32), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return large, nil })))
	var limit *DocumentLimitError
	if !errors.As(err, &limit) || limited == nil || len(limited.Includes) != 1 {
		t.Fatalf("include escaped global budget: %v %+v", err, limited)
	}
	if len(limited.Source())+len(limited.Includes[0].Document.Source()) != 33 || large.read != 23 {
		t.Fatalf("aggregate reads not bounded: root=%d child=%d", len(limited.Source()), large.read)
	}
	failure := errors.New("included input failure")
	failedChild := &observedInclude{Reader: &terminalBlockReader{data: "child partial", err: failure}}
	partial, err := ParseDocument(strings.NewReader(input), WithSourceName("parent"), WithIncludeResolver(includeResolverFunc(func(string) (io.Reader, error) { return failedChild, nil })))
	if !errors.Is(err, failure) || partial == nil || len(partial.Includes) != 1 || failedChild.closed != 1 {
		t.Fatalf("included failure/lifetime lost: %+v %v", partial, err)
	}
	assertDocumentSource(t, partial, input)
	assertDocumentSource(t, partial.Includes[0].Document, "child partial")
	if partial.Includes[0].Document.Source() != "child partial" {
		t.Fatal("failed include source discarded")
	}
}

func TestDocumentDataAndEOFAndNoProgress(t *testing.T) {
	for _, source := range []string{"", "雪", "one\n", ".de X\nbody\n..\n"} {
		doc, err := ParseDocument(&terminalBlockReader{data: source, err: io.EOF})
		if err != nil || !doc.SourceComplete {
			t.Fatalf("data plus EOF: %q %v", source, err)
		}
		assertDocumentSource(t, doc, source)
	}
	reader := &documentNoProgressReader{}
	doc, err := ParseDocument(reader)
	if !errors.Is(err, io.ErrNoProgress) || doc == nil || reader.calls > 100 {
		t.Fatalf("unbounded empty reads: calls=%d err=%v", reader.calls, err)
	}
}

type documentNoProgressReader struct{ calls int }

func (r *documentNoProgressReader) Read([]byte) (int, error) { r.calls++; return 0, nil }

func TestDocumentChunkBoundaries(t *testing.T) {
	source := ".de\tX END\r\n雪 hé\r\n.END\r\n\r\nlast 雪"
	baseline, err := ParseDocument(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	for chunk := 1; chunk <= 9; chunk++ {
		doc, err := ParseDocument(&documentChunkReader{source: strings.NewReader(source), chunk: chunk})
		if err != nil || !reflect.DeepEqual(doc.Tokens, baseline.Tokens) || !doc.SourceComplete {
			t.Fatalf("chunk=%d changed tokenization: %v", chunk, err)
		}
		assertDocumentSource(t, doc, source)
	}
}

type documentChunkReader struct {
	source io.Reader
	chunk  int
}

func (r *documentChunkReader) Read(p []byte) (int, error) {
	if len(p) > r.chunk {
		p = p[:r.chunk]
	}
	return r.source.Read(p)
}

func FuzzDocumentSource(f *testing.F) {
	for _, source := range []string{".TH DEMO 1\n", ".de X\r\n雪\r\n..\r\n", ".TS\nunterminated", ".B \"bad\nrest\n", "\xff\x00\r"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 8192 {
			return
		}
		for _, strict := range []bool{false, true} {
			doc, _ := ParseDocument(strings.NewReader(source), WithRegistry(NewRegistry()), WithStrict(strict), WithMaxLines(32), WithMaxLineLength(1024), WithDocumentBytes(4096))
			assertDocumentSource(t, doc, source)
		}
	})
}
