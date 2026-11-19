package author

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/man"
)

// AST returns the actual newly generated canonical syntax snapshot, with source
// positions into that generated text. It is not an original imported-source AST.
// No includes, user handlers or macros are executed during bounded validation.
func (p *Page) AST() (*groff.AST, error) {
	if p == nil || p.limits.MaxBlocks == 0 {
		return nil, ErrInvalid
	}
	model := &man.Document{Version: 1, Metadata: p.metadata, Nodes: []man.Node{{ID: 1, Kind: man.DocumentNode, State: man.State{Fill: true}}}}
	add := func(parent man.ID, kind man.Kind) man.ID {
		id := man.ID(len(model.Nodes) + 1)
		model.Nodes = append(model.Nodes, man.Node{ID: id, Parent: parent, Kind: kind, State: man.State{Fill: true}})
		model.Node(parent).Children = append(model.Node(parent).Children, id)
		return id
	}
	var appendBlocks func(man.ID, []Block)
	appendBlocks = func(parent man.ID, blocks []Block) {
		var list man.ID
		flowParent := parent
		for _, block := range blocks {
			if block.kind != definition {
				list = 0
			}
			switch block.kind {
			case paragraph:
				id := add(flowParent, man.Paragraph)
				model.Node(id).Runs = []man.Run{{Text: block.text}}
			case preformatted:
				id := add(flowParent, man.Preformatted)
				model.Node(id).Runs = []man.Run{{Text: block.text}}
			case definition:
				if list == 0 {
					list = add(flowParent, man.DefinitionList)
				}
				id := add(list, man.Item)
				model.Node(id).Tag = []man.Run{{Text: block.term, Style: man.Style{Bold: true}}}
				description := add(id, man.Paragraph)
				model.Node(description).Runs = []man.Run{{Text: block.text}}
			case subsection:
				id := add(parent, man.Subsection)
				model.Node(id).Heading = block.term
				appendBlocks(id, block.children)
				flowParent = id
			}
		}
	}
	for _, section := range p.sections {
		id := add(1, man.Section)
		model.Node(id).Heading = section.heading
		appendBlocks(id, section.blocks)
	}
	var data bytes.Buffer
	_, err := convert.WriteMan(&data, model, convert.Options{Strict: true, MaxOutputBytes: p.limits.MaxBytes, MaxNodes: 100000, MaxDepth: 16})
	if err != nil {
		if errors.Is(err, convert.ErrLimit) {
			err = errors.Join(ErrLimit, err)
		}
		return nil, err
	}
	tree, err := groff.ParseAST(bytes.NewReader(data.Bytes()), groff.WithPackage("man"), groff.WithRegistry(groff.NewRegistry()), groff.WithStrict(true), groff.WithSourceName("<author>"), groff.WithASTBytes(p.limits.MaxBytes), groff.WithASTNodes(100000), groff.WithASTDepth(16), groff.WithMaxLineLength(p.limits.MaxBytes))
	if err != nil {
		return tree, fmt.Errorf("%w: generated source failed bounded syntax validation: %w", ErrInvalid, err)
	}
	_, err = man.Build(tree, man.Options{Strict: true, MaxNodes: 100000, MaxBytes: p.limits.MaxBytes, MaxDepth: 16})
	if err != nil {
		return tree, fmt.Errorf("%w: generated source failed man validation: %w", ErrInvalid, err)
	}
	return tree, nil
}

// WriteRoff stages canonical generation and bounded syntax/man validation before
// touching writer. Invalid construction, encoding or budget errors leave it
// untouched. Writer failures can leave a partial stream. No files are opened.
func (p *Page) WriteRoff(writer io.Writer) error {
	if writer == nil || nilWriter(writer) {
		return fmt.Errorf("%w: nil writer", ErrInvalid)
	}
	tree, err := p.AST()
	if err != nil {
		return err
	}
	data := []byte(tree.Source())
	n, err := writer.Write(data)
	if n < 0 || n > len(data) {
		return errors.Join(fmt.Errorf("invalid author writer count %d", n), err)
	}
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

func nilWriter(writer io.Writer) bool {
	value := reflect.ValueOf(writer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}
