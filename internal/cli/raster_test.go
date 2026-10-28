package cli

import (
	"bytes"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/fontset"
)

func TestRasterCLIOutputsAndPolicies(t *testing.T) {
	source := ".TH TOOL 1\n.SH CONTENT\n" + strings.Repeat(".PP\nvisible paragraph\n", 25)
	var out, diags bytes.Buffer
	args := []string{"render", "--format", "png", "--width", "400", "--page-height", "300", "--margin", "24"}
	code := Run(args, Environment{In: strings.NewReader(source), Out: &out, Err: &diags})
	if code != ExitUsage || out.Len() != 0 || !strings.Contains(diags.String(), "multiple pages") {
		t.Fatalf("multi stdout %d %s", code, diags.String())
	}
	var files []OutputFile
	closed := false
	e := Environment{Err: &diags, Open: func(path string) (io.ReadCloser, error) {
		if path != "page.1" {
			t.Fatal("unexpected input resource", path)
		}
		return &convertCloser{Reader: strings.NewReader(source), closed: &closed}, nil
	}, WriteOutput: func(string, string, []byte) error { t.Fatal("multi-page used single output"); return nil }, WriteOutputs: func(source string, outputs []OutputFile) error {
		if !closed || source != "page.1" {
			t.Fatal("source not closed/identified before output")
		}
		files = outputs
		return nil
	}}
	code = Run(append(args, "-o", "page.png", "page.1"), e)
	if code != ExitOK || len(files) < 2 {
		t.Fatalf("file output %d %d %s", code, len(files), diags.String())
	}
	if files[0].Path != "page-001.png" || files[1].Path != "page-002.png" {
		t.Fatal("page names", files[0].Path, files[1].Path)
	}
	for _, f := range files {
		if _, err := png.Decode(bytes.NewReader(f.Data)); err != nil {
			t.Fatal(err)
		}
	}
	out.Reset()
	diags.Reset()
	code = Run([]string{"render", "--format", "png", "--single", "--section", "CONTENT"}, Environment{In: strings.NewReader(".TH TOOL 1\n.SH CONTENT\nselected\n.SH OTHER\nunselected\n"), Out: &out, Err: &diags})
	if code != ExitOK {
		t.Fatal(code, diags.String())
	}
	if _, err := png.Decode(&out); err != nil {
		t.Fatal(err)
	}
}
func TestRasterBadFlagsBeforeFileAccess(t *testing.T) {
	for _, extra := range [][]string{{"--width", "-1"}, {"--max-pixels", "0"}, {"--quality", "101"}, {"--font-size", "NaN"}, {"--dpi", "0"}, {"--margin", "-1"}, {"--fragment"}, {"-o", "wrong.jpg"}} {
		args := append([]string{"render", "--format", "png"}, extra...)
		args = append(args, "source.1")
		if code := Run(args, Environment{Open: func(string) (io.ReadCloser, error) { t.Fatal("opened on bad options"); return nil, nil }}); code != ExitUsage {
			t.Fatalf("%v: %d", extra, code)
		}
	}
}
func TestRasterStrictAndExplicitFonts(t *testing.T) {
	wrote := false
	var diags bytes.Buffer
	e := Environment{In: strings.NewReader(".UNKNOWN literal\n"), Err: &diags, WriteOutput: func(string, string, []byte) error { wrote = true; return nil }}
	if code := Run([]string{"render", "--format", "png", "--strict", "-o", "page.png"}, e); code != ExitInvalid || wrote {
		t.Fatal("strict wrote", code)
	}
	var out bytes.Buffer
	opens := 0
	closed := false
	e = Environment{In: strings.NewReader(".SH TEXT\ntext\n"), Out: &out, Err: &diags, Open: func(path string) (io.ReadCloser, error) {
		opens++
		if path != "custom.ttf" {
			t.Fatal("ambient font path", path)
		}
		return &convertCloser{Reader: bytes.NewReader(fontset.Default().Mono), closed: &closed}, nil
	}}
	if code := Run([]string{"render", "--format", "png", "--single", "--font", "custom.ttf"}, e); code != ExitOK || opens != 1 || !closed {
		t.Fatalf("custom font %d %d %v %s", code, opens, closed, diags.String())
	}
}
