package groff

import (
	"errors"
	"fmt"
	"io"
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
	reader     io.Reader
	terminal   error
	emptyReads int
}

func (r *checkedSourceReader) Read(p []byte) (int, error) {
	if r.terminal != nil {
		return 0, r.terminal
	}
	if len(p) == 0 {
		return 0, nil
	}
	if nilInterface(r.reader) {
		r.terminal = errors.New("nil source reader")
		return 0, r.terminal
	}
	n, err := r.reader.Read(p)
	if n < 0 || n > len(p) {
		r.terminal = &ReaderCountError{Count: n, Size: len(p), Err: err}
		return 0, r.terminal
	}
	if n > 0 {
		r.emptyReads = 0
	} else if err == nil {
		r.emptyReads++
		if r.emptyReads >= 100 {
			err = io.ErrNoProgress
		}
	}
	if err != nil {
		r.terminal = err
	}
	return n, err
}
