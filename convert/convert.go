// Package convert implements bounded, best-effort semantic HTML/man conversion.
// It evaluates no requests, executes no scripts and opens no files or URLs.
package convert

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderhtml"
)

var ErrLimit = errors.New("conversion resource limit exceeded")
var ErrStrict = errors.New("strict conversion rejected diagnostics")

// Options uses zero for defaults. Limits can be reduced, never raised above the
// hard ceilings (32 MiB input/output, 100000 nodes, 128 levels).
// HTML controls outbound HTML; no settings authorize resource loading.
type Options struct {
	Strict                                            bool
	SourceName                                        string
	MaxInputBytes, MaxOutputBytes, MaxNodes, MaxDepth int
	HTML                                              renderhtml.Options
}
type Result struct {
	Diagnostics []groff.Diagnostic
	Bytes       int
}

// Input retains the original input separately from its derived semantic tree.
// HTML origins identify the input snapshot, not token-accurate HTML positions.
type Input struct {
	Source   string
	Document *man.Document
}

func defaults(o Options) Options {
	if o.MaxInputBytes <= 0 {
		o.MaxInputBytes = 8 << 20
	}
	if o.MaxInputBytes > 32<<20 {
		o.MaxInputBytes = 32 << 20
	}
	if o.MaxOutputBytes <= 0 || o.MaxOutputBytes > 32<<20 {
		o.MaxOutputBytes = 32 << 20
	}
	if o.MaxNodes <= 0 || o.MaxNodes > 100000 {
		o.MaxNodes = 100000
	}
	if o.MaxDepth <= 0 || o.MaxDepth > 128 {
		o.MaxDepth = 128
	}
	return o
}
func read(r io.Reader, o Options) (string, error) {
	if r == nil {
		return "", fmt.Errorf("nil input reader")
	}
	b, err := io.ReadAll(io.LimitReader(r, int64(o.MaxInputBytes)+1))
	if err != nil {
		return "", err
	}
	if len(b) > o.MaxInputBytes {
		return "", ErrLimit
	}
	return string(b), nil
}

// ImportMan keeps the roff source; parsing and semantic diagnostics remain on
// Document. Strict mode returns that inspectable result together with ErrStrict.
func ImportMan(r io.Reader, o Options) (Input, error) {
	o = defaults(o)
	s, err := read(r, o)
	if err != nil {
		return Input{}, err
	}
	tree, err := groff.ParseAST(bytes.NewBufferString(s), groff.WithPackage("man"), groff.WithSourceName(o.SourceName), groff.WithMaxLineLength(o.MaxInputBytes), groff.WithASTBytes(o.MaxInputBytes), groff.WithASTNodes(o.MaxNodes), groff.WithASTDepth(o.MaxDepth))
	if err != nil {
		return Input{Source: s}, err
	}
	d, err := man.Build(tree, man.Options{MaxBytes: o.MaxInputBytes, MaxNodes: o.MaxNodes, MaxDepth: o.MaxDepth})
	in := Input{Source: s, Document: d}
	if err != nil {
		return in, err
	}
	if o.Strict && len(d.Diagnostics) > 0 {
		return in, ErrStrict
	}
	return in, nil
}
func HTMLToMan(r io.Reader, w io.Writer, o Options) (Result, error) {
	in, err := ImportHTML(r, o)
	if err != nil {
		var ds []groff.Diagnostic
		if in.Document != nil {
			ds = in.Document.Diagnostics
		}
		return Result{Diagnostics: ds}, err
	}
	return WriteMan(w, in.Document, o)
}
func ManToHTML(r io.Reader, w io.Writer, o Options) (Result, error) {
	o = defaults(o)
	in, err := ImportMan(r, o)
	if err != nil {
		var ds []groff.Diagnostic
		if in.Document != nil {
			ds = in.Document.Diagnostics
		}
		return Result{Diagnostics: ds}, err
	}
	h := o.HTML
	h.Strict = h.Strict || o.Strict
	if h.MaxOutputBytes == 0 || h.MaxOutputBytes > o.MaxOutputBytes {
		h.MaxOutputBytes = o.MaxOutputBytes
	}
	res, err := renderhtml.Render(w, in.Document, h)
	return Result{Diagnostics: res.Diagnostics, Bytes: res.Bytes}, err
}
func output(w io.Writer, b []byte, res Result, o Options) (Result, error) {
	if w == nil {
		return res, fmt.Errorf("nil output writer")
	}
	if len(b) > o.MaxOutputBytes {
		return res, ErrLimit
	}
	if o.Strict && len(res.Diagnostics) > 0 {
		return res, ErrStrict
	}
	n, err := w.Write(b)
	res.Bytes = n
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	return res, err
}
