package edit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/source"
)

func working(t *testing.T, s string, o Options) *Working {
	t.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(s), "page.1", o.Source)
	if err != nil {
		t.Logf("original parse: %v", err)
	}
	w, err := New(set, o)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func selector(t *testing.T, s string) query.Selector {
	t.Helper()
	q, e := query.ParseSelector(s)
	if e != nil {
		t.Fatal(e)
	}
	return q
}
func data(t *testing.T, w *Working) string {
	t.Helper()
	b, e := w.Data(w.RootName())
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func execute(t *testing.T, w *Working, c Command) *Plan {
	t.Helper()
	p, e := w.Plan(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Apply(p); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestExactSourceAndRestrictions(t *testing.T) {
	original := ".\\\" retained\r\n.UNKNOWN x\r\n.SH ONE\r\nfirst\r\n.SH ONE\r\nsecond\r\n\xff\r\n"
	for _, tc := range []struct {
		name string
		r    Restriction
		want string
	}{{"all", Restriction{}, strings.ReplaceAll(original, "ONE", "TWO")}, {"nth", Restriction{Nth: 2}, strings.Replace(original, ".SH ONE\r\nsecond", ".SH TWO\r\nsecond", 1)}, {"range", Restriction{From: 1, To: 1}, strings.Replace(original, "ONE", "TWO", 1)}, {"first", Restriction{FirstN: 1}, strings.Replace(original, "ONE", "TWO", 1)}, {"explicit all", Restriction{All: true}, strings.ReplaceAll(original, "ONE", "TWO")}} {
		t.Run(tc.name, func(t *testing.T) {
			w := working(t, original, Options{})
			p, e := w.Plan(context.Background(), Command{Select: query.Select(query.Section), Operation: FindReplaceSource, Find: "ONE", Text: "TWO", Restrict: tc.r})
			if e != nil {
				t.Fatal(e)
			}
			if data(t, w) != original {
				t.Fatal("planning mutated source")
			}
			if e = w.Apply(p); e != nil {
				t.Fatal(e)
			}
			if got := data(t, w); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			if len(p.Changes()) != 1 {
				t.Fatal("missing changes")
			}
		})
	}
}
func TestStrictAndNoMatches(t *testing.T) {
	for _, r := range []Restriction{{}, {All: true}, {From: 1, To: 2}, {FirstN: 2}} {
		w := working(t, ".SH A\na\n.SH B\nb\n", Options{Strict: true})
		_, e := w.Plan(context.Background(), Command{Select: query.Select(query.Section), Operation: Delete, Restrict: r})
		if !errors.Is(e, ErrSelection) {
			t.Fatalf("%+v: %v", r, e)
		}
	}
	w := working(t, ".SH A\na\n.SH B\nb\n", Options{Strict: true})
	execute(t, w, Command{Select: query.Select(query.Section), Operation: Delete, Restrict: Restriction{Nth: 2}})
	if data(t, w) != ".SH A\na\n" {
		t.Fatal(data(t, w))
	}
	w = working(t, "text\n", Options{})
	p := execute(t, w, Command{Select: query.Select(query.Definition), Operation: Delete})
	if len(p.Diagnostics()) != 1 || data(t, w) != "text\n" {
		t.Fatal("no match policy")
	}
	for _, r := range []Restriction{{Nth: 2}, {From: 0, To: 1}, {From: 2, To: 1}, {Nth: 1, All: true}, {FirstN: -1}} {
		_, e := w.Plan(context.Background(), Command{Select: Root(), Operation: Delete, Restrict: r})
		if e == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
func TestStaleOverlapAndCopies(t *testing.T) {
	w := working(t, ".SH A\nx\n", Options{})
	c := Command{Select: Root(), Operation: InsertAfter, Text: "tail\n"}
	p, e := w.Plan(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	foreign := working(t, data(t, w), Options{})
	if !errors.Is(foreign.Apply(p), ErrStale) {
		t.Fatal("foreign plan")
	}
	patches := p.Patches()
	patches[0].Data[0] = 'z'
	changes := p.Changes()
	changes[0].After[0] = 'z'
	if e = w.Apply(p); e != nil {
		t.Fatal(e)
	}
	if !strings.HasSuffix(data(t, w), "tail\n") {
		t.Fatal("accessor mutated plan")
	}
	if !errors.Is(w.Apply(p), ErrStale) {
		t.Fatal("replayed plan")
	}
	_, e = w.PlanPatches(context.Background(), []Patch{{Source: w.root, Digest: "bad", Start: 0, End: 0}})
	if !errors.Is(e, ErrStale) {
		t.Fatal(e)
	}
	_, e = w.Plan(context.Background(), Command{Select: query.Select(query.Any), Operation: Delete})
	if !errors.Is(e, ErrOverlap) {
		t.Fatal(e)
	}
	original := data(t, w)
	d := digest(original)
	for _, ps := range [][]Patch{{{Source: w.root, Digest: d, Start: 0, End: 5}, {Source: w.root, Digest: d, Start: 2, End: 2, Data: []byte("x")}}, {{Source: w.root, Digest: d, Start: -1, End: 0}}} {
		if _, e = w.PlanPatches(context.Background(), ps); e == nil {
			t.Fatal("invalid patches accepted")
		}
		if data(t, w) != original {
			t.Fatal("failed plan mutated")
		}
	}
}
func TestBatchRequeriesAndRollback(t *testing.T) {
	w := working(t, ".SH A\nx\n", Options{})
	p, e := w.Plan(context.Background(), Command{Select: Root(), Operation: AppendSection, Heading: "B", Text: "body"}, Command{Select: selector(t, `section[name="B"]`), Operation: FindReplaceSource, Find: "body", Text: "changed"})
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Apply(p); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(data(t, w), "changed") {
		t.Fatal(data(t, w))
	}
	before := data(t, w)
	_, e = w.Plan(context.Background(), Command{Select: Root(), Operation: InsertAfter, Text: "okay\n"}, Command{Select: Root(), Operation: ReplaceSource, Text: ".SH \"unclosed\n"})
	if e == nil {
		t.Fatal("invalid batch accepted")
	}
	if data(t, w) != before {
		t.Fatal("partial batch published")
	}
}
func TestCapturedIncludesAndOwnership(t *testing.T) {
	fsys := fstest.MapFS{"page.1": {Data: []byte(".so child\n.so child\n")}, "child": {Data: []byte(".SH ENVIRONMENT\n.TP\nVALUE\noriginal\n")}}
	set, e := source.Load(context.Background(), fsys, "page.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	w, e := New(set, Options{})
	if e != nil {
		t.Fatal(e)
	}
	c := Command{Select: query.Select(query.Definition), Operation: FindReplaceSource, Find: "original", Text: "updated"}
	if _, e = w.Plan(context.Background(), c); !errors.Is(e, ErrTarget) {
		t.Fatal(e)
	}
	w, e = New(set, Options{IncludeEdits: true})
	if e != nil {
		t.Fatal(e)
	}
	p := execute(t, w, c)
	child, _ := w.Data("child")
	if strings.Count(string(child), "updated") != 1 || len(p.Patches()) != 1 {
		t.Fatalf("repeated include applied twice: %q %+v", child, p.Patches())
	}
	if data(t, w) != ".so child\n.so child\n" {
		t.Fatal("root changed")
	}
	if len(p.Diagnostics()) < 2 {
		t.Fatal("missing mass/repeated diagnostics")
	}
	if _, e = w.Plan(context.Background(), Command{Select: Root(), Operation: InsertAfter, Text: ".so uncaptured\n"}); !errors.Is(e, ErrValidation) {
		t.Fatal(e)
	}
	if len(fsys) != 2 {
		t.Fatal("unexpected IO")
	}
}
func TestMoveAndInsert(t *testing.T) {
	w := working(t, ".SH A\na\n.SH B\nb\n.SH C\nc\n", Options{})
	execute(t, w, Command{Select: selector(t, `section[name="A"]`), Operation: MoveAfter, Destination: selector(t, `section[name="B"]`)})
	if data(t, w) != ".SH B\nb\n.SH A\na\n.SH C\nc\n" {
		t.Fatal(data(t, w))
	}
	execute(t, w, Command{Select: selector(t, `section[name="C"]`), Operation: InsertBefore, Text: ".\\\" inserted\n"})
	if !strings.Contains(data(t, w), ".\\\" inserted\n.SH C") {
		t.Fatal(data(t, w))
	}
	_, e := w.Plan(context.Background(), Command{Select: Root(), Operation: MoveBefore, Destination: query.Select(query.Paragraph), Restrict: Restriction{Nth: 1}})
	if !errors.Is(e, ErrSelection) {
		t.Fatal(e)
	}
}
func TestDescriptionsAndLiteralAuthoring(t *testing.T) {
	const original = ".SH ENVIRONMENT\r\n.TP\r\n.B VALUE\r\nold body\r\n.\\\" trailing\r\n.TP\r\nNEXT\r\nlast\r\n"
	w := working(t, original, Options{})
	execute(t, w, Command{Select: selector(t, `definition[term="VALUE"]`), Operation: ReplaceDescription, Text: ".danger \\ literal"})
	got := data(t, w)
	if !strings.Contains(got, ".B VALUE\r\n\\&.danger \\e literal\r\n.\\\" trailing\r\n.TP\r\nNEXT") {
		t.Fatalf("description boundary: %q", got)
	}
	execute(t, w, Command{Select: Root(), Operation: AppendSection, Heading: ".RAW", Text: ".so secret\n'control"})
	if strings.Contains(data(t, w), "\n.so secret") || !strings.Contains(data(t, w), ".RAW") {
		t.Fatal(data(t, w))
	}
	w = working(t, ".SH A\n.TP\nTERM\n.B styled\n", Options{})
	c := Command{Select: query.Select(query.Definition), Operation: ReplaceDescription, Text: "literal"}
	if _, e := w.Plan(context.Background(), c); !errors.Is(e, ErrNeedsRewrite) {
		t.Fatal(e)
	}
	c.Rewrite = true
	p := execute(t, w, c)
	if len(p.Diagnostics()) != 1 {
		t.Fatal(p.Diagnostics())
	}
	w = working(t, ".de X\nbody\n", Options{})
	if _, e := w.Plan(context.Background(), Command{Select: Root(), Operation: AppendParagraph, Text: "literal"}); !errors.Is(e, ErrTarget) {
		t.Fatal(e)
	}
}
func TestLimitsCancellationAndMalformedPreservation(t *testing.T) {
	w := working(t, ".SH A\ntext\n", Options{MaxBytes: 32})
	if _, e := w.Plan(context.Background(), Command{Select: Root(), Operation: InsertAfter, Text: strings.Repeat("x", 40)}); !errors.Is(e, ErrLimit) {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := w.Plan(ctx, Command{Select: Root(), Operation: Delete}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	w = working(t, ".TH \"unclosed\n.SH A\ntext\n", Options{})
	execute(t, w, Command{Select: query.Select(query.Section), Operation: FindReplaceSource, Find: "text", Text: "updated"})
	if !strings.HasPrefix(data(t, w), ".TH \"unclosed\n") {
		t.Fatal("malformed untouched line lost")
	}
}

func TestDescriptionCommentsAndMissingTag(t *testing.T) {
	original := ".TP\nTERM\n.\\\" leading\nold\n.\\\" trailing\n"
	w := working(t, original, Options{})
	execute(t, w, Command{Select: query.Select(query.Definition), Operation: ReplaceDescription, Text: "new"})
	if data(t, w) != ".TP\nTERM\n.\\\" leading\n\\&new\n.\\\" trailing\n" {
		t.Fatal(data(t, w))
	}
	w = working(t, ".TP\nTERM\none\n.\\\" interior\ntwo\n", Options{})
	c := Command{Select: query.Select(query.Definition), Operation: ReplaceDescription, Text: "new"}
	if _, e := w.Plan(context.Background(), c); !errors.Is(e, ErrNeedsRewrite) {
		t.Fatal(e)
	}
	w = working(t, ".TP\n", Options{})
	if _, e := w.Plan(context.Background(), c); !errors.Is(e, ErrTarget) {
		t.Fatal(e)
	}
}
func TestNoOpAndDisabledIncludesRetainSource(t *testing.T) {
	const original = ".TH \"unclosed\n.so missing\n.SH A\ntext\n"
	w := working(t, original, Options{Source: source.Options{NoIncludes: true}})
	p := execute(t, w, Command{Select: Root(), Operation: ReplaceSource, Text: original})
	if len(p.Changes()) != 0 || len(p.Patches()) != 0 {
		t.Fatal("no-op changed bytes")
	}
	execute(t, w, Command{Select: query.Select(query.Section), Operation: FindReplaceSource, Find: "text", Text: "updated"})
	if e := w.Sources().Edges(); len(e) != 1 || e[0].State != "disabled" {
		t.Fatal(e)
	}
}
func TestMoveBroadcastPreservesDistinctIdenticalNodes(t *testing.T) {
	fsys := fstest.MapFS{"page.1": {Data: []byte(".so child\n.so child\n.SH DEST\ntail\n")}, "child": {Data: []byte(".SH SAME\nx\n.SH SAME\nx\n")}}
	set, e := source.Load(context.Background(), fsys, "page.1", source.Options{})
	if e != nil {
		t.Fatal(e)
	}
	w, e := New(set, Options{IncludeEdits: true})
	if e != nil {
		t.Fatal(e)
	}
	p := execute(t, w, Command{Select: selector(t, `section[name=SAME]`), Operation: MoveBefore, Destination: selector(t, `section[name=DEST]`)})
	if strings.Count(data(t, w), ".SH SAME") != 2 {
		t.Fatal("distinct physical nodes lost or repeated", data(t, w))
	}
	if len(p.Patches()) != 4 {
		t.Fatalf("physical delete/insert count %d", len(p.Patches()))
	}
	b, _ := w.Data("child")
	if len(b) != 0 {
		t.Fatalf("child not emptied: %q", b)
	}
}
func TestNamedFragmentEOFAndInvalidLiteral(t *testing.T) {
	w := working(t, ".SH A\nold", Options{})
	execute(t, w, Command{Select: Root(), Operation: AppendDefinition, Term: "VALUE", Text: "description"})
	if !strings.HasPrefix(data(t, w), ".SH A\nold\n") {
		t.Fatal(data(t, w))
	}
	for _, c := range []Command{{Select: Root(), Operation: AppendParagraph, Text: "\xff"}, {Select: Root(), Operation: AppendSection, Heading: "bad\n.so file"}, {Select: Root(), Operation: AppendDefinition, Term: "bad\x00"}} {
		if _, e := w.Plan(context.Background(), c); !errors.Is(e, ErrOptions) {
			t.Fatal(e)
		}
	}
}
