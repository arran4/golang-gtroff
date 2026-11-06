package docdiff_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/docdiff"
	"github.com/arran4/golang-gtroff/source"
)

func input(t *testing.T, name, text string) docdiff.Input {
	t.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(text), name, source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return docdiff.Input{Name: name, Sources: set}
}
func compare(t *testing.T, before, after string, o docdiff.Options) *docdiff.Report {
	t.Helper()
	r, err := docdiff.Compare(context.Background(), input(t, "old.1", before), input(t, "new.1", after), o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

const header = ".TH TOOL 1 date source manual\n.SH ENVIRONMENT\n"

func def(name, body string) string { return ".TP\n.B " + name + "\n" + body + "\n" }
func TestFormattingAndSourceViewsAreSeparate(t *testing.T) {
	a := header + def("HOME", "Home directory.")
	b := ".TH \"TOOL\" \"1\" \"date\" \"source\" \"manual\"\r\n.SH \"ENVIRONMENT\"\r\n.TP 8\r\n\\fBHOME\\fR\r\nHome\r\ndirectory.\r\n"
	r := compare(t, a, b, docdiff.Options{})
	if !r.Complete || r.Status != "equal" || r.Different || len(r.Changes) > 0 {
		t.Fatalf("formatting false changes %+v", r)
	}
	r = compare(t, a, b, docdiff.Options{View: docdiff.Source})
	if !r.Different || len(r.Changes) != 1 || r.Changes[0].Kind != docdiff.Changed || r.Changes[0].Confidence != "source-bytes" {
		t.Fatal("source bytes hidden", r)
	}
}
func TestDefinitionsChangedAddedRemovedAndMoved(t *testing.T) {
	a := header + def("A", "First.") + def("B", "Second.")
	b := header + def("NEW", "New.") + def("A", "Changed.")
	r := compare(t, a, b, docdiff.Options{})
	counts := map[docdiff.ChangeKind]int{}
	for _, change := range r.Changes {
		counts[change.Kind]++
		if change.Before != nil && change.Before.Location.Span.Start.Filename != "old.1" {
			t.Fatal("old source location lost")
		}
		if change.After != nil && change.After.Location.Span.Start.Filename != "new.1" {
			t.Fatal("new source location lost")
		}
	}
	if counts[docdiff.Added] != 1 || counts[docdiff.Removed] != 1 || counts[docdiff.Changed] != 1 || counts[docdiff.Moved] != 0 {
		t.Fatalf("definition changes %+v", r.Changes)
	}
	r = compare(t, a, header+def("B", "Second.")+def("A", "First."), docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Kind != docdiff.Moved {
		t.Fatalf("reorder %+v", r.Changes)
	}
	r = compare(t, a+".SH OTHER\n", header+def("B", "Second.")+".SH OTHER\n"+def("A", "First."), docdiff.Options{})
	if len(r.Changes) != 1 || !r.Changes[0].Moved {
		t.Fatalf("scope move %+v", r.Changes)
	}
}
func TestInsertionDoesNotMoveExistingObjects(t *testing.T) {
	r := compare(t, header+def("A", "First.")+def("B", "Second."), header+def("NEW", "New.")+def("A", "First.")+def("B", "Second."), docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Kind != docdiff.Added {
		t.Fatalf("ordinal shift became moves %+v", r.Changes)
	}
}
func TestMetadataReferencesAndAmbiguity(t *testing.T) {
	a := header + ".UR https://old.example\nlink text\n.UE\n.MR printf 3\n"
	b := strings.Replace(a, "date", "newdate", 1)
	b = strings.Replace(b, "https://old.example", "https://new.example", 1)
	r := compare(t, a, b, docdiff.Options{})
	if len(r.Changes) != 2 {
		t.Fatalf("metadata/reference change %+v", r.Changes)
	}
	seen := map[docdiff.Kind]bool{}
	for _, change := range r.Changes {
		seen[change.Before.Kind] = true
	}
	if !seen[docdiff.Metadata] || !seen[docdiff.Reference] {
		t.Fatal("missing metadata/reference comparison")
	}
	duplicate := header + def("SAME", "One.") + def("SAME", "Two.")
	r = compare(t, duplicate, duplicate, docdiff.Options{})
	if r.Different || len(r.Ambiguities) > 0 {
		t.Fatal("identical source snapshot was ambiguous", r)
	}
	r = compare(t, duplicate, header+def("SAME", "Changed one.")+def("SAME", "Changed two."), docdiff.Options{})
	if len(r.Ambiguities) != 1 || r.Status != "ambiguous" || r.Different {
		t.Fatalf("duplicates were arbitrarily paired %+v", r)
	}
}
func TestUnsupportedAndIncompleteSource(t *testing.T) {
	r := compare(t, header+".UNKNOWN old\n", header+".UNKNOWN new\n", docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Before.Kind != docdiff.Opaque || r.Changes[0].Confidence != "uninterpreted-source" || len(r.Diagnostics) == 0 {
		t.Fatalf("unsupported source hidden %+v", r)
	}
	before := input(t, "old.1", header+".so missing.roff\n")
	after := input(t, "new.1", header)
	r, err := docdiff.Compare(context.Background(), before, after, docdiff.Options{})
	if !errors.Is(err, docdiff.ErrInput) || r.Complete || len(r.Changes) != 0 {
		t.Fatal("unavailable include invented removals", r, err)
	}
}
func TestIncludesRetainContextAndSourceIdentity(t *testing.T) {
	load := func(body string) docdiff.Input {
		set, err := source.Load(context.Background(), fstest.MapFS{"main.1": {Data: []byte(header + ".so defs.roff\n")}, "defs.roff": {Data: []byte(body)}}, "main.1", source.Options{})
		if err != nil {
			t.Fatal(err)
		}
		return docdiff.Input{Name: "main.1", Sources: set}
	}
	r, err := docdiff.Compare(context.Background(), load(def("HOME", "Old.")), load(def("HOME", "New.")), docdiff.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Changes) != 1 || r.Changes[0].Before.Kind != docdiff.Definition || r.Changes[0].Before.Location.Source.ID != 2 || r.Changes[0].After.Location.Span.Start.Filename != "defs.roff" {
		t.Fatalf("included change %+v", r.Changes)
	}
}
func TestBoundsAndCancellation(t *testing.T) {
	before := input(t, "old.1", header+def("A", "One."))
	after := input(t, "new.1", header+def("B", "Two."))
	for _, o := range []docdiff.Options{{MaxWork: 1}, {MaxNodes: 1}, {MaxObjects: 1}, {MaxChanges: 1}, {MaxDataBytes: 1}, {MaxResultBytes: 1}} {
		r, err := docdiff.Compare(context.Background(), before, after, o)
		if !errors.Is(err, docdiff.ErrLimit) || r.Complete {
			t.Fatalf("bound %+v -> %+v %v", o, r, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := docdiff.Compare(ctx, before, after, docdiff.Options{})
	if !errors.Is(err, context.Canceled) || r.Complete {
		t.Fatal("cancellation", r, err)
	}
}

func TestSubsectionsAndPreformattedData(t *testing.T) {
	before := ".TH TOOL 1\n.SH MAIN\nProse.\n.SS DETAIL\nInside.\n"
	after := strings.Replace(before, "Inside.", "Revised inside.", 1)
	r := compare(t, before, after, docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Before.Kind != docdiff.Subsection || r.Changes[0].Before.Location.Span.Start.Line != 4 {
		t.Fatal("subsection change cascaded or lost location", r)
	}
	before = header + ".nf\n one\n.fi\n"
	after = header + ".EX\n one\n.EE\n"
	r = compare(t, before, after, docdiff.Options{})
	if r.Different {
		t.Fatal("equivalent preformat wrappers changed content", r)
	}
	after = header + ".EX\n  one\n.EE\n"
	r = compare(t, before, after, docdiff.Options{})
	if !r.Different {
		t.Fatal("meaningful preformatted whitespace disappeared")
	}
}
func TestRepeatedLabelsAlignByUniqueContent(t *testing.T) {
	before := header + def("SAME", "One.") + def("SAME", "Two.")
	after := strings.Replace(before, ".TH TOOL 1", `.TH "TOOL" "1"`, 1)
	r := compare(t, before, after, docdiff.Options{})
	if r.Different || len(r.Ambiguities) > 0 {
		t.Fatal("unique unchanged content did not align repeated labels", r)
	}
}
func TestMetadataUsesPhysicalTitleLine(t *testing.T) {
	before := ".\\\" comment\n.\\\" another\n" + header
	after := strings.Replace(before, "date", "newdate", 1)
	r := compare(t, before, after, docdiff.Options{})
	if len(r.Changes) != 1 || r.Changes[0].Before.Kind != docdiff.Metadata || r.Changes[0].Before.Location.Span.Start.Line != 3 {
		t.Fatalf("metadata source: before=%+v after=%+v", r.Changes[0].Before, r.Changes[0].After)
	}
}
