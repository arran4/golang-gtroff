package script_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/script"
)

// Generate finite native programs from byte-valued data. No arbitrary supplied
// code or unbounded native operation is executed. Observe aliases/closures after
// a failed cell rather than testing the copier's internal representation.
func FuzzSessionGraphRollback(f *testing.F) {
	f.Add([]byte{1, 2, 3}, byte(1))
	f.Add([]byte{0}, byte(0))
	f.Fuzz(func(t *testing.T, data []byte, index byte) {
		if len(data) == 0 || len(data) > 32 {
			return
		}
		var literals []string
		for _, v := range data {
			literals = append(literals, fmt.Sprint(v))
		}
		start := int(index) % len(data)
		setup := fmt.Sprintf(`a := [%s]; b := a[%d:]; m := {a:a}; m.self=m
 make := func(){n:=0;return [func(){n++;return n},func(){return n}]}; fns:=make()`, strings.Join(literals, ","), start)
		s, e := script.NewSession(context.Background(), script.Request{}, script.SessionOptions{})
		if e != nil {
			t.Fatal(e)
		}
		eval(t, s, setup)
		r, e := s.Eval(context.Background(), script.Cell{Code: []byte(`b[0] = 999; m.self.a[0] = 888; fns[0](); doc.emit("withheld"); fail := 0; fail()`)})
		if e == nil || !strings.Contains(e.Error(), "not callable") || len(r.Output) != 0 || r.Working != nil {
			t.Fatal("failed cell published", e)
		}
		result := eval(t, s, `doc.emit_json([a[0],b[0],fns[1]()])`)
		want := fmt.Sprintf("[%d,%d,0]\n", data[0], data[start])
		if string(result.Output) != want {
			t.Fatalf("got %s want %s", result.Output, want)
		}
	})
}
