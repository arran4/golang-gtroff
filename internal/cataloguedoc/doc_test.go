package cataloguedoc

import "testing"

func TestReferencesPreserveLiteralAndOwnershipAcrossOrder(t *testing.T) {
	input := []Text{{Ref: "shared"}, {Literal: "unrelated"}, {ID: "shared", Literal: "  A\r\nB  "}, {}}
	r, err := New(input)
	if err != nil {
		t.Fatal(err)
	}
	input[2].Literal = "changed"
	for _, i := range []int{0, 2} {
		text, owner, err := r.Resolve(i)
		if err != nil || text != "  A\r\nB  " || owner != 2 {
			t.Fatalf("reference lost ownership or literal bytes: %q %d %v", text, owner, err)
		}
	}
	if text, owner, err := r.Resolve(3); err != nil || text != "" || owner != 3 {
		t.Fatal("legacy unnamed empty literal was rejected")
	}
	for _, i := range []int{-1, 4} {
		if _, _, err := r.Resolve(i); err == nil {
			t.Fatal("invalid index accepted")
		}
	}
}

func TestRejectInvalidDocumentation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		texts []Text
	}{
		{"missing", []Text{{Ref: "missing"}}},
		{"duplicate", []Text{{ID: "same", Literal: "A"}, {ID: "same", Literal: "B"}}},
		{"empty-owner", []Text{{ID: "empty", Literal: " \n"}}},
		{"bad-name", []Text{{ID: "bad name", Literal: "A"}}},
		{"literal-and-ref", []Text{{ID: "a", Literal: "A"}, {Literal: "B", Ref: "a"}}},
		{"self-cycle", []Text{{ID: "a", Ref: "a"}}},
		{"chain", []Text{{ID: "a", Literal: "A"}, {ID: "b", Ref: "a"}, {Ref: "b"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.texts); err == nil {
				t.Fatal("invalid documentation accepted")
			}
		})
	}
	var nilResolver *Resolver
	if _, _, err := nilResolver.Resolve(0); err == nil {
		t.Fatal("nil resolver accepted")
	}
}
