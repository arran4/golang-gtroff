package manlib_test

import (
	"context"
	"errors"
	"fmt"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
)

func ExampleBuild_readBudget() {
	files := fstest.MapFS{
		"a.1":    {Data: []byte("accepted\n")},
		"b.1.gz": {Data: []byte("not gzip\n")},
		"c.1.gz": {Data: []byte("not gzip\n")},
	}
	index, err := manlib.Build(context.Background(), []manlib.Root{{Label: "manuals", FS: files}}, manlib.Limits{MaxTotalBytes: 25})
	// The invalid b page consumed bytes too. Inspect the partial snapshot and
	// handle the error before deciding whether it is suitable for serving.
	fmt.Printf("indexed=%d skipped=%d budgetExceeded=%v\n", len(index.Entries()), len(index.Notices()), errors.Is(err, manlib.ErrLimit))
	// Output:
	// indexed=1 skipped=1 budgetExceeded=true
}
