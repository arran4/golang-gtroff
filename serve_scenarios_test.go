package groff_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/renderhtml"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/serve/*.txtar
var serveCases embed.FS
var updateServe = flag.Bool("update-serve", false, "explicitly update HTTP library scenario expectations")

type scenarioProvider string

func (p scenarioProvider) Search(context.Context, string) ([]manlib.Hit, error) {
	switch p {
	case "unavailable":
		return nil, errors.New("fake man-db unavailable")
	case "hostile":
		return []manlib.Hit{{Name: "external", Section: "1", Description: "<script>PROVIDER</script>", URL: "javascript:alert(1)"}}, nil
	}
	return nil, nil
}

type scenarioFaultFS struct {
	fs.FS
	faults map[string]string
	opened map[string]*scenarioFaultFile
}

func (s *scenarioFaultFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil || s.faults[name] == "" {
		return f, err
	}
	w := &scenarioFaultFile{File: f, mode: s.faults[name]}
	s.opened[name] = w
	return w, nil
}

type scenarioFaultFile struct {
	fs.File
	mode          string
	reads, closes int
}

func (f *scenarioFaultFile) Read(p []byte) (int, error) {
	f.reads++
	switch f.mode {
	case "empty":
		if f.reads > 100 {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, nil
	case "negative":
		return -1, io.ErrUnexpectedEOF
	case "oversized":
		return len(p) + 1, io.ErrUnexpectedEOF
	default:
		panic("unknown fixture read fault")
	}
}

func (f *scenarioFaultFile) Close() error { f.closes++; return f.File.Close() }

func TestServeScenarios(t *testing.T) {
	paths, e := fs.Glob(serveCases, "testdata/serve/*.txtar")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/serve/"), func(t *testing.T) {
			raw, e := serveCases.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			sys := fstest.MapFS{}
			for _, f := range ar.Files {
				if _, exists := files[f.Name]; exists {
					t.Fatal("duplicate archive name")
				}
				files[f.Name] = f.Data
				if strings.HasPrefix(f.Name, "tree/") {
					name := strings.TrimPrefix(f.Name, "tree/")
					if !fs.ValidPath(name) {
						t.Fatal("unsafe fixture path")
					}
					sys[name] = &fstest.MapFile{Data: bytes.Clone(f.Data)}
				}
			}
			var tc struct {
				Gzip            []string
				CorruptGzip     []string
				ReadFaults      map[string]string
				Limits          manlib.Limits
				Error, Provider string
				Requests        []struct {
					Method, Target, Host string
					Status               int
					Contains, Absent     []string
				}
			}
			if e := json.Unmarshal(files["scenario.json"], &tc); e != nil {
				t.Fatal(e)
			}
			for _, name := range tc.Gzip {
				f := sys[name]
				if f == nil {
					t.Fatal("gzip source missing")
				}
				var out bytes.Buffer
				z := gzip.NewWriter(&out)
				_, _ = z.Write(f.Data)
				if e = z.Close(); e != nil {
					t.Fatal(e)
				}
				f.Data = out.Bytes()
			}
			for _, name := range tc.CorruptGzip {
				compressed := false
				for _, generated := range tc.Gzip {
					compressed = compressed || generated == name
				}
				if !compressed {
					t.Fatal("checksum corruption requires a generated gzip fixture")
				}
				sys[name].Data[len(sys[name].Data)-8] ^= 0xff
			}
			faults := &scenarioFaultFS{FS: sys, faults: tc.ReadFaults, opened: map[string]*scenarioFaultFile{}}
			for name, mode := range tc.ReadFaults {
				if sys[name] == nil || (mode != "empty" && mode != "negative" && mode != "oversized") {
					t.Fatal("invalid read-fault fixture", name, mode)
				}
			}
			idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "fixture", FS: faults}}, tc.Limits)
			for name, f := range faults.opened {
				wantReads := 1
				if f.mode == "empty" {
					wantReads = 100
				}
				if f.closes != 1 || f.reads != wantReads {
					t.Fatalf("%s: reads=%d want=%d closes=%d want=1", name, f.reads, wantReads, f.closes)
				}
			}
			if tc.Error != "" {
				if err == nil || !strings.Contains(err.Error(), tc.Error) {
					t.Fatalf("want %q got %v", tc.Error, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			opts := manlib.HandlerOptions{Render: renderhtml.Options{Assets: renderhtml.OmitFonts}}
			if tc.Provider != "" {
				opts.SearchProvider = scenarioProvider(tc.Provider)
			}
			h, e := manlib.Handler(idx, opts)
			if e != nil {
				t.Fatal(e)
			}
			actual := map[string][]byte{}
			type response struct {
				Method, Target string
				Status         int
				Headers        map[string][]string
				Body           string
			}
			var responses []response
			for n, q := range tc.Requests {
				method := q.Method
				if method == "" {
					method = "GET"
				}
				req := httptest.NewRequest(method, q.Target, nil)
				req.Host = q.Host
				if req.Host == "" {
					req.Host = "localhost:8080"
				}
				out := httptest.NewRecorder()
				h.ServeHTTP(out, req)
				if out.Code != q.Status {
					t.Fatalf("%s %s: status=%d want=%d body=%s", method, q.Target, out.Code, q.Status, out.Body.String())
				}
				for _, s := range q.Contains {
					if !strings.Contains(out.Body.String(), s) {
						t.Errorf("%s body missing %q", q.Target, s)
					}
				}
				for _, s := range q.Absent {
					if strings.Contains(out.Body.String(), s) {
						t.Errorf("%s body contains forbidden %q", q.Target, s)
					}
				}
				name := fmt.Sprintf("expected/response-%03d.body", n+1)
				actual[name] = bytes.Clone(out.Body.Bytes())
				responses = append(responses, response{method, q.Target, out.Code, out.Header(), name})
			}
			entries, _ := json.MarshalIndent(idx.Entries(), "", "  ")
			notices, _ := json.MarshalIndent(idx.Notices(), "", "  ")
			responseJSON, _ := json.MarshalIndent(responses, "", "  ")
			actual["expected/entries.json"] = append(entries, '\n')
			actual["expected/notices.json"] = append(notices, '\n')
			actual["expected/responses.json"] = append(responseJSON, '\n')
			if *updateServe {
				next := &txtar.Archive{Comment: ar.Comment}
				for _, f := range ar.Files {
					if !strings.HasPrefix(f.Name, "expected/") {
						next.Files = append(next.Files, f)
					}
				}
				var names []string
				for name := range actual {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					next.Files = append(next.Files, txtar.File{Name: name, Data: actual[name]})
				}
				if e := os.WriteFile(path, txtar.Format(next), 0644); e != nil {
					t.Fatal(e)
				}
			} else {
				for name, got := range actual {
					if want, ok := files[name]; !ok || !bytes.Equal(want, got) {
						t.Errorf("%s mismatch; refresh explicitly with -update-serve\n%s", name, got)
					}
				}
			}
		})
	}
}

func TestServeSyntheticExample(t *testing.T) {
	source, e := htmlCases.ReadFile("examples/man-tool.1")
	if e != nil {
		t.Fatal(e)
	}
	idx, e := manlib.Build(context.Background(), []manlib.Root{{Label: "example", FS: fstest.MapFS{"man1/man-tool.1": {Data: source}}}}, manlib.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	h, e := manlib.Handler(idx, manlib.HandlerOptions{})
	if e != nil {
		t.Fatal(e)
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/page/1/man-tool", nil))
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	for _, s := range []string{"data:font/ttf;base64,", "Display verbose diagnostics", "No automatic includes or external commands.", "#man-node-", "User Commands"} {
		if !strings.Contains(out.Body.String(), s) {
			t.Fatal("synthetic example missing", s)
		}
	}
}
