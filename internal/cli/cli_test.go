package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
)

func runTest(args []string, input string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(args, Environment{In: strings.NewReader(input), Out: &out, Err: &errOut, Build: BuildInfo{Version: "test", Commit: "abc123", Date: "2026-10-21"}})
	return code, out.String(), errOut.String()
}
func TestValidationPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, input, status string
		strict              bool
		code                int
	}{
		{"valid", ".TH DEMO 1\n.SH NAME\ndemo\n", "valid", false, ExitOK},
		{"fragment", ".B fragment\n", "valid", false, ExitOK},
		{"unknown", ".CUSTOM text\n", "partial", false, ExitOK},
		{"strict unknown", ".CUSTOM text\n", "invalid", true, ExitInvalid},
		{"malformed permissive", ".B \"broken\n", "invalid", false, ExitInvalid},
		{"malformed strict", ".B \"broken\n", "invalid", true, ExitInvalid},
		{"include", ".so /etc/passwd\n", "partial", false, ExitOK},
		{"include strict", ".so /etc/passwd\n", "invalid", true, ExitInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"validate", "--format", "json"}
			if tc.strict {
				args = append(args, "--strict")
			}
			code, out, stderr := runTest(args, tc.input)
			if code != tc.code || stderr != "" {
				t.Fatalf("code=%d stderr=%s", code, stderr)
			}
			var v validationResult
			if err := json.Unmarshal([]byte(out), &v); err != nil {
				t.Fatal(err)
			}
			if v.Status != tc.status || v.Assessment != "syntax-only" || v.SchemaVersion != 1 {
				t.Fatalf("incorrect assessment: %+v", v)
			}
		})
	}
}

func TestUsageDoesNotOpenOrReadInput(t *testing.T) {
	for _, args := range [][]string{
		{"validate", "--format", "yaml"}, {"validate", "--package", "invalid"}, {"validate", "--select", "node:1"},
		{"validate", "a", "b"}, {"ast", "--select", "node:0"}, {"ast", "--select", "node:no"}, {"ast", "--select", "node:18446744073709551616"},
		{"ast", "--select", "path:/-1"}, {"ast", "--select", "path:abc"}, {"ast", "--select", "path:/0/"}, {"ast", "--select", "wat:x"},
		{"ast", "--select", "section:"}, {"ast", "--unknown"}, {"render"}, {"version", "a"}, {"help", "a", "b"}, {"help", "absent"}, {"--help", "extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, stderr bytes.Buffer
			code := Run(args, Environment{In: panicReader{}, Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { t.Fatal("opened input on invalid usage"); return nil, nil }})
			if code != ExitUsage || out.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("usage code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
			}
		})
	}
	for _, args := range [][]string{{}, {"help"}, {"--help"}, {"-h"}, {"help", "ast"}, {"help", "validate"}, {"help", "version"}, {"version"}, {"--version"}, {"ast", "-h"}, {"validate", "--help"}} {
		var out bytes.Buffer
		if code := Run(args, Environment{In: panicReader{}, Out: &out}); code != ExitOK || out.Len() == 0 {
			t.Fatalf("help/version code %d", code)
		}
	}
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("input read unexpectedly") }

var errBoundary = errors.New("boundary failed")

type boundaryReader struct {
	data     string
	sent     bool
	closeErr error
	closed   bool
}

func (r *boundaryReader) Read(p []byte) (int, error) {
	if r.closed {
		panic("read after close")
	}
	if r.sent {
		return 0, errBoundary
	}
	r.sent = true
	return copy(p, r.data), errBoundary
}
func (r *boundaryReader) Close() error { r.closed = true; return r.closeErr }

type closeReader struct {
	*strings.Reader
	closed bool
	err    error
}

func (r *closeReader) Close() error { r.closed = true; return r.err }

func TestIOFailureAndOwnership(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(string(rune('0'+boolInt(strict))), func(t *testing.T) {
			r := &boundaryReader{data: ".B \"broken\nrest"}
			var out, stderr bytes.Buffer
			args := []string{"ast", "--format", "json"}
			if strict {
				args = append(args, "--strict")
			}
			args = append(args, "source.1")
			code := Run(args, Environment{In: panicReader{}, Out: &out, Err: &stderr, Open: func(string) (io.ReadCloser, error) { return r, nil }})
			if code != ExitIO || !r.closed || stderr.Len() != 0 {
				t.Fatalf("reader failure lost: code=%d closed=%v stderr=%s", code, r.closed, stderr.String())
			}
			var v astResult
			if err := json.Unmarshal(out.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
			if v.Source != r.data || v.Status != "incomplete" || !strings.Contains(v.Error, errBoundary.Error()) {
				t.Fatalf("partial output %+v", v)
			}
		})
	}
	for _, closeErr := range []error{nil, errBoundary} {
		r := &closeReader{Reader: strings.NewReader(".B hello\n"), err: closeErr}
		var out bytes.Buffer
		code := Run([]string{"validate", "--format", "json", "source.1"}, Environment{Out: &out, Open: func(string) (io.ReadCloser, error) { return r, nil }})
		if !r.closed {
			t.Fatal("named input leaked")
		}
		if closeErr != nil && code != ExitIO {
			t.Fatal("close error ignored")
		}
		if closeErr == nil && code != ExitOK {
			t.Fatal("valid input rejected")
		}
	}
	// stdin belongs to the caller and must remain open.
	stdin := &closeReader{Reader: strings.NewReader("hello")}
	if code := Run([]string{"validate"}, Environment{In: stdin}); code != ExitOK || stdin.closed {
		t.Fatal("stdin ownership violated")
	}
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errBoundary }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }
func TestOutputFailures(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"version"}, {"validate"}, {"validate", "--format", "json"}, {"ast"}, {"ast", "--format", "text"}} {
		for _, writer := range []io.Writer{errorWriter{}, shortWriter{}} {
			if code := Run(args, Environment{In: strings.NewReader(".B hello\n"), Out: writer}); code != ExitIO {
				t.Fatalf("writer error ignored: %v code=%d", args, code)
			}
		}
	}
	if code := Run([]string{"validate"}, Environment{In: strings.NewReader(".CUSTOM\n"), Err: errorWriter{}}); code != ExitIO {
		t.Fatal("diagnostic error ignored")
	}
	if code := Run([]string{"bad"}, Environment{Err: shortWriter{}}); code != ExitIO {
		t.Fatal("usage writer error ignored")
	}
}

func TestASTSelectionMatchesSourceTree(t *testing.T) {
	input := ".TH DEMO 1\n.SH NAME\nfirst\n.SS NAME\nnested\n.SH OTHER\noutside\n.SH NAME\nsecond\n"
	tree, err := groff.ParseAST(strings.NewReader(input), groff.WithPackage("man"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		selector string
		roots    []groff.NodeID
	}{
		{"node:1", []groff.NodeID{1}}, {"path:/", []groff.NodeID{1}}, {"path:/0", []groff.NodeID{2}}, {"node:2", []groff.NodeID{2}},
	}
	var repeated []groff.NodeID
	for _, n := range tree.Root().FindSection("NAME", false) {
		repeated = append(repeated, n.ID())
	}
	cases = append(cases, struct {
		selector string
		roots    []groff.NodeID
	}{"section:NAME", repeated})
	for _, tc := range cases {
		code, out, stderr := runTest([]string{"ast", "--select", tc.selector}, input)
		if code != ExitOK || stderr != "" {
			t.Fatalf("%s: code=%d stderr=%s", tc.selector, code, stderr)
		}
		var v astResult
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatal(err)
		}
		if len(v.RootIDs) != len(tc.roots) {
			t.Fatal("wrong root count")
		}
		for i, id := range tc.roots {
			if v.RootIDs[i] != id {
				t.Fatal("source IDs changed")
			}
		}
		seen := map[groff.NodeID]bool{}
		var previous groff.NodeID
		for _, row := range v.Nodes {
			if seen[row.ID] || row.ID <= previous {
				t.Fatal("duplicate/reordered selected node")
			}
			seen[row.ID] = true
			previous = row.ID
			n := tree.Node(row.ID)
			if n == nil || row.Raw != n.Raw() {
				t.Fatal("node raw mismatch")
			}
		}
		for _, id := range v.ExternalParentIDs {
			if seen[id] {
				t.Fatal("external parent is inside selected tree")
			}
		}
	}
	for _, sel := range []string{"node:999999", "path:/999999", "section:absent"} {
		code, out, _ := runTest([]string{"ast", "--select", sel}, input)
		if code != ExitUsage || out != "" {
			t.Fatal("empty selection produced output")
		}
	}
}
func TestInvalidUTF8IsByteRecoverableInJSON(t *testing.T) {
	input := string([]byte{'x', 0xff, '\n'})
	code, out, _ := runTest([]string{"ast"}, input)
	if code != ExitInvalid {
		t.Fatal("invalid UTF-8 not diagnosed")
	}
	var v astResult
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	recovered, err := base64.StdEncoding.DecodeString(v.SourceBase64)
	if err != nil || string(recovered) != input {
		t.Fatal("original invalid source lost")
	}
	if len(v.Nodes) < 2 || v.Nodes[1].RawBase64 == "" {
		t.Fatal("node raw bytes missing")
	}
}
func TestExportBudgetLeavesStdoutEmpty(t *testing.T) {
	// Multiple selected ancestor spans would otherwise multiply the source size.
	input := strings.Repeat(".RS\n", 40) + strings.Repeat("x", 1<<20) + "\n" + strings.Repeat(".RE\n", 40)
	code, out, stderr := runTest([]string{"ast"}, input)
	if code != ExitIO || out != "" || !strings.Contains(stderr, "budget") {
		t.Fatalf("export budget code=%d output bytes=%d stderr=%s", code, len(out), stderr)
	}
}
func FuzzCLIInspection(f *testing.F) {
	for _, s := range []string{"", ".B \"broken\n", ".CUSTOM\n", ".TS\n雪", "hello\xff", ".so /etc/passwd\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 8192 {
			t.Skip()
		}
		code, out, stderr := runTest([]string{"ast", "--strict"}, input)
		if code != ExitOK && code != ExitInvalid {
			t.Fatalf("unexpected exit %d: %s", code, stderr)
		}
		var v astResult
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatal(err)
		}
		if v.SchemaVersion != 1 || len(v.Nodes) == 0 {
			t.Fatal("missing AST")
		}
		if utf8.ValidString(input) && v.Source != input {
			t.Fatal("source loss")
		}
	})
}
