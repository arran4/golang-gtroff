package extract

import (
	"bytes"
	"errors"

	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
	"github.com/arran4/golang-gtroff/renderhtml"
	"github.com/arran4/golang-gtroff/source"
)

func (b *builder) render(model *man.Document, node *man.Node, record *Record) (string, error) {
	// Renderer validation scans the complete source model per selected fragment.
	// Charge that cost before invoking the existing independently bounded renderer.
	if err := b.meter.spend(len(model.Nodes)*4 + len(record.Text) + 1); err != nil {
		return "", err
	}
	id := node.ID
	if record.Kind == query.Term {
		// A virtual term is rendered as a paragraph of its existing styled runs. No
		// definition description is added to a term-only selection.
		root := cloneNode(model.Nodes[0])
		root.Children = []man.ID{2}
		root.ID = 1
		root.Parent = 0
		paragraph := man.Node{ID: 2, Parent: 1, Kind: man.Paragraph, Runs: cloneRuns(node.Tag), State: node.State, Origin: man.Origin{ASTIDs: record.ASTIDs, Span: record.Span}}
		model = &man.Document{Version: 1, Metadata: model.Metadata, Nodes: []man.Node{root, paragraph}}
		id = 2
	}
	limit := b.options.MaxBytes - b.bytes
	if limit < 1 {
		return "", ErrLimit
	}
	var out bytes.Buffer
	rendered, err := renderhtml.RenderSelection(&out, model, &man.Selection{Roots: []man.ID{id}}, renderhtml.Options{Fragment: true, NoAnchors: true, Assets: renderhtml.OmitFonts, MaxOutputBytes: limit})
	for _, d := range rendered.Diagnostics {
		b.addDiagnostic(source.Diagnostic{Source: record.Source.ID, Code: "html", Diagnostic: d})
	}
	if err != nil {
		if errors.Is(err, renderhtml.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return "", err
	}
	if err := b.meter.spend(out.Len()); err != nil {
		return "", err
	}
	return out.String(), nil
}
