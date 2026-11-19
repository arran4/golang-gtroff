package groff

import (
	"reflect"
	"testing"
)

func TestMatcherReservesRequiredArguments(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bind     func([]string, ...bool) ([]string, error)
		args     []string
		optional []bool
		want     []string
	}{
		{"forward optional prefix", ForwardMatcher, []string{"path"}, []bool{true, false}, []string{"", "path"}},
		{"backward optional suffix", BackwardMatcher, []string{"path"}, []bool{false, true}, []string{"path", ""}},
		{"forward gap", ForwardMatcher, []string{"a", "b"}, []bool{false, true, false}, []string{"a", "", "b"}},
		{"backward gap", BackwardMatcher, []string{"a", "b"}, []bool{false, true, false}, []string{"a", "", "b"}},
		{"forward explicit empty", ForwardMatcher, []string{"", "path"}, []bool{true, false}, []string{"", "path"}},
		{"backward explicit empty", BackwardMatcher, []string{"path", ""}, []bool{false, true}, []string{"path", ""}},
		{"legacy forward remainder", ForwardMatcher, []string{"a", "b", "c"}, []bool{false, true}, []string{"a", "b c"}},
		{"legacy backward remainder", BackwardMatcher, []string{"a", "b", "c"}, []bool{true, false}, []string{"a b", "c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.bind(tc.args, tc.optional...)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
	for _, bind := range []func([]string, ...bool) ([]string, error){ForwardMatcher, BackwardMatcher} {
		if _, err := bind(nil, true, false, true); err == nil {
			t.Fatal("missing required argument accepted")
		}
	}
}

func TestPnRequiredPathWithOptionalPunctuation(t *testing.T) {
	for _, tc := range []struct{ input, before, path, after string }{
		{`.Pn path`, "", "path", ""},
		{`.Pn path .`, "", "path", "."},
		{`.Pn ( path )`, "(", "path", ")"},
		{`.Pn "" path ""`, "", "path", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			tok, err := ParseLineWithPackage(tc.input, "man")
			if err != nil {
				t.Fatal(err)
			}
			pn := tok.(*TokenMacroPn)
			if pn.Punct != tc.before || pn.Path != tc.path || pn.Punct2 != tc.after {
				t.Fatalf("wrong binding: %+v", pn)
			}
			again, err := ParseLineWithPackage(pn.String(), "man")
			if err != nil || !reflect.DeepEqual(tok, again) {
				t.Fatalf("canonical round trip: %+v %v", again, err)
			}
		})
	}
}
