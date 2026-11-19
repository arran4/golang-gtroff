package man_test

import (
	"encoding/json"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/man"
)

func TestSourceNodesErrorsAndIsolation(t *testing.T) {
	tree, err := groff.ParseAST(strings.NewReader(".SH NAME\r\ntext\r\n.\\\" trailing\r\n"), groff.WithPackage("man"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := man.Build(tree, man.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lines, err := document.SourceNodes(1)
	if err != nil {
		t.Fatal(err)
	}
	lines[0] = nil
	again, err := document.SourceNodes(1)
	if err != nil || again[0] == nil {
		t.Fatal("returned slice aliases source index", err)
	}
	for _, id := range []man.ID{0, ^man.ID(0)} {
		if _, err := document.SourceNodes(id); err == nil {
			t.Fatalf("invalid ID %d accepted", id)
		}
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	var decoded man.Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Syntax() != nil {
		t.Fatal("JSON invented a source tree")
	}
	if _, err := decoded.SourceNodes(1); err == nil {
		t.Fatal("JSON invented physical lines")
	}
	document.Nodes[0].Origin.Span.End.Offset = -1
	if _, err := document.SourceNodes(1); err == nil {
		t.Fatal("invalid model span accepted")
	}
	if tree.Source() != ".SH NAME\r\ntext\r\n.\\\" trailing\r\n" {
		t.Fatal("model edit changed syntax")
	}
}

func TestSourceNodesPartialInput(t *testing.T) {
	tree, err := groff.ParseAST(strings.NewReader(".SH NAME\ntext\nremaining\n"), groff.WithASTBytes(12))
	if err == nil {
		t.Fatal("expected source budget failure")
	}
	document, err := man.Build(tree, man.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lines, err := document.SourceNodes(1)
	if err != nil {
		t.Fatal(err)
	}
	var raw strings.Builder
	for _, line := range lines {
		raw.WriteString(line.Raw())
	}
	if document.Syntax() != tree || raw.String() != tree.Source() || tree.SourceComplete() {
		t.Fatal("partial source misrepresented")
	}
	empty, err := groff.ParseAST(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	document, err = man.Build(empty, man.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lines, err = document.SourceNodes(1)
	if err != nil || len(lines) != 0 {
		t.Fatal("empty source acquired lines", err)
	}
}

func TestSourceNodesExactBytes(t *testing.T) {
	for _, source := range []string{
		".SH NAME\r\ntext\r\n.\\\" trailing\r\n",
		".SH NAME\ninvalid \xff\n.UNKNOWN \xfe",
		".SH NAME\n雪\n\n",
	} {
		tree, err := groff.ParseAST(strings.NewReader(source), groff.WithPackage("man"), groff.WithSourceName("bytes.1"))
		if err != nil {
			t.Fatal(err)
		}
		document, err := man.Build(tree, man.Options{})
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range document.Nodes {
			lines, err := document.SourceNodes(node.ID)
			if err != nil {
				t.Fatal(err)
			}
			var raw strings.Builder
			for _, line := range lines {
				if line.Span().Start.Filename != "bytes.1" || tree.Node(line.ID()) != line {
					t.Fatal("source ownership changed")
				}
				raw.WriteString(line.Raw())
			}
			if raw.String() != document.Raw(node.ID) {
				t.Fatal("raw bytes changed")
			}
		}
	}
}
