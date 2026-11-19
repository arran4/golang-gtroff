package script_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/golang-gtroff/script"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleRun() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".SH ENVIRONMENT\n.TP\nTOOL_HOME\nDirectory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	program := `for match in doc.query("section[name=ENVIRONMENT] definition") {doc.emit_json({term:match.term,file:match.source,line:match.line})}`
	result, err := script.Run(ctx, script.Request{Name: "extract.tengo", Code: []byte(program), Sources: set}, script.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(result.Output))
	// Output: {"file":"tool.1","line":2,"term":"TOOL_HOME"}
}
func ExampleRun_edit() {
	ctx := context.Background()
	set, err := source.Read(ctx, strings.NewReader(".TP\nTOOL_HOME\nOld directory.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	result, err := script.Run(ctx, script.Request{Name: "edit.tengo", Sources: set, Code: []byte(`doc.edit({selector:"definition[term=TOOL_HOME]",operation:"replace-description",text:"New directory."});doc.emit(doc.extract("definition","raw"))`)}, script.Options{AllowEdits: true})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(result.Output))
	fmt.Println("pending files:", len(result.Working.Changes()))
	fmt.Println("original retained:", strings.Contains(set.File(1).AST().Source(), "Old directory."))
	// Output:
	// .TP
	// TOOL_HOME
	// \&New directory.
	// pending files: 1
	// original retained: true
}
func ExampleRun_create() {
	result, err := script.Run(context.Background(), script.Request{Code: []byte(`doc.create("new.1","NEW","1");doc.edit({selector:"source[source-id=1]",operation:"append-section",heading:"NAME",text:"new - demo"});doc.write("out.1")`)}, script.Options{AllowEdits: true, AllowWriteRequests: true})
	if err != nil {
		panic(err)
	}
	// These are staged intents, not file writes. The application must authorize
	// the destination and enforce original-file guards at its storage boundary.
	fmt.Println("new:", result.Created, "requests:", len(result.Writes))
	fmt.Println(result.Writes[0].Destination)
	// Output:
	// new: true requests: 1
	// out.1
}
