package man_test

import (
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func TestConversionLiteralArgumentsAndCombinedFont(t *testing.T) {
	source := ".TH \"A \\(dqquote\\(dq \\e path\" 1\n.SH NAME\n.UR \"https://example.com/?q=\\(dqvalue\\(dq\"\nlink\n.UE\n.MR \"na\\(dqme\" 3\n.PP\n\\f[CBI]combined\\f[R]\n"
	tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := man.Build(tree, man.Options{Strict: true})
	if err != nil {
		t.Fatalf("%v %+v", err, d.Diagnostics)
	}
	if d.Metadata.Title != `A "quote" \ path` {
		t.Fatalf("metadata not decoded: %q", d.Metadata.Title)
	}
	foundLink, foundReference, foundFont := false, false, false
	for _, n := range d.Nodes {
		for _, r := range n.Runs {
			if r.Text == "link" {
				foundLink = true
				if r.Link != `https://example.com/?q="value"` {
					t.Fatalf("link not decoded %q", r.Link)
				}
			}
			if r.Reference != "" {
				foundReference = true
				if r.Reference != `na"me(3)` {
					t.Fatalf("reference not decoded %q", r.Reference)
				}
			}
			if r.Text == "combined" {
				foundFont = true
				if r.Style != (man.Style{Mono: true, Bold: true, Italic: true}) {
					t.Fatalf("combined font lost: %+v", r.Style)
				}
			}
		}
	}
	if !foundLink || !foundReference || !foundFont {
		t.Fatal("expected semantic runs absent")
	}
	if d.Source() != source {
		t.Fatal("semantic decoding changed source bytes")
	}
}
