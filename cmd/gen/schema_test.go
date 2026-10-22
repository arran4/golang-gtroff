package main

import (
	_ "embed"
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
