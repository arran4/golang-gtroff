package author_test

import (
	"bytes"
	"testing"

	"github.com/arran4/golang-gtroff/author"
	"github.com/arran4/golang-gtroff/man"
)

func FuzzLiteralAuthoring(f *testing.F) {
	for _, text := range []string{"plain text", ".sy never-run\n'so never-read\n", `quoted " and \fB literal`, "漢字 café\tline\r\nnext", ""} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 4096 {
			t.Skip()
		}
		p, err := author.NewManPage("tool", "1")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.AddSection("TEXT", author.Paragraph(text), author.Definition("VALUE", text), author.Preformatted(text)); err != nil {
			return
		}
		tree, err := p.AST()
		if err != nil {
			t.Fatalf("accepted literal did not generate supported syntax: %v", err)
		}
		if _, err := man.Build(tree, man.Options{Strict: true}); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := p.WriteRoff(&out); err != nil || out.String() != tree.Source() {
			t.Fatal("generation changed between calls", err)
		}
	})
}
