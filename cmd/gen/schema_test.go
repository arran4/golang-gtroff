package main

import (
	_ "embed"
	"strings"
	"testing"
)

func TestNormalizeCatalog(t *testing.T) {
	cat, err := NormalizeCatalog(inJson)
	if err != nil {
		t.Fatalf("NormalizeCatalog failed: %v", err)
	}

	if len(cat.Entries) != 518 {
		t.Fatalf("expected 518 entries, got %d", len(cat.Entries))
	}

	summary, err := cat.GenerateCoverageSummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary.UnexplainedGaps != 0 {
		t.Errorf("expected 0 unexplained gaps, got %d", summary.UnexplainedGaps)
	}
	for status, want := range map[string]int{supportGenerated: 330, supportCustom: 17, supportPreserved: 171} {
		if got := summary.EntriesByParseStatus[status]; got != want {
			t.Errorf("unexpected %s classification: got %d, want %d", status, got, want)
		}
	}
	if summary.EntriesByKind["Macro"] != 141 {
		t.Errorf("expected 141 Macro entries, got %d", summary.EntriesByKind["Macro"])
	}
	if summary.EntriesByKind["Request"] != 206 {
		t.Errorf("expected 206 Request entries, got %d", summary.EntriesByKind["Request"])
	}
	if summary.EntriesByKind["String"] != 59 {
		t.Errorf("expected 59 String entries, got %d", summary.EntriesByKind["String"])
	}
	if summary.EntriesByKind["Register"] != 112 {
		t.Errorf("expected 112 Register entries, got %d", summary.EntriesByKind["Register"])
	}
	if summary.DistinctTokensByKind["Macro"] != 111 {
		t.Errorf("expected 111 distinct Macro tokens, got %d", summary.DistinctTokensByKind["Macro"])
	}
	if summary.DistinctTokensByKind["Request"] != 179 {
		t.Errorf("expected 179 distinct Request tokens, got %d", summary.DistinctTokensByKind["Request"])
	}
	if summary.DistinctTokensByKind["String"] != 59 {
		t.Errorf("expected 59 distinct String tokens, got %d", summary.DistinctTokensByKind["String"])
	}
	if summary.DistinctTokensByKind["Register"] != 111 {
		t.Errorf("expected 111 distinct Register tokens, got %d", summary.DistinctTokensByKind["Register"])
	}

	t.Logf("Coverage summary: %+v", summary)
}

func TestDocumentationReferencesAndLegacyParserNamespace(t *testing.T) {
	input := []byte(`[
 {"ID":"ms-quotation","Description":"An existing quotation description.","Contents":[{"Type":"Macro","Token":".QS","Args":"","Package":"ms","ParserPackage":"man"}]},
 {"Description":"Unrelated fallback.","Contents":[{"Type":"Macro","Token":".QS","Args":"","Package":"ms","DescriptionRef":"ms-quotation"}]}
]`)
	cat, err := NormalizeCatalog(input)
	if err != nil {
		t.Fatal(err)
	}
	for i, entry := range cat.Entries {
		if entry.Index != i || entry.Package != PackageMs || entry.HasCollision || entry.Description != "An existing quotation description." {
			t.Fatalf("incorrect source metadata: %+v", entry)
		}
	}
	if cat.Entries[0].GoName != "TokenManQS" || cat.Entries[1].GoName != "TokenMsQS" {
		t.Fatal("metadata correction renamed existing Go bindings")
	}
	macros, err := prepareMacrosData(cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(macros.Definitions) != 2 {
		t.Fatal("distinct existing concrete token types were collapsed")
	}
	lines := prepareParselineData(cat, macros)
	if len(lines.Dispatches) != 1 || !lines.Dispatches[0].HasVariants || lines.Dispatches[0].DefaultParser != "TokenManQS" || lines.Dispatches[0].MsParser != "TokenMsQS" {
		t.Fatalf("existing package dispatch changed: %+v", lines.Dispatches)
	}
	report := buildSupportReport(cat, macros, lines)
	if report.Summary.UnexplainedGaps != 0 || report.Entries[0].ParserPackage != PackageMan || report.Entries[0].Package != PackageMs || report.Entries[1].ParserPackage != "" {
		t.Fatalf("source family and executable context not distinguished: %+v", report)
	}
}

func TestInvalidDocumentationAndParserNamespace(t *testing.T) {
	for _, input := range []string{
		`[{"DescriptionRef":"missing","Contents":[{"Type":"Macro","Token":".NEW"}]}]`,
		`[{"ID":"same","Description":"A","Contents":[{"ID":"same","Description":"B","Type":"Macro","Token":".NEW"}]}]`,
		`[{"Description":"A","DescriptionRef":"missing","Contents":[{"Type":"Macro","Token":".NEW"}]}]`,
		`[{"Contents":[{"Type":"Macro","Token":".NEW","ParserPackage":"invalid"}]}]`,
	} {
		if _, err := NormalizeCatalog([]byte(input)); err == nil {
			t.Errorf("invalid documentation/binding accepted: %s", input)
		}
	}
}

func TestSharedMultilineDescriptionIsEmittedOnce(t *testing.T) {
	cat, err := NormalizeCatalog([]byte(`[
 {"ID":"shared","Description":"First line.\nSecond line.","Contents":[{"Type":"Request","Token":".NEW","Args":""}]},
 {"DescriptionRef":"shared","Contents":[{"Type":"Request","Token":".NEW","Args":""}]}
]`))
	if err != nil {
		t.Fatal(err)
	}
	macros, err := prepareMacrosData(cat)
	if err != nil || len(macros.Definitions) != 1 || strings.Count(macros.Definitions[0].DescriptionCommented, "First line.") != 1 {
		t.Fatalf("shared prose duplicated in generated comments: %+v %v", macros, err)
	}
}
