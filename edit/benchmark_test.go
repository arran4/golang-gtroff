package edit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func BenchmarkEnvironmentEdit(b *testing.B) {
	for _, n := range []int{10, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var input strings.Builder
			input.WriteString(".SH ENVIRONMENT\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&input, ".TP\n.B TOOL_%d\nOld directory.\n", i)
			}
			set, e := source.Read(context.Background(), strings.NewReader(input.String()), "bench.1", source.Options{})
			if e != nil {
				b.Fatal(e)
			}
			c := edit.Command{Select: query.Select(query.Definition), Operation: edit.ReplaceDescription, Text: "New directory."}
			b.ReportAllocs()
			b.SetBytes(int64(input.Len()))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w, e := edit.New(set, edit.Options{})
				if e != nil {
					b.Fatal(e)
				}
				p, e := w.Plan(context.Background(), c)
				if e != nil {
					b.Fatal(e)
				}
				if len(p.Patches()) != n {
					b.Fatal("lost patches")
				}
				if e = w.Apply(p); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
func FuzzUntouchedSource(f *testing.F) {
	for _, s := range []string{".SH ENVIRONMENT\n.TP\nVALUE\nneedle\n", ".\\\" retained\r\nneedle\r\n\xff\n", ".TH \"broken\nneedle\n", ".de X\nneedle\n..\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 4096 {
			t.Skip()
		}
		set, e := source.Read(context.Background(), strings.NewReader(s), "fuzz.1", source.Options{})
		if e != nil {
			return
		}
		w, e := edit.New(set, edit.Options{})
		if e != nil {
			return
		}
		p, e := w.Plan(context.Background(), edit.Command{Select: edit.Root(), Operation: edit.FindReplaceSource, Find: "needle", Text: "replacement"})
		if e != nil {
			return
		}
		if e = w.Apply(p); e != nil {
			t.Fatal(e)
		}
		b, e := w.Data("fuzz.1")
		if e != nil {
			t.Fatal(e)
		}
		if string(b) != strings.ReplaceAll(s, "needle", "replacement") {
			t.Fatalf("unintended bytes changed: %q -> %q", s, b)
		}
	})
}
