package cli

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strconv"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
)

func TestRenderASTBlockAndCapturedLineSelection(t *testing.T) {
	for _, tc := range []struct {
		name, raw, text string
		captured        bool
	}{
		{"indent", ".RS 4\nchosen\n.RE\n", "chosen", false},
		{"no-fill", ".nf\nchosen\n.fi\n", "chosen", false},
		{"example", ".EX\nchosen\n.EE\n", "chosen", false},
		{"captured", ".de1 SAVED\nstored\n..\n", ".de1 SAVED", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := ".TH OUTER 1\n.SH REAL\nbefore\n" + tc.raw + "outside\n"
			tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"))
			if err != nil {
				t.Fatal(err)
			}
			block := tree.Root().FindKind(groff.ASTBlock)[0]
			targets := []groff.NodeID{block.ID()}
			if tc.captured {
				for _, line := range block.FindKind(groff.ASTLine) {
					targets = append(targets, line.ID())
					for _, inline := range line.FindKind(groff.ASTInline) {
						targets = append(targets, inline.ID())
					}
				}
			}
			for _, format := range []string{"html", "png", "jpg"} {
				t.Run(format, func(t *testing.T) {
					var first []byte
					for _, id := range targets {
						args := []string{"render", "--format", format, "--node", strconv.FormatUint(uint64(id), 10)}
						if format == "html" {
							args = append(args, "--fragment", "--assets", "omit-fonts")
						} else {
							args = append(args, "--single", "--width", "400", "--margin", "24")
						}
						var out, diags bytes.Buffer
						code := Run(args, Environment{In: strings.NewReader(source), Out: &out, Err: &diags})
						if code != ExitOK {
							t.Fatalf("AST ID %d: exit %d: %s", id, code, diags.String())
						}
						if format == "html" {
							if html := out.String(); !strings.Contains(html, tc.text) || strings.Contains(html, ">before<") || strings.Contains(html, ">outside<") {
								t.Fatal("HTML selected wrong content")
							}
						} else {
							decoded, actual, err := image.Decode(bytes.NewReader(out.Bytes()))
							if err != nil || decoded.Bounds().Dx() != 400 || actual != map[string]string{"png": "png", "jpg": "jpeg"}[format] {
								t.Fatal("invalid selected image", err)
							}
						}
						if first == nil {
							first = bytes.Clone(out.Bytes())
						} else if !bytes.Equal(first, out.Bytes()) {
							t.Fatalf("AST ID %d selected a different block", id)
						}
					}
				})
			}
		})
	}
}
