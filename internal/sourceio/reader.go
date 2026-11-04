// Package sourceio validates source-reader results before consumers slice buffers.
package sourceio

import (
	"errors"
	"io"
	"reflect"
)

type Reader struct {
	source       io.Reader
	invalidCount func(int, int, error) error
	terminal     error
	emptyReads   int
}

// NewReader caches terminal errors and stops after 100 consecutive empty reads.
// invalidCount is a required internal error constructor, keeping public error
// types in their owning package without an import cycle. Readers are not closed;
// a blocked Read or a panic raised by caller code remains caller-owned behavior.
func NewReader(source io.Reader, invalidCount func(int, int, error) error) *Reader {
	return &Reader{source: source, invalidCount: invalidCount}
}

func IsNil(r io.Reader) bool {
	if r == nil {
		return true
	}
	v := reflect.ValueOf(r)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (r *Reader) Terminal() error { return r.terminal }

func (r *Reader) Read(p []byte) (int, error) {
	if r.terminal != nil {
		return 0, r.terminal
	}
	if len(p) == 0 {
		return 0, nil
	}
	if IsNil(r.source) {
		r.terminal = errors.New("nil source reader")
		return 0, r.terminal
	}
	n, err := r.source.Read(p)
	if n < 0 || n > len(p) {
		r.terminal = r.invalidCount(n, len(p), err)
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
