package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestConvertCommand(t *testing.T) {
	var out, diags bytes.Buffer
	code := Run([]string{"convert", "--from", "html", "--to", "roff"}, Environment{In: strings.NewReader("<h1>TOOL</h1><p>.danger</p>"), Out: &out, Err: &diags})
	if code != ExitOK || !strings.Contains(out.String(), `.TH "TOOL" "1"`) || !strings.Contains(out.String(), `\&\f[R].danger`) {
		t.Fatalf("convert: %d %q %q", code, out.String(), diags.String())
	}
	source := out.String()
	out.Reset()
	diags.Reset()
	code = Run([]string{"convert", "--from", "roff", "--to", "html", "--fragment", "--assets", "omit-fonts"}, Environment{In: strings.NewReader(source), Out: &out, Err: &diags})
	if code != ExitOK || !strings.Contains(out.String(), "<article") || strings.Contains(out.String(), "<!doctype") {
		t.Fatalf("HTML: %d %q", code, diags.String())
	}
}
func TestConvertFileCapabilities(t *testing.T) {
	for _, sourceName := range []string{"source.html", "<stdin>"} {
		t.Run(sourceName, func(t *testing.T) {
			closed := false
			wrote := false
			opens := 0
			var diags bytes.Buffer
			e := Environment{Err: &diags, Open: func(path string) (io.ReadCloser, error) {
				opens++
				if path != sourceName {
					t.Fatalf("unexpected resource read %q", path)
				}
				return &convertCloser{Reader: strings.NewReader("<p>text</p>"), closed: &closed}, nil
			}, WriteOutput: func(destination, source string, data []byte) error {
				if !closed {
					t.Fatal("input open during replacement")
				}
				if destination != "result.1" || source != sourceName || !bytes.Contains(data, []byte("text")) {
					t.Fatalf("output args %q %q %s", destination, source, data)
				}
				wrote = true
				return nil
			}}
			code := Run([]string{"convert", "--from", "html", "--to", "roff", "-i", sourceName, "-o", "result.1"}, e)
			if code != ExitOK || !wrote || opens != 1 {
				t.Fatalf("conversion %d %q", code, diags.String())
			}
		})
	}
}

type convertCloser struct {
	io.Reader
	closed   *bool
	closeErr error
}

func (c *convertCloser) Close() error { *c.closed = true; return c.closeErr }
func TestConvertFailurePreservesOutput(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		strict       bool
		closeErr     error
		want         int
	}{
		{"strict", "<script>literal</script>", true, nil, ExitInvalid},
		{"close", "<p>text</p>", false, errors.New("close failed"), ExitIO},
	} {
		t.Run(tc.name, func(t *testing.T) {
			closed := false
			wrote := false
			var diags bytes.Buffer
			e := Environment{Err: &diags, Open: func(string) (io.ReadCloser, error) {
				return &convertCloser{Reader: strings.NewReader(tc.source), closed: &closed, closeErr: tc.closeErr}, nil
			}, WriteOutput: func(string, string, []byte) error { wrote = true; return nil }}
			args := []string{"convert", "--from", "html", "--to", "roff", "-i", "source", "-o", "dest"}
			if tc.strict {
				args = append(args, "--strict")
			}
			if code := Run(args, e); code != tc.want || wrote || !closed {
				t.Fatalf("failure handling: %d wrote=%v closed=%v %s", code, wrote, closed, diags.String())
			}
		})
	}
}
func TestConvertUsageBeforeAccess(t *testing.T) {
	for _, args := range [][]string{{"convert"}, {"convert", "--from", "html", "--to", "html"}, {"convert", "--from", "html", "--to", "roff", "source"}, {"convert", "--from", "html", "--to", "roff", "--fragment"}, {"convert", "--from", "roff", "--to", "html", "--assets", "remote"}, {"convert", "--from", "html", "--to", "roff", "-i", ""}} {
		e := Environment{Open: func(string) (io.ReadCloser, error) { t.Fatal("opened input on usage failure"); return nil, nil }}
		if code := Run(args, e); code != ExitUsage {
			t.Fatalf("%v: %d", args, code)
		}
	}
}
