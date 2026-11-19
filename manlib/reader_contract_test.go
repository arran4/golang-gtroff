package manlib

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
)

type faultyPageReader struct {
	mode   string
	calls  int
	empty  int
	prefix string
}

func (r *faultyPageReader) Read(p []byte) (int, error) {
	r.calls++
	if r.prefix != "" {
		n := copy(p, r.prefix)
		r.prefix = r.prefix[n:]
		return n, nil
	}
	switch r.mode {
	case "empty":
		r.empty++
		// Bound the regression itself when the production guard is absent.
		if r.empty > 100 {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, nil
	case "negative":
		return -1, io.ErrUnexpectedEOF
	case "oversized":
		return len(p) + 1, io.ErrUnexpectedEOF
	}
	panic("unknown test reader")
}

func TestPageReaderContracts(t *testing.T) {
	for _, prefix := range []string{"", "abc"} {
		for _, mode := range []string{"empty", "negative", "oversized"} {
			t.Run(mode+"/prefix="+prefix, func(t *testing.T) {
				r := &faultyPageReader{mode: mode, prefix: prefix}
				budget := &pageByteBudget{limit: 16}
				data, err := budget.read(context.Background(), r, 16)
				want := io.ErrNoProgress
				calls := 100
				if mode != "empty" {
					want = groff.ErrInvalidReadCount
					calls = 1
				}
				if prefix != "" {
					calls++
				}
				if string(data) != prefix || budget.used != len(prefix) || r.calls != calls || !errors.Is(err, want) {
					t.Fatalf("data=%q used=%d calls=%d error=%v; want %d calls and %v", data, budget.used, r.calls, err, calls, want)
				}
				if mode != "empty" && !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatal("lost reader cause", err)
				}
			})
		}
	}
}

func TestPageReaderNilAndCancellation(t *testing.T) {
	for _, r := range []io.Reader{nil, (*faultyPageReader)(nil)} {
		budget := &pageByteBudget{limit: 16}
		data, err := budget.read(context.Background(), r, 16)
		if err == nil || len(data) != 0 || budget.used != 0 {
			t.Fatalf("nil reader: data=%q used=%d err=%v", data, budget.used, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &faultyPageReader{mode: "negative"}
	budget := &pageByteBudget{limit: 16}
	if _, err := budget.read(ctx, r, 16); !errors.Is(err, context.Canceled) || r.calls != 0 {
		t.Fatalf("cancelled read called source: calls=%d err=%v", r.calls, err)
	}
	// A successful bounded read still preserves source and charges it once.
	data, err := budget.read(context.Background(), strings.NewReader("visible\n"), 16)
	if err != nil || string(data) != "visible\n" || budget.used != len(data) {
		t.Fatalf("valid read: data=%q used=%d err=%v", data, budget.used, err)
	}
}
