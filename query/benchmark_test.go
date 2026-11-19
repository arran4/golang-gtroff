package query_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func BenchmarkEnvironmentQuery(b *testing.B) {
	for _, definitions := range []int{10, 1000} {
		b.Run(fmt.Sprint(definitions), func(b *testing.B) {
			var input strings.Builder
			input.WriteString(".SH ENVIRONMENT\n")
			for i := 0; i < definitions; i++ {
				fmt.Fprintf(&input, ".TP\n.B TOOL_%d\nDirectory for tool data.\n", i)
			}
			set, err := source.Read(context.Background(), strings.NewReader(input.String()), "bench.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			selector, err := query.ParseSelector(`section[name="ENVIRONMENT"] definition`)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(input.Len()))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := query.Execute(context.Background(), set, selector, query.Options{})
				if err != nil || len(result.Matches) != definitions {
					b.Fatal("query benchmark lost definitions", err)
				}
			}
		})
	}
}
func BenchmarkParseSelector(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := query.ParseSelector(`section[name="ENVIRONMENT"] definition[term^="TOOL_"]`); err != nil {
			b.Fatal(err)
		}
	}
}
