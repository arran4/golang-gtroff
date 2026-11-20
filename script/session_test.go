package script_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/arran4/golang-gtroff/edit"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/scriptapi"
)

func session(t *testing.T, o script.SessionOptions) *script.Session {
	t.Helper()
	s, e := script.NewSession(context.Background(), script.Request{Sources: input(t)}, o)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func accept(t *testing.T, s *script.Session) {
	t.Helper()
	if e := s.Publish(context.Background(), func(context.Context, *script.Result) error { return nil }); e != nil {
		t.Fatal(e)
	}
}
func eval(t *testing.T, s *script.Session, code string) *script.Result {
	t.Helper()
	r, e := s.Eval(context.Background(), script.Cell{Code: []byte(code)})
	if e != nil {
		t.Fatal(e)
	}
	accept(t, s)
	return r
}
func TestSessionNativeGlobalsClosuresAndRollback(t *testing.T) {
	s := session(t, script.SessionOptions{})
	eval(t, s, `values := [1]; alias := values; m := {v: values}; m.self = m
 make := func() { n := 0; return [func() { n++; return n }, func() { return n }] }
 fns := make(); count := func(x) { return 99 }; saved_emit := doc.emit_json; copied_doc := copy(doc)`)
	if got := string(eval(t, s, `values[0] = 2; saved_emit([alias[0], m.v[0], fns[0](), fns[1](), count([])])`).Output); got != "[2,2,1,1,99]\n" {
		t.Fatal(got)
	}
	if r, e := s.Eval(context.Background(), script.Cell{Code: []byte(`values[0] = 8; m.self.v[0] = 7; fns[0](); temp := 12; doc.emit("partial"); bad := 1/0`)}); e == nil || r.Working != nil || len(r.Output) != 0 {
		t.Fatal(r, e)
	}
	if got := string(eval(t, s, `copied_doc.emit_json([alias[0], m.v[0], fns[1]()])`).Output); got != "[2,2,1]\n" {
		t.Fatal(got)
	}
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.emit_json(temp)`)}); e == nil {
		t.Fatal("failed symbol escaped")
	}
	// Compiler errors must not retain newly defined symbols either.
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`fresh := 1; missing()`)}); e == nil {
		t.Fatal("expected compiler error")
	}
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.emit_json(fresh)`)}); e == nil {
		t.Fatal("compiler symbol escaped")
	}
	eval(t, s, `make_rec := func(){ f:=undefined; f=func(n){if n==0{return 0};return 1+f(n-1)};return f }; rec:=make_rec()`)
	if got := string(eval(t, s, `doc.emit_json(rec(4))`).Output); got != "4\n" {
		t.Fatal(got)
	}

}
func TestSessionSlicesImmutableWrappersAndCycles(t *testing.T) {
	s := session(t, script.SessionOptions{})
	eval(t, s, `a := [1,2,3]; b := a[1:]; imm := immutable(a); m := {x: 1}; im := immutable(m); raw := bytes("abc"); tail := raw[1:]
 cycle := [undefined]; cycle[0] = cycle`)
	got := string(eval(t, s, `b[0] = 8; m.x = 9; doc.emit_json([a[1], imm[1], im.x, string(raw)])`).Output)
	if got != "[8,8,9,\"abc\"]\n" {
		t.Fatal(got)
	}
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`cycle[0] = 4; a[0] = 5; b[0] = 6; m.x = 7; x := 1/0`)}); e == nil {
		t.Fatal("expected failure")
	}
	got = string(eval(t, s, `doc.emit_json([a[0], a[1], im.x, string(raw), is_array(cycle[0])])`).Output)
	if got != "[1,8,9,\"abc\",true]\n" {
		t.Fatal(got)
	}
	// Top-level compiler-private iterators must not prevent persistence, while
	// functions referencing global block variables retain their original slots.
	eval(t, s, `fn := undefined; for x in [10,20] { fn = func() { return x } }`)
	if got := string(eval(t, s, `doc.emit_json(fn())`).Output); got != "20\n" {
		t.Fatal(got)
	}
}
func TestSessionDocumentOriginalBaselineAndPendingRetry(t *testing.T) {
	s := session(t, script.SessionOptions{Cell: script.Options{AllowEdits: true, AllowWriteRequests: true}})
	before := string(eval(t, s, `old := doc.query("definition"); doc.emit(doc.source())`).Output)
	eval(t, s, `doc.edit({selector:"definition",restrict:{nth:1},operation:"replace-description",text:"First."})`)
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.edit({selector:"definition",restrict:{nth:2},operation:"replace-description",text:"Failed."}); x:=1/0`)}); e == nil {
		t.Fatal("expected failure")
	}
	r, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.edit({selector:"definition",restrict:{nth:2},operation:"replace-description",text:"Second."}); doc.emit_json(old[0].text); doc.write("out.1")`)})
	if e != nil || len(r.Writes) != 1 || len(r.Working.Changes()) != 1 {
		t.Fatal(r, e)
	}
	change := r.Working.Changes()[0]
	if string(change.Before) != before || !strings.Contains(string(change.After), "First.") || !strings.Contains(string(change.After), "Second.") {
		t.Fatal(change)
	}
	if !strings.Contains(string(r.Output), "Old directory.") {
		t.Fatal("captured query changed", string(r.Output))
	}
	if _, e = s.Eval(context.Background(), script.Cell{Code: []byte(`x:=1`)}); !errors.Is(e, script.ErrPending) {
		t.Fatal(e)
	}
	sentinel := errors.New("host publication failed")
	var captured edit.CommitBatch
	e = s.Publish(context.Background(), func(ctx context.Context, r *script.Result) error {
		return r.Working.Commit(ctx, func(_ context.Context, b edit.CommitBatch) error { captured = b; return sentinel })
	})
	if !errors.Is(e, sentinel) || len(captured.Changes) != 1 || string(captured.Changes[0].Before) != before {
		t.Fatal(e, captured)
	}
	pending, e := s.Pending(context.Background())
	if e != nil || !bytes.Equal(pending.Output, r.Output) || !bytes.Equal(pending.Writes[0].Data, r.Writes[0].Data) {
		t.Fatal(pending, e)
	}
	// The failed publisher and isolated Pending access must not acknowledge edits.
	p, e := pending.Working.Plan(context.Background(), edit.Command{Select: query.Select(query.Definition), Restrict: edit.Restriction{Nth: 1}, Operation: edit.ReplaceDescription, Text: "outside", Rewrite: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = pending.Working.Apply(p); e != nil {
		t.Fatal(e)
	}
	next, _ := s.Pending(context.Background())
	if strings.Contains(string(next.Writes[0].Data), "outside") {
		t.Fatal("pending alias")
	}
	if e = s.Publish(context.Background(), func(ctx context.Context, r *script.Result) error {
		return r.Working.Commit(ctx, func(_ context.Context, b edit.CommitBatch) error {
			if string(b.Changes[0].Before) != before {
				t.Fatal("baseline reset")
			}
			return nil
		})
	}); e != nil {
		t.Fatal(e)
	}
	result := eval(t, s, `doc.edit({selector:"definition",restrict:{nth:1},operation:"replace-description",text:"Third.",rewrite:true})`)
	if strings.Contains(string(result.Working.Changes()[0].Before), "Old directory.") || !strings.Contains(string(result.Working.Changes()[0].Before), "First.") {
		t.Fatal("successful commit baseline not adopted")
	}
}
func TestSessionLimitsFailureRecoveryAndStaleWrites(t *testing.T) {
	s := session(t, script.SessionOptions{Cell: script.Options{MaxSteps: 20, AllowEdits: true, AllowWriteRequests: true}})
	eval(t, s, `empty_loop := func(){for{}}`)
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`empty_loop()`)}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	eval(t, s, `x:=1`)
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.write("out.1");doc.edit({selector:"definition",restrict:{nth:1},operation:"replace-description",text:"stale"})`)}); !errors.Is(e, edit.ErrStale) {
		t.Fatal(e)
	}
	if got := string(eval(t, s, `doc.emit_json(x)`).Output); got != "1\n" {
		t.Fatal(got)
	}
	s = session(t, script.SessionOptions{MaxCells: 1})
	eval(t, s, `x:=1`)
	if _, e := s.Eval(context.Background(), script.Cell{}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	s = session(t, script.SessionOptions{MaxStateBytes: 1000})
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`x:=1`)}); !errors.Is(e, script.ErrLimit) {
		t.Fatal(e)
	}
	s = session(t, script.SessionOptions{Cell: script.Options{Duration: 20 * time.Millisecond, MaxSteps: 10000000, MaxAllocs: 1000000, MaxHostCalls: 100000}, MaxSessionHostCalls: 1000000})
	eval(t, s, `retained_loop := func(){for{doc.query("definition")}}`)
	if _, e := s.Eval(context.Background(), script.Cell{Code: []byte(`retained_loop()`)}); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	eval(t, s, `doc.emit("ok")`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Eval(ctx, script.Cell{}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	var _ scriptapi.Session = s
}

func TestSessionCombinedOutputIntentBudget(t *testing.T) {
	set := input(t)
	n := len(set.File(1).AST().Source())
	s, e := script.NewSession(context.Background(), script.Request{Sources: set}, script.SessionOptions{Cell: script.Options{AllowWriteRequests: true}, MaxSessionOutputBytes: 2*n - 1})
	if e != nil {
		t.Fatal(e)
	}
	result, e := s.Eval(context.Background(), script.Cell{Code: []byte(`doc.emit(doc.source());doc.write("out.1")`)})
	if !errors.Is(e, script.ErrLimit) || result.Working != nil || len(result.Output) > 0 || len(result.Writes) > 0 {
		t.Fatal(result, e)
	}
}
