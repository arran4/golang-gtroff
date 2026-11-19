package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type conversionFaultReader struct{ done bool }

func (r *conversionFaultReader) Read(p []byte) (int, error) {
	if r.done {
		panic("terminal reader retried")
	}
	r.done = true
	return copy(p, "kept\n"), errors.New("input read failed")
}

type conversionNilFile struct{}

func (*conversionNilFile) Read([]byte) (int, error) { panic("typed-nil file read") }
func (*conversionNilFile) Close() error             { panic("typed-nil file close") }

func TestConvertInputFailureKeepsOutputAndDiagnostics(t *testing.T) {
	for _, from := range []string{"html", "roff"} {
		to := "roff"
		if from == "roff" {
			to = "html"
		}
		for _, mode := range []string{"borrowed", "owned", "read-and-close", "typed-nil"} {
			t.Run(from+"/"+mode, func(t *testing.T) {
				var out, diags bytes.Buffer
				wrote, closed := false, false
				reader := &conversionFaultReader{}
				e := Environment{In: reader, Out: &out, Err: &diags, WriteOutput: func(string, string, []byte) error { wrote = true; return nil }}
				args := []string{"convert", "--from", from, "--to", to, "-o", "existing-output"}
				if mode != "borrowed" {
					args = append(args, "-i", "source")
					e.Open = func(string) (io.ReadCloser, error) {
						if mode == "typed-nil" {
							return (*conversionNilFile)(nil), nil
						}
						var closeErr error
						if mode == "read-and-close" {
							closeErr = errors.New("input close failed")
						}
						return &convertCloser{Reader: reader, closed: &closed, closeErr: closeErr}, nil
					}
				}
				code := Run(args, e)
				if code != ExitIO || wrote || out.Len() != 0 || closed != (mode == "owned" || mode == "read-and-close") {
					t.Fatalf("code=%d wrote=%v closed=%v diagnostics=%s", code, wrote, closed, diags.String())
				}
				if mode == "typed-nil" {
					if !strings.Contains(diags.String(), "nil input reader") {
						t.Fatal(diags.String())
					}
				} else if !strings.Contains(diags.String(), "input read failed") {
					t.Fatal("reader cause lost", diags.String())
				}
				if mode == "read-and-close" && !strings.Contains(diags.String(), "input close failed") {
					t.Fatal("close cause lost", diags.String())
				}
			})
		}
	}
}
