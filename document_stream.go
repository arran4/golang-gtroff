package groff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// DocumentInputError separates terminal reader/resource failures from syntax
// diagnostics. Unwrap retains the original reader error or DocumentLimitError.
type DocumentInputError struct {
	Position Position
	Err      error
}

func (e *DocumentInputError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.Position.Filename, e.Position.Line, e.Position.Column, e.Err)
}
func (e *DocumentInputError) Unwrap() error { return e.Err }

// DocumentLimitError identifies a ParseDocument resource budget failure.
type DocumentLimitError struct {
	Kind  string
	Limit int
}

func (e *DocumentLimitError) Error() string {
	return fmt.Sprintf("%s budget exceeded (%d)", e.Kind, e.Limit)
}

type documentByteBudget struct{ limit, read int }

type documentCapture struct {
	reader   io.Reader
	budget   *documentByteBudget
	source   strings.Builder
	terminal error
}

func (r *documentCapture) Read(p []byte) (int, error) {
	if r.terminal != nil {
		return 0, r.terminal
	}
	remaining := r.budget.limit - r.budget.read + 1
	if remaining <= 0 {
		r.terminal = &DocumentLimitError{Kind: "document byte", Limit: r.budget.limit}
		return 0, r.terminal
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := r.reader.Read(p)
	r.source.Write(p[:n])
	r.budget.read += n
	if r.budget.read > r.budget.limit {
		err = errors.Join(&DocumentLimitError{Kind: "document byte", Limit: r.budget.limit}, nonEOF(err))
	}
	if err != nil {
		r.terminal = err
	}
	return n, err
}
func nonEOF(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}

type documentLine struct {
	raw  string
	span Span
}
type documentStream struct {
	input                    *documentCapture
	buffer                   *bufio.Reader
	position                 Position
	lines, maxLines, maxLine int
	stopped                  error
}

func newDocumentStream(r io.Reader, o ParserOptions, budget *documentByteBudget) *documentStream {
	input := &documentCapture{reader: r, budget: budget}
	return &documentStream{input: input, buffer: bufio.NewReaderSize(input, 4096), position: Position{Filename: o.SourceName, Line: 1, Column: 1}, maxLines: o.MaxLines, maxLine: o.MaxLineLength}
}
func (s *documentStream) next() (documentLine, error) {
	start := s.position
	if s.stopped != nil {
		return documentLine{}, s.stopped
	}
	fail := func(raw string, err error) (documentLine, error) {
		s.position = advanceSourcePosition(start, raw)
		if err != nil && err != io.EOF {
			position := s.position
			var limit *DocumentLimitError
			if errors.As(err, &limit) && limit.Kind == "line length" {
				position = start
			}
			err = &DocumentInputError{Position: position, Err: err}
		}
		s.stopped = err
		return documentLine{raw: raw, span: Span{Start: start, End: s.position}}, err
	}
	if s.maxLines > 0 && s.lines >= s.maxLines {
		if _, err := s.buffer.Peek(1); err != nil {
			return fail("", err)
		}
		return fail("", &DocumentLimitError{Kind: "line", Limit: s.maxLines})
	}
	var raw strings.Builder
	for {
		part, err := s.buffer.ReadSlice('\n')
		raw.Write(part)
		if s.maxLine > 0 && raw.Len() > s.maxLine {
			return fail(raw.String(), errors.Join(&DocumentLimitError{Kind: "line length", Limit: s.maxLine}, nonEOF(nonBufferFull(err))))
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if raw.Len() > 0 {
			s.lines++
		}
		if err != nil {
			return fail(raw.String(), err)
		}
		s.position = advanceSourcePosition(start, raw.String())
		return documentLine{raw: raw.String(), span: Span{Start: start, End: s.position}}, nil
	}
}
func nonBufferFull(err error) error {
	if err == bufio.ErrBufferFull {
		return nil
	}
	return err
}

// Buffered data was already read from the caller. Preserve it without attempting
// any more reads after a terminal input or strict syntax failure.
func (s *documentStream) drainTail() documentLine {
	data := s.input.source.String()
	start := s.position
	if start.Offset >= len(data) {
		return documentLine{}
	}
	raw := data[start.Offset:]
	s.position = advanceSourcePosition(start, raw)
	return documentLine{raw: raw, span: Span{Start: start, End: s.position}}
}
func documentLineBody(raw string) string {
	if strings.HasSuffix(raw, "\n") {
		return strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
	}
	return raw
}
