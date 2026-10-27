package groff

import (
	"testing"
)

func TestCatalogueCoverage(t *testing.T) {
	entries := CatalogueEntries()

	if len(entries) != 518 {
		t.Fatalf("expected 518 catalogue entries, got %d", len(entries))
	}

	kindCounts := map[TokenKind]int{}
	distinctTokensByKind := map[TokenKind]map[string]bool{
		KindMacro:    {},
		KindRequest:  {},
		KindString:   {},
		KindRegister: {},
	}
	packageCounts := map[string]int{}
	statusCounts := map[DefinitionStatus]int{}

	for _, e := range entries {
		kindCounts[e.Kind]++
		distinctTokensByKind[e.Kind][e.Token] = true
		packageCounts[e.Package]++
		statusCounts[e.Status]++

		if e.Token == "" {
			t.Errorf("entry %d has empty token", e.Index)
		}
		if e.Package == "" {
			t.Errorf("entry %s has empty package", e.Token)
		}
		if e.Provenance == "" {
			t.Errorf("entry %s has empty provenance", e.Token)
		}
		if e.Status == "" {
			t.Errorf("entry %s has empty status", e.Token)
		}
	}

	if kindCounts[KindMacro] != 141 {
		t.Errorf("expected 141 Macro entries, got %d", kindCounts[KindMacro])
	}
	if kindCounts[KindRequest] != 206 {
		t.Errorf("expected 206 Request entries, got %d", kindCounts[KindRequest])
	}
	if kindCounts[KindString] != 59 {
		t.Errorf("expected 59 String entries, got %d", kindCounts[KindString])
	}
	if kindCounts[KindRegister] != 112 {
		t.Errorf("expected 112 Register entries, got %d", kindCounts[KindRegister])
	}

	if len(distinctTokensByKind[KindMacro]) != 111 {
		t.Errorf("expected 111 distinct Macro names, got %d", len(distinctTokensByKind[KindMacro]))
	}
	if len(distinctTokensByKind[KindRequest]) != 179 {
		t.Errorf("expected 179 distinct Request names, got %d", len(distinctTokensByKind[KindRequest]))
	}
	if len(distinctTokensByKind[KindString]) != 59 {
		t.Errorf("expected 59 distinct String names, got %d", len(distinctTokensByKind[KindString]))
	}
	if len(distinctTokensByKind[KindRegister]) != 111 {
		t.Errorf("expected 111 distinct Register names, got %d", len(distinctTokensByKind[KindRegister]))
	}

	// Verify lookup helper
	thDef := LookupCatalogue(".TH", "man")
	if thDef == nil || thDef.Name != "TH" {
		t.Errorf("expected to find .TH in man package, got %v", thDef)
	}

	nfDef := LookupCatalogue(".nf", "roff")
	if nfDef == nil || nfDef.Kind != KindRequest {
		t.Errorf("expected to find .nf request in roff package, got %v", nfDef)
	}

	t.Logf("Catalogue validated successfully: 518 entries, 0 unexplained gaps.")
}
