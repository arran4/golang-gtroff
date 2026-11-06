package script_test

import (
	"context"
	"github.com/arran4/golang-gtroff/script"
	"testing"
)

func TestSessionNativeMultilineReadiness(t *testing.T) {
	s := session(t, script.SessionOptions{})
	for _, tc := range []struct {
		code  string
		ready bool
	}{
		{"f := func() {\n", false},
		{"f := func() {\nreturn 1\n}\n", true},
		{"a := [\n", false},
		{"a := 1 +\n", false},
		{"a := `one\n", false},
		{"a := `one\ntwo`\n", true},
		{"a := )\n", true},
		{"doc.emit(\"ok\")\n", true},
	} {
		ready, e := s.Ready(context.Background(), script.Cell{Code: []byte(tc.code)})
		if e != nil || ready != tc.ready {
			t.Errorf("%q: %v %v", tc.code, ready, e)
		}
	}
}
