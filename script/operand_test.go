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
		`f:=func(...v){doc.emit_json(len(v))}; g:=func(){doc.emit("wrong-dispatch")}; f(` + strings.Repeat("0,", 255) + `g)`,
	} {
		r, e := script.Run(context.Background(), script.Request{Code: []byte(code)}, script.Options{})
		if !errors.Is(e, script.ErrLimit) || len(r.Output) > 0 {
			t.Fatal(e, r)
		}
	}
}

func TestNativeCallOperandBoundary(t *testing.T) {
	code := `f:=func(...v){doc.emit_json(len(v))}; f(` + strings.TrimSuffix(strings.Repeat("0,", 255), ",") + `)`
	r, e := script.Run(context.Background(), script.Request{Code: []byte(code)}, script.Options{})
	if e != nil || string(r.Output) != "255\n" {
		t.Fatal(e, string(r.Output))
	}
}
