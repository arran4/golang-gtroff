package manlib_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/renderhtml"
)

func ExampleHandler() {
	files := fstest.MapFS{"man1/tool.1": {Data: []byte(".TH TOOL 1\n.SH NAME\ntool \\- inspect a document\n")}}
	index, err := manlib.Build(context.Background(), []manlib.Root{{Label: "example", FS: files}}, manlib.Limits{})
	if err != nil {
		panic(err)
	}
	handler, err := manlib.Handler(index, manlib.HandlerOptions{Render: renderhtml.Options{Assets: renderhtml.OmitFonts}})
	if err != nil {
		panic(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "http://localhost/search?q=inspect", nil))
	fmt.Println(response.Code, strings.Contains(response.Body.String(), "tool(1)"))
	// Output: 200 true
}
