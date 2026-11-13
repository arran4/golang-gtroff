package analyze_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/analyze"
)

func TestAnalysisOutputContracts(t *testing.T) {
	report, err := analyze.Run(context.Background(), readSet(t, ".TH TOOL 1\n.SH NAME\ntext\n"), analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []analyze.Format{analyze.Text, analyze.JSON, analyze.JSONL} {
		var out bytes.Buffer
		if err := report.Write(&out, format, false); err != nil {
			t.Fatal(err)
		}
		switch format {
		case analyze.Text:
			if !strings.Contains(out.String(), "profile: man") || !strings.Contains(out.String(), "rendering not assessed") || strings.Contains(out.String(), "input.1:1:1: control") {
				t.Fatal("text summary contract changed")
			}
		case analyze.JSON:
			var decoded analyze.Report
			if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || decoded.SchemaVersion != 1 || len(decoded.Findings) != len(report.Findings) {
				t.Fatal("JSON report lost capability/evidence data", err)
			}
		case analyze.JSONL:
			decoder := json.NewDecoder(&out)
			records := 0
			for {
				var row struct {
					SchemaVersion int
					Type          string
				}
				err := decoder.Decode(&row)
				if err == io.EOF {
					break
				}
				if err != nil || row.SchemaVersion != 1 {
					t.Fatal("invalid JSONL record", err)
				}
				if records == 0 && row.Type != "summary" || records > 0 && row.Type != "finding" {
					t.Fatal("unstable JSONL framing")
				}
				records++
			}
			if records != len(report.Findings)+1 {
				t.Fatal("JSONL findings dropped")
			}
		}
		if err := report.Write(shortOutput{}, format, false); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal("short output accepted", format, err)
		}
	}
	var detailed bytes.Buffer
	if err := report.Write(&detailed, analyze.Text, true); err != nil || !strings.Contains(detailed.String(), "input.1:1:1: control .TH") {
		t.Fatal("detailed source evidence absent", err)
	}
	var rejected bytes.Buffer
	if err := report.Write(&rejected, analyze.JSON, true); err == nil || rejected.Len() != 0 {
		t.Fatal("invalid output options wrote bytes")
	}
}

type shortOutput struct{}

func (shortOutput) Write(p []byte) (int, error) { return len(p) / 2, nil }
