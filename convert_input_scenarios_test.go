package groff_test

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

	groff "github.com/arran4/golang-gtroff"
	conversion "github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/renderhtml"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/convert-input/*.txtar
var convertInputCases embed.FS
var updateConvertInput = flag.Bool("update-convert-input", false, "explicitly refresh conversion input-error expectations")
var errConvertInput = errors.New("input fixture failed")

type conversionInputReader struct {
	mode                 string
	data                 []byte
	calls, empty, closed int
	done                 bool
}

func (r *conversionInputReader) Read(p []byte) (int, error) {
	r.calls++
	if r.done {
		panic("conversion retried terminal reader")
	}
	if len(r.data) == 0 && strings.HasPrefix(r.mode, "prefix-") {
		r.mode = strings.TrimPrefix(r.mode, "prefix-")
	}
	switch r.mode {
	case "zero":
		r.empty++
		if r.empty > 100 {
			panic("empty reads not bounded")
		}
		return 0, nil
	case "negative":
		r.done = true
		return -1, errConvertInput
	case "oversized":
		r.done = true
		return len(p) + 1, errConvertInput
	}
	if len(r.data) > 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		if r.mode == "data-error" {
			r.done = true
			return n, errConvertInput
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
func (r *conversionInputReader) Close() error { r.closed++; return nil }

type conversionNilReader struct{}

func (*conversionNilReader) Read([]byte) (int, error) { panic("typed-nil reader called") }
func (*conversionNilReader) Close() error             { panic("typed-nil closer called") }

type conversionCountWriter struct{ calls int }

func (w *conversionCountWriter) Write(p []byte) (int, error) { w.calls++; return len(p), nil }

func TestConvertInputScenarios(t *testing.T) {
	paths, err := fs.Glob(convertInputCases, "testdata/convert-input/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/convert-input/"), func(t *testing.T) {
			raw, err := convertInputCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, file := range archive.Files {
				if _, exists := files[file.Name]; exists {
					t.Fatal("duplicate archive member")
				}
				switch file.Name {
				case "input.txt", "input.hex", "options.json", "expected.txt":
				default:
					t.Fatal("unknown archive member", file.Name)
				}
				files[file.Name] = file.Data
			}
			source, plain := files["input.txt"]
			encoded, binary := files["input.hex"]
			if plain == binary {
				t.Fatal("exactly one input file required")
			}
			if binary {
				source, err = hex.DecodeString(strings.TrimSpace(string(encoded)))
				if err != nil {
					t.Fatal(err)
				}
			}
			var options struct {
				Mode          string
				MaxInputBytes int
				Strict        bool
				NoFinalLF     bool
			}
			decoder := json.NewDecoder(bytes.NewReader(files["options.json"]))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&options); err != nil {
				t.Fatal(err)
			}
			if options.NoFinalLF {
				source = bytes.TrimSuffix(source, []byte("\n"))
			}
			newReader := func() (io.Reader, *conversionInputReader) {
				observed := &conversionInputReader{mode: options.Mode, data: bytes.Clone(source)}
				switch options.Mode {
				case "nil":
					return nil, observed
				case "typed-nil":
					return (*conversionNilReader)(nil), observed
				}
				return observed, observed
			}
			o := conversion.Options{MaxInputBytes: options.MaxInputBytes, SourceName: "input.txt", Strict: options.Strict, HTML: renderhtml.Options{Assets: renderhtml.OmitFonts}}
			var got strings.Builder
			for _, kind := range []string{"html", "roff"} {
				r, observed := newReader()
				var in conversion.Input
				var inputErr error
				if kind == "html" {
					in, inputErr = conversion.ImportHTML(r, o)
				} else {
					in, inputErr = conversion.ImportMan(r, o)
				}
				if !bytes.HasPrefix(source, []byte(in.Source)) || observed.closed != 0 {
					t.Fatal("input changed or borrowed reader closed")
				}
				message := ""
				if inputErr != nil {
					message = inputErr.Error()
				}
				fmt.Fprintf(&got, "%s source=%q document=%v calls=%d\nerror=%q readerCause=%v limit=%v noProgress=%v invalidCount=%v\n", kind, in.Source, in.Document != nil, observed.calls, message, errors.Is(inputErr, errConvertInput), errors.Is(inputErr, conversion.ErrLimit), errors.Is(inputErr, io.ErrNoProgress), errors.Is(inputErr, groff.ErrInvalidReadCount))
				var positioned *conversion.InputError
				if errors.As(inputErr, &positioned) {
					if positioned.Position.Offset != len(in.Source) || in.Document != nil {
						t.Fatal("input failure interpreted partial source")
					}
					fmt.Fprintf(&got, "position=%d/%d:%d\n", positioned.Position.Offset, positioned.Position.Line, positioned.Position.Column)
				}
				var count *groff.ReaderCountError
				if errors.As(inputErr, &count) && fmt.Sprintf("%T", count) != "*groff.ReaderCountError" {
					t.Fatal("public count-error type changed")
				}
				r, observed = newReader()
				writer := &conversionCountWriter{}
				var adapterErr error
				if kind == "html" {
					_, adapterErr = conversion.HTMLToMan(r, writer, o)
				} else {
					_, adapterErr = conversion.ManToHTML(r, writer, o)
				}
				if (adapterErr != nil && writer.calls != 0) || observed.closed != 0 {
					t.Fatal("failed conversion wrote or closed borrowed input")
				}
				fmt.Fprintf(&got, "adapterError=%v wrote=%v\n", adapterErr != nil, writer.calls > 0)
			}
			if *updateConvertInput {
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

func FuzzConvertInput(f *testing.F) {
	for i, source := range []string{"<p>kept</p>\n", "雪\n", "\xff\r\n", ".SH NAME\nbody\n", ""} {
		f.Add(source, uint8(i), uint8(16))
	}
	f.Fuzz(func(t *testing.T, source string, mode, limit uint8) {
		if len(source) > 2048 {
			t.Skip()
		}
		modes := []string{"data-error", "prefix-zero", "prefix-negative", "prefix-oversized", "negative", "oversized", "plain", "data-eof"}
		o := conversion.Options{MaxInputBytes: int(limit) + 1, Strict: mode%2 == 1, HTML: renderhtml.Options{Assets: renderhtml.OmitFonts}}
		newReader := func() *conversionInputReader {
			return &conversionInputReader{mode: modes[int(mode)%len(modes)], data: []byte(source)}
		}
		for _, html := range []bool{false, true} {
			r := newReader()
			var in conversion.Input
			var err error
			if html {
				in, err = conversion.ImportHTML(r, o)
			} else {
				in, err = conversion.ImportMan(r, o)
			}
			if !strings.HasPrefix(source, in.Source) || len(in.Source) > o.MaxInputBytes+1 || r.closed != 0 {
				t.Fatal("source/ownership contract changed")
			}
			var positioned *conversion.InputError
			if errors.As(err, &positioned) && (in.Document != nil || positioned.Position.Offset != len(in.Source)) {
				t.Fatal("failed input interpreted or mispositioned")
			}
			writer := &conversionCountWriter{}
			r = newReader()
			if html {
				_, err = conversion.HTMLToMan(r, writer, o)
			} else {
				_, err = conversion.ManToHTML(r, writer, o)
			}
			if err != nil && writer.calls != 0 {
				t.Fatal("failed conversion wrote output")
			}
			if r.closed != 0 {
				t.Fatal("borrowed reader closed")
			}
		}
	})
}
