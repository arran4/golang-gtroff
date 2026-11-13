package groff_test

import (
	"reflect"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
)

func TestCatalogueDescriptionSubjects(t *testing.T) {
	for _, tc := range []struct{ token, subject string }{
		{".warn", "warning"}, {".sy", "shell"}, {".cp", "compatibility"},
		{".psbb", "bounding box"}, {".lsm", "line"}, {".de", "macro"},
		{".tr", "translat"}, {".hlm", "hyphen"}, {".ad", "adjust"},
		{".device", "device-control"}, {".devicem", "device-control"},
		{".ds", "string"}, {".ds1", "string"}, {".brp", "adjustment"},
	} {
		t.Run(tc.token, func(t *testing.T) {
			entry := groff.LookupCatalogue(tc.token, "roff")
			if entry == nil || !strings.Contains(strings.ToLower(entry.Description), tc.subject) || strings.Contains(entry.Description, "pre-grohtml") {
				t.Fatalf("wrong description subject for %s: %+v", tc.token, entry)
			}
		})
	}
	for _, entry := range groff.CatalogueEntries() {
		for _, category := range []string{"Macro:", "Request:", "Register:", "String:", "Escape:"} {
			if strings.HasPrefix(entry.Description, category) {
				t.Fatalf("entry %d still describes the next heading: %q", entry.Index, entry.Description)
			}
		}
	}
}

func TestDisplayVariantDescriptions(t *testing.T) {
	want := map[string]string{"L": "left", "I": "indent", "C": "center", "B": "block", "R": "right"}
	for _, entry := range groff.CatalogueEntries() {
		if entry.Kind != "Macro" || entry.Token != ".DS" || entry.Package != "ms" {
			continue
		}
		if subject, ok := want[entry.ArgsRaw]; ok {
			if !strings.Contains(strings.ToLower(entry.Description), subject) {
				t.Fatalf("display placement %s has wrong description: %q", entry.ArgsRaw, entry.Description)
			}
			delete(want, entry.ArgsRaw)
		}
	}
	if len(want) != 0 {
		t.Fatalf("unverified placement variants: %v", want)
	}
}

func TestCorrectedPackageMetadataPreservesQuotedRegionConstructors(t *testing.T) {
	entries := groff.CatalogueEntries()
	if len(entries) != 518 || entries[0].Token != ".QS" || entries[505].Token != ".QS" {
		t.Fatal("existing catalogue order changed")
	}
	for _, index := range []int{0, 505} {
		if entries[index].Package != "ms" || entries[index].HasCollision {
			t.Fatalf("quoted region falsely attributed to man: %+v", entries[index])
		}
	}
	man, err := groff.ParseLineWithPackage(".QS", "man")
	if err != nil {
		t.Fatal(err)
	}
	ms, err := groff.ParseLineWithPackage(".QS", "ms")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.(*groff.TokenManQS); !ok {
		t.Fatalf("legacy man constructor changed: %T", man)
	}
	if _, ok := ms.(*groff.TokenMsQS); !ok || reflect.TypeOf(man) == reflect.TypeOf(ms) {
		t.Fatalf("existing distinct concrete types collapsed: %T / %T", man, ms)
	}
	for _, token := range []string{`\n[.br]`, `\n[ps4html]`, `\n[urx]`, `\n[.hlm]`} {
		entry := groff.LookupCatalogue(token, "roff")
		if entry == nil || entry.Package != "roff" {
			t.Fatalf("core register mislabelled as a macro-package register: %s %+v", token, entry)
		}
	}
}

func TestArgumentVariantDescriptions(t *testing.T) {
	for _, tc := range []struct{ token, args, subject string }{
		{".ft", "nnn", "numeric position"},
		{".nr", "ident value", "creating the register"},
		{".nr", "ident +value", "leading sign"},
		{".nr", "ident -value", "leading sign"},
		{".nr", "ident value incr", "auto-incrementing"},
	} {
		found := false
		for _, entry := range groff.CatalogueEntries() {
			if entry.Token == tc.token && entry.ArgsRaw == tc.args {
				found = true
				if !strings.Contains(entry.Description, tc.subject) {
					t.Errorf("%s %s describes a different argument form: %q", tc.token, tc.args, entry.Description)
				}
			}
		}
		if !found {
			t.Errorf("missing variant %s %s", tc.token, tc.args)
		}
	}
}
