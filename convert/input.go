package convert

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/internal/sourceio"
)

// InputError locates a source-reader or byte-limit failure after the validated
// input prefix. Unwrap retains original reader errors and ErrLimit, including
// when both occur in the same read. Source in the returned Input stays available.
type InputError struct {
	Position groff.Position
	Err      error
}

func (e *InputError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.Position.Filename, e.Position.Line, e.Position.Column, e.Err)
}
func (e *InputError) Unwrap() error { return e.Err }

func read(r io.Reader, o Options) (string, error) {
	var data []byte
	var err error
	if sourceio.IsNil(r) {
		err = fmt.Errorf("nil input reader")
	} else {
		checked := sourceio.NewReader(r, func(count, size int, err error) error {
			return &groff.ReaderCountError{Count: count, Size: size, Err: err}
		})
		data, err = io.ReadAll(io.LimitReader(checked, int64(o.MaxInputBytes)+1))
	}
	if len(data) > o.MaxInputBytes {
		err = errors.Join(ErrLimit, err)
	}
	source := string(data)
	if err != nil {
		name := o.SourceName
		if name == "" {
			name = "<input>"
		}
		position := groff.Position{
			Filename: name,
			Offset:   len(data),
			Line:     strings.Count(source, "\n") + 1,
			Column:   utf8.RuneCountInString(source[strings.LastIndexByte(source, '\n')+1:]) + 1,
		}
		err = &InputError{Position: position, Err: err}
	}
	return source, err
}
