package query_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/query"
)

func TestQueryOutputContracts(t *testing.T) {
	result, err := query.Execute(context.Background(), environmentSet(t), query.Select(query.Definition).Where(query.TermText, query.Equal, "TOOL_HOME"), query.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []query.Format{query.JSON, query.JSONL, query.PlainText, query.RawSource, query.Grep, query.CSV, query.TSV} {
		var out bytes.Buffer
		if err := result.Write(&out, format, nil); err != nil {
			t.Fatal(err)
		}
		switch format {
		case query.JSON:
			var decoded query.Result
			if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || len(decoded.Matches) != 1 || !decoded.Complete {
				t.Fatal("full JSON lost report", err)
			}
		case query.JSONL:
			var decoded query.Match
			if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || decoded.Index != 1 || decoded.Source.Name != "terms.roff" {
				t.Fatal("JSONL lost standalone provenance", err)
			}
		case query.RawSource:
			if out.String() != result.Matches[0].Raw {
				t.Fatal("raw serializer changed source")
			}
		case query.PlainText:
			if !strings.Contains(out.String(), "Home directory.") {
				t.Fatal("text lost description")
			}
		case query.Grep:
			if !strings.HasPrefix(out.String(), "terms.roff:1:1:") || strings.Count(out.String(), "\n") != 1 {
				t.Fatal("grep output lost location/line boundary")
			}
		case query.CSV, query.TSV:
			parser := csv.NewReader(&out)
			if format == query.TSV {
				parser.Comma = '\t'
			}
			rows, err := parser.ReadAll()
			if err != nil || len(rows) != 2 || rows[1][1] != "terms.roff" || rows[1][5] != "TOOL_HOME" {
				t.Fatal("record output lost location/term", err)
			}
		}
	}
	var selected bytes.Buffer
	if err := result.Write(&selected, query.CSV, []string{"term", "line"}); err != nil || selected.String() != "term,line\nTOOL_HOME,1\n" {
		t.Fatal("selected fields changed", err)
	}
	for _, format := range []query.Format{query.JSON, query.JSONL, query.PlainText, query.RawSource, query.Grep, query.CSV, query.TSV} {
		if err := result.Write(shortOutput{}, format, nil); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write ignored for %s: %v", format, err)
		}
	}
	var rejected bytes.Buffer
	if err := result.Write(&rejected, query.CSV, []string{"made-up"}); err == nil || rejected.Len() != 0 {
		t.Fatal("invalid output partially written")
	}
}

type shortOutput struct{}

func (shortOutput) Write(p []byte) (int, error) { return len(p) / 2, nil }
