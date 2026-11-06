package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
	"github.com/arran4/golang-gtroff/query"
)

func TestQueryCLISharedSourcePolicy(t *testing.T) {
	fsys := fstest.MapFS{
		"main.1": {Data: []byte(".SH ENVIRONMENT\n.so child\n")},
		"child":  {Data: []byte(".TP\n.B TOOL_HOME\nHome directory.\n")},
	}
	for _, noIncludes := range []bool{false, true} {
		args := []string{"query"}
		if noIncludes {
			args = append(args, "--no-includes")
		}
		args = append(args, `section[name="ENVIRONMENT"] definition`, "docs/main.1")
		var out, stderr bytes.Buffer
		code := Run(args, Environment{Out: &out, Err: &stderr, OpenRoot: func(root string) (manlib.Root, io.Closer, error) {
			if root != "docs" {
				t.Fatal(root)
			}
			return manlib.Root{FS: fsys}, nil, nil
		}})
		if code != ExitOK || stderr.Len() != 0 {
			t.Fatalf("query failed: %d %s", code, stderr.String())
		}
		var result query.Result
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := 1
		if noIncludes {
			want = 0
		}
		if len(result.Matches) != want || !result.Complete {
			t.Fatal("include default not shared with query")
		}
		if !noIncludes && (result.Matches[0].Term != "TOOL_HOME" || result.Matches[0].Source.Name != "child") {
			t.Fatal("query lost included definition provenance")
		}
	}
}
func TestQueryCLIRejectsInvalidOptionsBeforeReading(t *testing.T) {
	for _, args := range [][]string{
		{"query"}, {"query", "broken >"}, {"query", "--format", "yaml", "line"}, {"query", "--view", "wrong", "line"},
		{"query", "--fields", "made-up", "line"}, {"query", "--format", "csv", "--fields", "made-up", "line"},
		{"query", "--max-results", "0", "line"}, {"query", "--max-work", "0", "line"}, {"query", "--max-result-bytes", "0", "line"}, {"query", "line", "a", "b"},
	} {
		var out, stderr bytes.Buffer
		code := Run(args, Environment{In: panicReader{}, Out: &out, Err: &stderr, OpenRoot: func(string) (manlib.Root, io.Closer, error) {
			t.Fatal("opened invalid query")
			return manlib.Root{}, nil, nil
		}})
		if code != ExitUsage || out.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("invalid query options accepted: %v code %d", args, code)
		}
	}
	for _, format := range []string{"json", "jsonl", "text", "raw", "grep", "csv", "tsv"} {
		if code := Run([]string{"query", "--format", format, "paragraph"}, Environment{In: strings.NewReader("text\n"), Out: shortWriter{}}); code != ExitIO {
			t.Fatal("query output failure ignored", format)
		}
	}
}
