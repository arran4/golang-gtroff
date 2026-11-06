package script_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/source"
	"github.com/d5/tengo/v2"
)

func input(t *testing.T) *source.Set {
	t.Helper()
	set, e := source.Read(context.Background(), strings.NewReader(".SH ENVIRONMENT\n.TP\nTOOL_HOME\nOld directory.\n.TP\nPATH\nSearch path.\n"), "tool.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	return set
}
func TestRunQueryEditAndStagedOutput(t *testing.T) {
	set := input(t)
	code := `matches := doc.query("section[name=ENVIRONMENT] definition")
 for match in matches {
  if match.term == "TOOL_HOME" {
   doc.emit_json({term: match.term, file: match.source, index: match.index})
   doc.edit({selector: "definition[term=TOOL_HOME]", operation: "replace-description", text: "New directory."})
  }
 }
 doc.emit(doc.extract("definition[term=TOOL_HOME]", "raw"))
 doc.write("out.1")`
	result, e := script.Run(context.Background(), script.Request{Name: "change.tengo", Code: []byte(code), Sources: set}, script.Options{AllowEdits: true, AllowWriteRequests: true})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(string(result.Output), "{\"file\":\"tool.1\",\"index\":1,\"term\":\"TOOL_HOME\"}\n") || !strings.Contains(string(result.Output), "New directory.") {
		t.Fatal(string(result.Output))
	}
	if result.Working == nil || len(result.Working.Changes()) != 1 || len(result.Writes) != 1 || result.Writes[0].Destination != "out.1" {
		t.Fatalf("missing result: %+v", result)
	}
	if !strings.Contains(set.File(1).AST().Source(), "Old directory.") {
		t.Fatal("original source changed")
	}
	failed, e := script.Run(context.Background(), script.Request{Name: "failed.tengo", Code: []byte(code + `; invalid := doc.query("section[")`), Sources: set}, script.Options{AllowEdits: true, AllowWriteRequests: true})
	if e == nil || failed.Working != nil || len(failed.Output) != 0 || len(failed.Writes) != 0 {
		t.Fatalf("partial effects published: %+v %v", failed, e)
	}
}
func TestRunDeniedCapabilitiesAndErrorLocations(t *testing.T) {
	for _, code := range []string{`doc.edit({selector:"source",operation:"delete"})`, `doc.create("new.1")`, `doc.write("out.1")`} {
		result, e := script.Run(context.Background(), script.Request{Name: "denied.tengo", Code: []byte(code), Sources: input(t)}, script.Options{})
		if !errors.Is(e, script.ErrDenied) || !strings.Contains(e.Error(), "denied.tengo:1:") || result.Working != nil {
			t.Fatal(code, e)
		}
	}
	for _, code := range []string{`m := import("os")`, `m := import("../secrets")`, `doc.query("section[")`} {
		if _, e := script.Run(context.Background(), script.Request{Name: "invalid.tengo", Code: []byte(code), Sources: input(t)}, script.Options{}); e == nil {
			t.Fatal(code)
		}
	}
}
func TestRunBudgetsAndCycles(t *testing.T) {
	for _, tc := range []struct {
		code string
		o    script.Options
		want error
	}{
		{`for {}`, script.Options{Duration: 10 * time.Millisecond, MaxSteps: 10000000, MaxAllocs: 1000000}, context.DeadlineExceeded},
		{`n:=0;for {n++}`, script.Options{MaxAllocs: 100}, tengo.ErrObjectAllocLimit},
		{`doc.emit("first");doc.emit("second")`, script.Options{MaxHostCalls: 1}, script.ErrLimit},
		{`doc.emit("long output")`, script.Options{MaxOutputBytes: 4}, script.ErrLimit},
		{`x:=0`, script.Options{MaxScriptBytes: 1}, script.ErrLimit},
		{`x:=[[[1]]]`, script.Options{MaxNesting: 2}, script.ErrLimit},
		{`a:=[undefined];a[0]=a;doc.emit_json(a)`, script.Options{}, script.ErrOptions},
	} {
		result, e := script.Run(context.Background(), script.Request{Name: "limits.tengo", Code: []byte(tc.code)}, tc.o)
		if !errors.Is(e, tc.want) || result.Working != nil || len(result.Output) != 0 {
			t.Fatalf("%s: %+v %v", tc.code, result, e)
		}
	}
}
func TestRunCreatesAndRejectsStaleWrites(t *testing.T) {
	code := `doc.create("new.1", "NEW", "1");doc.edit({selector:"source[source-id=1]",operation:"append-section",heading:"NAME",text:"new - demo"});doc.write("out.1")`
	result, e := script.Run(context.Background(), script.Request{Name: "new.tengo", Code: []byte(code)}, script.Options{AllowEdits: true, AllowWriteRequests: true})
	if e != nil || result.Working == nil || len(result.Writes) != 1 {
		t.Fatal(result, e)
	}
	_, e = script.Run(context.Background(), script.Request{Code: []byte(code + `;doc.edit({selector:"source[source-id=1]",operation:"append-paragraph",text:"later"});doc.write("other.1")`)}, script.Options{AllowEdits: true, AllowWriteRequests: true})
	if !errors.Is(e, edit.ErrStale) {
		t.Fatal(e)
	}
}
func TestRunDeterministicRecordsAndInputErrors(t *testing.T) {
	request := script.Request{Name: "query.tengo", Sources: input(t), Code: []byte(`for match in doc.query("definition") {doc.emit_json({term:match.term,source:match.source,index:match.index})}`)}
	first, e := script.Run(context.Background(), request, script.Options{})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 10; i++ {
		next, e := script.Run(context.Background(), request, script.Options{})
		if e != nil || string(next.Output) != string(first.Output) {
			t.Fatal("unstable records", e)
		}
	}
	failure := errors.New("source read failed")
	request.InputError = failure
	if _, e := script.Run(context.Background(), request, script.Options{}); !errors.Is(e, failure) {
		t.Fatal(e)
	}
}

func TestAllocationLimitIsNotHeapQuota(t *testing.T) {
	// A native byte buffer counts as an object, rather than charging each byte.
	// This controlled 1 MiB example documents the engine limit's actual contract.
	result, e := script.Run(context.Background(), script.Request{Code: []byte(`data:=bytes(1048576);doc.emit(string(len(data)))`)}, script.Options{MaxAllocs: 10})
	if e != nil || string(result.Output) != "1048576" {
		t.Fatal(result, e)
	}
}
func TestScriptBroadcastDiagnosticsAndPreview(t *testing.T) {
	result, e := script.Run(context.Background(), script.Request{Sources: input(t), Code: []byte(`doc.edit({selector:"definition",operation:"find-replace-source",find:".",text:"!"});doc.emit(doc.preview("diff"))`)}, script.Options{AllowEdits: true})
	if e != nil || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "broadcast" || !strings.Contains(string(result.Output), "bytes [") {
		t.Fatal(result, e)
	}
}

func TestNativeCheckpointsControlFlowAndIsolation(t *testing.T) {
	for _, tc := range []struct{ code, want string }{
		{`x:=0;for i:=0;i<5;i++{if i==2{continue};x+=i};doc.emit(string(x))`, "8"},
		{`x:=0;for {x++;if x==3{break}};doc.emit(string(x))`, "3"},
		{`f:=func(n){if n==0{return 1};return n*f(n-1)};doc.emit(string(f(5)))`, "120"},
		{`m:={"__gtroff_checkpoint":1};doc.emit_json(m)`, "{\"__gtroff_checkpoint\":1}\n"},
	} {
		result, e := script.Run(context.Background(), script.Request{Name: "flow.tengo", Code: []byte(tc.code)}, script.Options{})
		if e != nil || string(result.Output) != tc.want || result.Stats.Steps < 1 {
			t.Fatal(tc.code, result, e)
		}
	}
	for _, code := range []string{`for {}`, `f:=[func(){for {}}];f[0]()`, `f:={run:func(){for {}}};f.run()`, `f:=true?func(){for {}}:func(){};f()`, `f:=func(){return f()};f()`} {
		result, e := script.Run(context.Background(), script.Request{Name: "steps.tengo", Code: []byte(code)}, script.Options{MaxSteps: 10})
		if !errors.Is(e, script.ErrLimit) || !strings.Contains(e.Error(), "steps.tengo:") || result.Stats.Steps != 11 {
			t.Fatal(code, result, e)
		}
	}
	for _, code := range []string{`__gtroff_checkpoint=func(){}`, `f:=func(__gtroff_fake){return 1}`} {
		if _, e := script.Run(context.Background(), script.Request{Code: []byte(code)}, script.Options{}); !errors.Is(e, script.ErrDenied) {
			t.Fatal(code, e)
		}
	}
	result, e := script.Run(context.Background(), script.Request{Code: []byte(`for i:=0;i<10;i++{};doc.emit("ok")`)}, script.Options{MaxHostCalls: 1})
	if e != nil || string(result.Output) != "ok" || result.Stats.HostCalls != 1 {
		t.Fatal(result, e)
	}
}

func TestSourceEditPreservesExplicitInvalidBytes(t *testing.T) {
	set, e := source.Read(context.Background(), strings.NewReader("\xff\n"), "bad.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	result, e := script.Run(context.Background(), script.Request{Sources: set, Code: []byte(`doc.edit({selector:"source[source-id=1]",operation:"replace-source",text:"\xff\n"});doc.emit(doc.source())`)}, script.Options{AllowEdits: true})
	if e != nil || string(result.Output) != "\xff\n" {
		t.Fatalf("source edit normalized bytes: %q %v", result.Output, e)
	}
	// Introducing malformed source fails instead of silently replacing its bytes.
	_, e = script.Run(context.Background(), script.Request{Sources: input(t), Code: []byte(`doc.edit({selector:"source[source-id=1]",operation:"replace-source",text:"\xff\n"})`)}, script.Options{AllowEdits: true})
	if !errors.Is(e, edit.ErrValidation) {
		t.Fatal(e)
	}
	for _, code := range []string{`doc.edit({selector:"source",operation:"replace-source",text:bytes(1)})`, `doc.edit({selector:"definition",operation:"delete",restrict:{nth:0}})`} {
		if _, e = script.Run(context.Background(), script.Request{Sources: input(t), Code: []byte(code)}, script.Options{AllowEdits: true}); !errors.Is(e, script.ErrOptions) {
			t.Fatal(code, e)
		}
	}
}
