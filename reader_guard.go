package groff

import (
	"errors"
	"fmt"
	"io"

	"github.com/arran4/golang-gtroff/internal/sourceio"
)

// ErrInvalidReadCount identifies a reader violating 0 <= n <= len(p).
var ErrInvalidReadCount = errors.New("invalid source reader count")

// ReaderCountError retains a malformed reader result and its original error.
// No bytes from that call are trusted; earlier validated bytes remain available.
type ReaderCountError struct {
	Count, Size int
	Err         error
}

func (e *ReaderCountError) Error() string {
	return fmt.Sprintf("%v: got %d for buffer of %d bytes", ErrInvalidReadCount, e.Count, e.Size)
}
func (e *ReaderCountError) Unwrap() []error {
	if e.Err == nil {
		return []error{ErrInvalidReadCount}
	}
	return []error{ErrInvalidReadCount, e.Err}
}

// checkedSourceReader validates counts before callers slice buffers, caches
// terminal outcomes and rejects 100 consecutive empty/no-error reads. It does
// not close the caller's reader or interrupt a blocked Read method.
type checkedSourceReader struct {
	reader   io.Reader
	terminal error
	guard    *sourceio.Reader
}

func (r *checkedSourceReader) Read(p []byte) (int, error) {
	if r.guard == nil {
		r.guard = sourceio.NewReader(r.reader, func(count, size int, err error) error {
			return &ReaderCountError{Count: count, Size: size, Err: err}
		})
	}
	n, err := r.guard.Read(p)
	r.terminal = r.guard.Terminal()
	return n, err
}
