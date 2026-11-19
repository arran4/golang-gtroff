package agentskill_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/agentskill"
	"github.com/arran4/golang-gtroff/extract"
	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/query"
)

func TestRecipeCapabilitiesAndActualSelections(t *testing.T) {
	bundle, e := agentskill.Lookup(agentskill.Name)
	if e != nil {
		t.Fatal(e)
	}
	capabilities := map[string]bool{}
	for _, command := range cli.GetInventory() {
		for _, capability := range command.Capabilities {
			capabilities[capability] = true
		}
	}
	for _, required := range bundle.Manifest().Contracts.Capabilities {
		if !capabilities[required] {
			t.Fatal("unsupported recipe capability", required)
		}
	}
	input := ".TH TOOL 1\n.SH ENVIRONMENT\n.TP\n.B TOOL_HOME\nData directory.\n.TP\nPATH\nSearch paths.\n"
	fsys := fstest.MapFS{"document.1": {Data: []byte(input)}}
	run := func(args []string) []byte {
		t.Helper()
		var out, stderr bytes.Buffer
		code := cli.Run(args, cli.Environment{Out: &out, Err: &stderr, OpenRoot: func(string) (manlib.Root, io.Closer, error) { return manlib.Root{FS: fsys}, nil, nil }})
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("%v -> %d %s", args, code, stderr.String())
		}
		return out.Bytes()
	}
	var selection query.Result
	b := run([]string{"query", "--format", "json", `section[name="ENVIRONMENT"] definition`, "document.1"})
	if e = json.Unmarshal(b, &selection); e != nil {
		t.Fatal(e)
	}
	if !selection.Complete || len(selection.Matches) != 2 || selection.Matches[0].Term != "TOOL_HOME" {
		t.Fatal(string(b))
	}
	first := selection.Matches[0]
	if first.Source.Name != "document.1" || first.Source.Digest != fmt.Sprintf("%x", sha256.Sum256([]byte(input))) || first.Span.Start.Line != 3 || first.Raw != input[first.Span.Start.Offset:first.Span.End.Offset] {
		t.Fatal("citation/source contract", first)
	}
	for _, args := range [][]string{{"query", "--format", "json", `definition[term="TOOL_HOME"]`, "document.1"}, {"query", "--view", "syntax", "--format", "json", `macro[macro=".B"][arg.0="TOOL_HOME"]`, "document.1"}, {"query", "--format", "json", `source[source-id=1] > document > section[name="ENVIRONMENT"]`, "document.1"}} {
		var result query.Result
		if e = json.Unmarshal(run(args), &result); e != nil || len(result.Matches) != 1 {
			t.Fatal(args, result, e)
		}
	}
	var extracted extract.Report
	b = run([]string{"extract", "--format", "json", `definition[term="TOOL_HOME"]`, "document.1"})
	if e = json.Unmarshal(b, &extracted); e != nil || len(extracted.Records) != 1 || extracted.Records[0].Description != "Data directory.\n" || !extracted.Complete || !extracted.QueryComplete {
		t.Fatal(string(b), e)
	}
	csvData := run([]string{"extract", "--format", "csv", "--fields", "term,description,file,line", "definition", "document.1"})
	rows, e := csv.NewReader(bytes.NewReader(csvData)).ReadAll()
	if e != nil || len(rows) != 3 || len(rows[1]) != 4 || rows[1][0] != "TOOL_HOME" || rows[1][1] != "Data directory.\n" || rows[1][2] != "document.1" || rows[1][3] != "3" {
		t.Fatal(rows, e)
	}

	raw := run([]string{"extract", "--format", "raw", `section[name="ENVIRONMENT"]`, "document.1"})
	if string(raw) != input[strings.Index(input, ".SH ENVIRONMENT"):] {
		t.Fatal("raw canonicalized", string(raw))
	}
	if string(fsys["document.1"].Data) != input {
		t.Fatal("recipe modified source")
	}
}
