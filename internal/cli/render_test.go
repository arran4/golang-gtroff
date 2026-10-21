package cli

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
)

func TestRenderCLIStreamsAndSelection(t *testing.T) {
	source := ".TH DEMO 1\n.SH NAME\nname\n.SH OPTIONS\n.RS 4\nchosen\n.RE\n"
	for _, args := range [][]string{{"render", "--section", "OPTIONS", "--assets", "omit-fonts"}, {"render", "--fragment", "--section", "OPTIONS", "--assets", "omit-fonts"}} {
		code, out, stderr := runTest(args, source)
		if code != ExitOK || stderr != "" || !strings.Contains(out, "chosen") || strings.Contains(out, ">name<") {
			t.Fatalf("render selection code=%d stderr=%s", code, stderr)
		}
	}
	tree, _ := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	line := tree.Root().FindText("chosen", false)[0]
	code, out, stderr := runTest([]string{"render", "--node", fmtID(uint64(line.ID())), "--assets", "omit-fonts"}, source)
	if code != ExitOK || stderr != "" || !strings.Contains(out, "selection-context") || strings.Contains(out, ">name<") {
		t.Fatal("AST node selection context")
	}
}
func fmtID(id uint64) string { return strconv.FormatUint(id, 10) }
func TestRenderCLIFailureDoesNotWriteFile(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		input string
		code  int
	}{
		{[]string{"render", "--strict", "--assets", "omit-fonts", "-o", "out.html"}, ".UNKNOWN\n", ExitInvalid},
		{[]string{"render", "--section", "ABSENT", "-o", "out.html"}, ".SH NAME\ntext\n", ExitUsage},
		{[]string{"render", "--format", "png", "-o", "out.html"}, "", ExitUsage},
		{[]string{"render", "--node", "0", "-o", "out.html"}, "", ExitUsage},
		{[]string{"render", "--node", "2", "--section", "NAME", "-o", "out.html"}, "", ExitUsage},
		{[]string{"render", "--font-size", "0", "-o", "out.html"}, "", ExitUsage},
		{[]string{"render", "--assets", "remote", "-o", "out.html"}, "", ExitUsage},
	} {
		var out, stderr bytes.Buffer
		writes := 0
		code := Run(tc.args, Environment{In: strings.NewReader(tc.input), Out: &out, Err: &stderr, WriteOutput: func(string, string, []byte) error { writes++; return nil }})
		if code != tc.code || writes != 0 || out.Len() != 0 {
			t.Fatalf("failure wrote file: %v code=%d writes=%d stdout=%s stderr=%s", tc.args, code, writes, out.String(), stderr.String())
		}
	}
}
func TestRenderCLIFileAndFontBoundaries(t *testing.T) {
	source := ".SH NAME\nhello\n"
	fonts := fontset.Default()
	var opened []string
	var output []byte
	var sourceName string
	var stdout, stderr bytes.Buffer
	code := Run([]string{"render", "--font-regular", "font.ttf", "-o", "out.html", "input.1"}, Environment{Out: &stdout, Err: &stderr, Open: func(path string) (io.ReadCloser, error) {
		opened = append(opened, path)
		if path == "input.1" {
			return io.NopCloser(strings.NewReader(source)), nil
		}
		if path == "font.ttf" {
			return io.NopCloser(bytes.NewReader(fonts.Regular)), nil
		}
		t.Fatal("unrequested resource read: " + path)
		return nil, nil
	}, WriteOutput: func(path, input string, data []byte) error {
		if path != "out.html" {
			t.Fatal("unexpected output path")
		}
		sourceName = input
		output = append([]byte(nil), data...)
		return nil
	}})
	if code != ExitOK || len(opened) != 2 || sourceName != "input.1" || len(output) == 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("file/font code=%d stderr=%s", code, stderr.String())
	}
	code = Run([]string{"render", "--assets", "omit-fonts", "-o", "out.html"}, Environment{In: strings.NewReader(source), WriteOutput: func(string, string, []byte) error { return errors.New("replace failed") }})
	if code != ExitIO {
		t.Fatal("file replacement error ignored")
	}
	code = Run([]string{"render", "--assets", "omit-fonts"}, Environment{In: &boundaryReader{data: "partial"}})
	if code != ExitIO {
		t.Fatal("read failure rendered a partial page")
	}
	for _, writer := range []io.Writer{errorWriter{}, shortWriter{}} {
		if code := Run([]string{"render", "--assets", "omit-fonts"}, Environment{In: strings.NewReader(source), Out: writer}); code != ExitIO {
			t.Fatal("stdout failure ignored")
		}
	}
	for _, input := range []string{".B \"broken\n", ".UNKNOWN\n"} {
		code, out, _ := runTest([]string{"render", "--assets", "omit-fonts"}, input)
		if out == "" || (code != ExitOK && code != ExitInvalid) {
			t.Fatal("fallback HTML/exit contract")
		}
	}
}
