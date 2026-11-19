package script_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/source"
)

// Fuzz a finite literal-only program, not arbitrary untrusted native Tengo
// operations (the VM's allocation counter is not a hard heap-byte quota).
func FuzzNativeExactSource(f *testing.F) {
	for _, s := range []string{".SH A\ntext\n", ".UNKNOWN x\r\n.\\\" keep\r\n\xff\n", ".de X\nbody\n", ".so missing\n", "\x00\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 4096 {
			t.Skip()
		}
		set, e := source.Read(context.Background(), strings.NewReader(s), "fuzz.1", source.Options{NoIncludes: true})
		if e != nil {
			return
		}
		program := `doc.edit({selector:"source[source-id=1]",operation:"replace-source",text:` + strconv.Quote(s) + `});doc.emit(doc.source())`
		result, e := script.Run(context.Background(), script.Request{Code: []byte(program), Sources: set}, script.Options{AllowEdits: true})
		if e != nil {
			if result.Working != nil || len(result.Output) != 0 {
				t.Fatal("failed run leaked changes")
			}
			return
		}
		if string(result.Output) != s || set.File(1).AST().Source() != s {
			t.Fatalf("literal normalization: %q -> %q", s, result.Output)
		}
	})
}
