package docdiff_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/docdiff"
	"github.com/arran4/golang-gtroff/source"
)

func FuzzIdenticalSnapshots(f *testing.F) {
	for _, text := range []string{header + def("HOME", "Directory."), header + def("SAME", "One.") + def("SAME", "Two."), header + ".UNKNOWN literal\n", ".de STORED\n.SH hidden\n..\n"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 4096 {
			t.Skip()
		}
		set, loadErr := source.Read(context.Background(), strings.NewReader(text), "fuzz.1", source.Options{MaxBytes: 4096, MaxNodes: 2048})
		if set == nil {
			return
		}
		before := set.File(1).AST().Source()
		input := docdiff.Input{Name: "fuzz.1", Sources: set, Error: loadErr}
		for _, view := range []docdiff.View{docdiff.Semantic, docdiff.Source} {
			r, err := docdiff.Compare(context.Background(), input, input, docdiff.Options{View: view, MaxNodes: 4096, MaxObjects: 256, MaxDataBytes: 1 << 20, MaxResultBytes: 1 << 20, MaxWork: 1 << 20})
			if err == nil && (r.Different || len(r.Ambiguities) > 0 || !r.Complete) {
				t.Fatal("identical inspected input changed", r)
			}
			if set.File(1).AST().Source() != before {
				t.Fatal("comparison mutated source")
			}
		}
	})
}
func BenchmarkCompare(b *testing.B) {
	for _, n := range []int{10, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var old, new strings.Builder
			old.WriteString(header)
			new.WriteString(header)
			for i := 0; i < n; i++ {
				body := "Description."
				old.WriteString(def(fmt.Sprintf("VALUE_%d", i), body))
				if i == n-1 {
					body = "Changed description."
				}
				new.WriteString(def(fmt.Sprintf("VALUE_%d", i), body))
			}
			beforeSet, err := source.Read(context.Background(), strings.NewReader(old.String()), "old.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			afterSet, err := source.Read(context.Background(), strings.NewReader(new.String()), "new.1", source.Options{})
			if err != nil {
				b.Fatal(err)
			}
			before, after := docdiff.Input{Sources: beforeSet}, docdiff.Input{Sources: afterSet}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r, err := docdiff.Compare(context.Background(), before, after, docdiff.Options{})
				if err != nil || len(r.Changes) != 1 {
					b.Fatal("benchmark compare", err, r.Status)
				}
			}
		})
	}
}
