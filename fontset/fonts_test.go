package fontset

import "testing"

func TestDefaultsAreIndependentAndComplete(t *testing.T) {
	fonts := Default()
	if err := fonts.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(fonts.Regular) == 0 || len(fonts.Bold) == 0 || len(fonts.Italic) == 0 || len(fonts.Mono) == 0 {
		t.Fatal("missing bundled face")
	}
	first := fonts.Regular[0]
	fonts.Regular[0] ^= 0xff
	if again := Default(); again.Regular[0] != first {
		t.Fatal("default data was shared")
	}
	inherited := (Set{}).WithDefaults()
	inherited.Regular[0] ^= 0xff
	if again := (Set{}).WithDefaults(); again.Regular[0] != first {
		t.Fatal("fallback data was shared")
	}
	for _, data := range [][]byte{[]byte("not a font"), make([]byte, MaxFaceBytes+1)} {
		bad := Default()
		bad.Regular = data
		if bad.Validate() == nil {
			t.Fatal("invalid or excessive font accepted")
		}
	}
}
