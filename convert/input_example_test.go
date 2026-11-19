package convert_test

import (
	"errors"
	"fmt"
	"io"

	conversion "github.com/arran4/golang-gtroff/convert"
)

type failedInputExample struct{}

func (failedInputExample) Read(p []byte) (int, error) {
	return copy(p, "<p>kept</p>\n"), io.ErrUnexpectedEOF
}

func ExampleInputError() {
	input, err := conversion.ImportHTML(failedInputExample{}, conversion.Options{SourceName: "draft.html"})
	fmt.Printf("source=%q document=%v\n", input.Source, input.Document != nil)
	var inputErr *conversion.InputError
	if errors.As(err, &inputErr) {
		p := inputErr.Position
		fmt.Printf("readerError=%v position=%s:%d:%d byte=%d\n", errors.Is(err, io.ErrUnexpectedEOF), p.Filename, p.Line, p.Column, p.Offset)
	}
	// Output:
	// source="<p>kept</p>\n" document=false
	// readerError=true position=draft.html:2:1 byte=12
}
