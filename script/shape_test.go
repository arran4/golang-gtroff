package script_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/script"
)

func TestNativeOperandBoundsBeforeExecution(t *testing.T) {
	args := strings.TrimSuffix(strings.Repeat("1,", 256), ",")
	var body strings.Builder
	for i := 0; i < 256; i++ {
		fmt.Fprintf(&body, "v%d:= %d;", i, i)
	}
	for _, code := range []string{
		"doc.emit(\"must be withheld\"); f:=func(...v){}; f(" + args + ")",
		"f:=func(){" + body.String() + "return v255}; f()",
	} {
		r, e := script.Run(context.Background(), script.Request{Code: []byte(code)}, script.Options{})
		if !errors.Is(e, script.ErrLimit) || len(r.Output) > 0 {
			t.Fatal(e, r)
		}
		s := session(t, script.SessionOptions{})
		r, e = s.Eval(context.Background(), script.Cell{Code: []byte(code)})
		if !errors.Is(e, script.ErrLimit) || len(r.Output) > 0 {
			t.Fatal(e, r)
		}
		if got := string(eval(t, s, `doc.emit("recovered")`).Output); got != "recovered" {
			t.Fatal(got)
		}
	}
}
func TestSessionStateAndCumulativeBoundsAreTransactional(t *testing.T) {
	for _, o := range []script.SessionOptions{
		{Cell: script.Options{MaxConstants: 4}},
		{MaxStateObjects: 100},
	} {
		s := session(t, o)
		code := `huge := [1,2,3,4,5,6]`
		if o.MaxStateObjects != 0 {
			code = `huge := range(0,100)`
		}
		if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(code)}); !errors.Is(e, script.ErrLimit) {
			t.Fatal(e)
		}
		if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.emit_json(huge)`)}); e == nil {
			t.Fatal("failed state escaped")
		}
		eval(t, s, `doc.emit("ok")`)
	}
	var globals strings.Builder
	for i := 0; i < 1024; i++ {
		fmt.Fprintf(&globals, "v%d:=1;", i)
	}
	s := session(t, script.SessionOptions{})
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(globals.String())}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	eval(t, s, `doc.emit("ok")`)
	s = session(t, script.SessionOptions{MaxSessionHostCalls: 1})
	eval(t, s, `doc.emit("one")`)
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.emit("two")`)}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	s = session(t, script.SessionOptions{Cell: script.Options{AllowWriteRequests: true}, MaxSessionOutputBytes: 8})
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.write("out.1")`)}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	s = session(t, script.SessionOptions{MaxSessionBytes: 1})
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte("x:=1")}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
}
