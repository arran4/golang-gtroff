package analyze_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/source"
)

func BenchmarkFeatureInventory(b *testing.B) {
	for _, definitions := range []int{10, 1000} {
		b.Run(fmt.Sprint(definitions), func(b *testing.B) {
			input := ".TH TOOL 1\n.SH ENVIRONMENT\n" + strings.Repeat(".TP\n.B TOOL_HOME\nData directory.\n", definitions)
			set, err := source.Read(context.Background(), strings.NewReader(input), "bench.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := analyze.Run(context.Background(), set, analyze.Options{})
				if err != nil || report.Counts.Controls[".TP"] != definitions {
					b.Fatal("inventory benchmark lost control evidence", err)
				}
			}
		})
	}
}
func FuzzParseRules(f *testing.F) {
	for _, input := range []string{`{"schemaVersion":1}`, `{"schemaVersion":1,"rules":[{"id":"project.table","match":{"macro":".TS"},"feature":"table","reason":"Review source"}]}`, "{"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		registry, err := analyze.ParseRules(strings.NewReader(input))
		if err == nil {
			if _, err := analyze.NewRegistry(registry.Rules()...); err != nil {
				t.Fatal(err)
			}
		}
	})
}
