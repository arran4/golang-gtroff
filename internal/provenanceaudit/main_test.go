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

func TestEffectiveEntryOwnership(t *testing.T) {
	raw := []byte(`[
 {"ID":"primary","Description":"Primary description.","Contents":[{"Type":"Macro","Token":".A"}]},
 {"Description":"Unrelated fallback.","Contents":[
  {"Type":"Macro","Token":".B","DescriptionRef":"primary"},
  {"Type":"Macro","Token":".C","ID":"inline","Description":"Independent summary."},
  {"Type":"Macro","Token":".D","DescriptionRef":"inline"}]},
 {"DescriptionRef":"primary","Contents":[{"Type":"Macro","Token":".E"}]}
]`)
	sources := []source{{evidence: evidence{Label: "manual", Kind: "manual-html"}, definitions: []definition{{text: "Primary description.", anchor: "primary", line: 10}, {text: "Unrelated fallback.", anchor: "wrong", line: 20}}}}
	r, err := compareEntries(raw, "reviewed-input", sources)
	if err != nil {
		t.Fatal(err)
	}
	if r.Schema != 2 || r.Groups != 3 || r.Entries != 5 || r.MatchedEntries != 3 || r.UnmatchedEntries != 2 || r.SourceEntries["manual"] != 3 {
		t.Fatalf("effective-entry coverage wrong: %+v", r)
	}
	for i, row := range r.Rows {
		if row.Index != i {
			t.Fatal("entry order changed")
		}
		if i == 0 || i == 1 || i == 4 {
			if row.DescriptionPath != "/0/Description" || row.DescriptionLine != 2 || len(row.Matches) != 1 || row.Matches[0].Anchor != "primary" {
				t.Fatalf("reference points at wrong owner: %+v", row)
			}
		} else if row.DescriptionPath != "/1/Contents/1/Description" || row.DescriptionLine != 5 || len(row.Matches) != 0 {
			t.Fatalf("inline owner wrong: %+v", row)
		}
	}
	if _, err := compare(raw, "reviewed-input", sources); err == nil {
		t.Fatal("legacy group comparison accepted heterogeneous descriptions")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "modern-history.json")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	history, err := load("modern="+file, "historical-catalogue")
	if err != nil || !history.history["Primary description."] || !history.history["Independent summary."] || history.history["Unrelated fallback."] {
		t.Fatal("history attributed an unused fallback instead of effective entries")
	}
}

func TestCommittedCorrectionReceipts(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "cmd", "gen", "in.json"))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := compareEntries(raw, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	readJSON := func(name string, out any) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, "docs", "provenance", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatal(err)
		}
	}
	var receipt entryReport
	readJSON("catalogue-corrections.json", &receipt)
	if receipt.Schema != 2 || receipt.CatalogueSHA256 != digest(raw) || receipt.Entries != actual.Entries || len(receipt.Rows) != len(actual.Rows) {
		t.Fatal("effective-description receipt is stale")
	}
	for i, row := range actual.Rows {
		got := receipt.Rows[i]
		if got.Index != i || got.Group != row.Group || got.Content != row.Content || got.SHA256 != row.SHA256 || got.DescriptionLine != row.DescriptionLine || got.DescriptionPath != row.DescriptionPath || got.Item != row.Item {
			t.Fatalf("effective-description owner changed at entry %d", i)
		}
	}
	var manifest struct {
		CatalogueSHA256 string     `json:"catalogue_sha256"`
		ChangedEntries  int        `json:"changed_entries"`
		Sources         []evidence `json:"sources"`
		Rows            []struct {
			Index   int    `json:"index"`
			SHA256  string `json:"description_sha256"`
			Path    string `json:"description_path"`
			Sources []struct {
				Source   string `json:"source"`
				Anchor   string `json:"anchor"`
				Line     int    `json:"heading_line"`
				BodyLine int    `json:"body_line"`
				Basis    string `json:"basis"`
			} `json:"sources"`
		} `json:"rows"`
	}
	readJSON("association-corrections.json", &manifest)
	if manifest.CatalogueSHA256 != digest(raw) || manifest.ChangedEntries != len(manifest.Rows) || len(manifest.Rows) != 179 {
		t.Fatal("association inventory is stale")
	}
	labels := make(map[string]bool)
	for _, s := range manifest.Sources {
		if s.Kind != "manual-html" || len(s.SHA256) != 64 || labels[s.Label] {
			t.Fatal("invalid source identity")
		}
		labels[s.Label] = true
	}
	seen := make(map[int]bool)
	for _, row := range manifest.Rows {
		if row.Index < 0 || row.Index >= len(actual.Rows) || seen[row.Index] || len(row.Sources) == 0 {
			t.Fatalf("invalid association entry %d", row.Index)
		}
		seen[row.Index] = true
		if row.SHA256 != actual.Rows[row.Index].SHA256 || row.Path != actual.Rows[row.Index].DescriptionPath {
			t.Fatalf("association entry %d has a stale literal owner", row.Index)
		}
		for _, citation := range row.Sources {
			if !labels[citation.Source] || citation.Anchor == "" || citation.Line <= 0 || citation.BodyLine <= citation.Line {
				t.Fatalf("invalid source citation for entry %d", row.Index)
			}
			switch citation.Basis {
			case "feature_heading", "shared_definition_body", "whitespace_identical_definition_body":
			default:
				t.Fatalf("unknown evidence basis %q", citation.Basis)
			}
		}
	}
}
