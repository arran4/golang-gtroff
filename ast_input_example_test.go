package groff_test

import (
	"errors"
	"fmt"
	"io"

	groff "github.com/arran4/golang-gtroff"
)

type partialASTExampleReader struct{ done bool }

func (r *partialASTExampleReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.ErrUnexpectedEOF
	}
	r.done = true
	return copy(p, "kept\n"), io.ErrUnexpectedEOF
}
func ExampleAST_SourceComplete() {
	tree, err := groff.ParseAST(&partialASTExampleReader{}, groff.WithSourceName("draft.1"))
	fmt.Printf("source=%q complete=%v readerError=%v\n", tree.Source(), tree.SourceComplete(), errors.Is(err, io.ErrUnexpectedEOF))
	var inputErr *groff.ASTInputError
	if errors.As(err, &inputErr) {
		fmt.Printf("position=%s:%d:%d\n", inputErr.Position.Filename, inputErr.Position.Line, inputErr.Position.Column)
	}
	// Output:
	// source="kept\n" complete=false readerError=true
	// position=draft.1:2:1
}
