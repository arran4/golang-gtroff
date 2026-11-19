package manlib

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"testing"
	"testing/fstest"
)

func TestDecodedWorkStopsAtSharedDetectionByte(t *testing.T) {
	var out bytes.Buffer
	z := gzip.NewWriter(&out)
	if _, err := z.Write(bytes.Repeat([]byte("x"), 4096)); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	raw := out.Bytes()
	budget := &pageByteBudget{limit: len(raw) + 16}
	data, err := readPage(context.Background(), fstest.MapFS{"page.1.gz": {Data: raw}}, "page.1.gz", 8192, budget)
	if !errors.Is(err, ErrLimit) || data != nil || budget.used != budget.limit+1 {
		t.Fatalf("decode work escaped shared cap: used=%d limit=%d error=%v", budget.used, budget.limit, err)
	}
}
