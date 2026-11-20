package script_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/source"
)

// Include creation, capture/query of native records, then a second transaction
// copying retained query data and calling a persistent function. Sources parse
// before timing so this measures the session adapter rather than input loading.
func BenchmarkSessionCells(b *testing.B) {
	for _, n := range []int{10, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			ctx := context.Background()
			var input strings.Builder
			input.WriteString(".SH ENVIRONMENT\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&input, ".TP\nVAR_%d\nDirectory %d.\n", i, i)
			}
			set, e := source.Read(ctx, strings.NewReader(input.String()), "page.1", source.Options{})
			if e != nil {
				b.Fatal(e)
			}
			accept := func(context.Context, *script.Result) error { return nil }
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				s, e := script.NewSession(ctx, script.Request{Sources: set}, script.SessionOptions{})
				if e != nil {
					b.Fatal(e)
				}
				for _, program := range []string{`matches := doc.query("definition"); count := 0; next := func() { count++; return count }`, `doc.emit_json([len(matches),next()])`} {
					if _, e = s.Eval(ctx, script.Cell{Code: []byte(program)}); e != nil {
						b.Fatal(e)
					}
					if e = s.Publish(ctx, accept); e != nil {
						b.Fatal(e)
					}
				}
			}
		})
	}
}
