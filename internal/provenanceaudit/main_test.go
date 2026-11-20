package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceBoundaries(t *testing.T) {
	// Latin-1 copyright, HTML entities and an empty alias dt are intentional.
	rawHTML := []byte("<meta charset=iso-8859-1>\n<dl>\n" +
		"<dt id='alias'>Request: <b>.alias</b></dt>\n" +
		"<dt id='actual'>Request: <b>.real</b></dt>\n" +
		"<dd><p>Copyright \xa9 A&nbsp;&amp; B.</p>\n<p>Keep 'quotes'.</p></dd></dl>")
	defs, err := manualDefinitions(rawHTML)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].anchor != "actual" || defs[0].line != 4 {
		t.Fatalf("alias or anchor attribution wrong: %+v", defs)
	}
	var catalogue bytes.Buffer
	groups := []group{
		{Description: "Copyright © A & B.\n\nKeep 'quotes'.", Contents: []item{{"Request", ".real", ""}}},
		{Description: "Copyright © A & B. Keep ‘quotes’.", Contents: []item{{"Request", ".different", ""}}},
		{Description: "Reference only.", Contents: []item{{"Macro", ".R", ""}}},
		{Description: "Unmatched.", Contents: []item{{"Macro", ".U", ""}}},
	}
	if err := json.NewEncoder(&catalogue).Encode(groups); err != nil {
		t.Fatal(err)
	}
	sources := []source{
		{evidence: evidence{Label: "manual"}, definitions: defs},
		{evidence: evidence{Label: "render"}, text: "Header Reference only. Footer"},
		{evidence: evidence{Label: "old"}, history: map[string]bool{"Copyright © A & B. Keep 'quotes'.": true, "Unmatched.": true}, reflowed: map[string]bool{"Copyright © A & B. Keep 'quotes'.": true, "Unmatched.": true}},
	}
	r, err := compare(catalogue.Bytes(), "revision", sources)
	if err != nil {
		t.Fatal(err)
	}
	if r.Groups != 4 || r.Items != 4 || r.MatchedGroups != 2 || r.UnmatchedGroups != 2 || len(r.Rows) != 4 {
		t.Fatalf("coverage wrong: %+v", r)
	}
	if m := r.Rows[0].Matches; len(m) != 1 || m[0].Source != "manual" || m[0].Anchor != "actual" || m[0].Line != 4 {
		t.Fatalf("definition location wrong: %+v", m)
	}
	if len(r.Rows[0].History) != 0 || len(r.Rows[3].History) != 1 || len(r.Rows[0].ReflowedHistory) != 1 {
		t.Fatalf("history must use bytes, not whitespace or semantic equivalence: %+v", r.Rows)
	}
	if len(r.Rows[1].Matches) != 0 || r.Rows[2].Matches[0].Kind != "reference-whitespace-substring" {
		t.Fatalf("typographic difference/reference boundary lost: %+v", r.Rows)
	}
	var again bytes.Buffer
	if err := json.NewEncoder(&again).Encode(r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(again.String(), "Copyright") {
		t.Fatal("report unexpectedly recopied description prose")
	}
}

func TestAuditCommand(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	cat := write("catalogue.json", "[\n {\"Description\":\"Break the line.\",\"Contents\":[{\"Type\":\"Request\",\"Token\":\".br\",\"Args\":\"\"}]}\n]")
	manual := write("manual.html", "<meta charset=utf-8><dl><dt id='br'>Request: .br</dt><dd>Break the line.</dd></dl>")
	args := []string{"-revision", "reviewed-commit", "-catalogue", cat, "-manual", "upstream=" + manual, "-history", "old=" + cat}
	var first, second bytes.Buffer
	if err := run(args, &first); err != nil {
		t.Fatal(err)
	}
	if err := run(args, &second); err != nil || !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("nondeterministic output: %v", err)
	}
	if bytes.Contains(first.Bytes(), []byte(dir)) {
		t.Fatal("machine path leaked into report")
	}
	var r report
	if err := json.Unmarshal(first.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Revision != "reviewed-commit" || r.MatchedGroups != 1 || r.Rows[0].DescriptionLine != 2 || len(r.Rows[0].History) != 1 || r.Sources[0].SHA256 == "" {
		t.Fatalf("report metadata wrong: %+v", r)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing-input", nil},
		{"extra-positional", append(append([]string{}, args...), "extra")},
		{"missing-file", []string{"-revision", "r", "-manual", "upstream=" + filepath.Join(dir, "missing")}},
		{"duplicate-within-kind", append(append([]string{}, args...), "-manual", "upstream="+manual)},
		{"duplicate-across-kinds", append(append([]string{}, args...), "-reference", "upstream="+manual)},
		{"bad-spec", []string{"-manual", "label"}},
		{"bad-catalogue", []string{"-revision", "r", "-manual", "u=" + manual, "-catalogue", write("bad.json", "null")}},
		{"no-definitions", []string{"-revision", "r", "-manual", "u=" + write("empty.html", "<p>No definitions.</p>"), "-catalogue", cat}},
		{"duplicate-anchor", []string{"-revision", "r", "-manual", "u=" + write("duplicate.html", "<dl><dt id='x'>Request: .a</dt><dd>A</dd><dt id='x'>Request: .b</dt><dd>B</dd></dl>"), "-catalogue", cat}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := run(tc.args, &out); err == nil || out.Len() != 0 {
				t.Fatalf("invalid evidence must fail before output: err=%v output=%s", err, &out)
			}
		})
	}
}

func TestRejectIncompleteCatalogue(t *testing.T) {
	for _, raw := range []string{
		`[]`, `{}`, `not json`,
		`[{"Description":"","Contents":[{"Type":"Request","Token":".br"}]}]`,
		`[{"Description":"A","Contents":[]}]`,
		`[{"Description":"A","Contents":[{"Token":".br"}]}]`,
		`[{"Description":"A","Description":"B","Contents":[{"Type":"Request","Token":".br"}]}]`,
	} {
		if _, err := compare([]byte(raw), "revision", nil); err == nil {
			t.Errorf("accepted incomplete/ambiguous catalogue %s", raw)
		}
	}
}
