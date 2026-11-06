package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/agentskill"
)

func TestSkillReadOnlyDiscoveryAndExportGrants(t *testing.T) {
	for _, args := range [][]string{{"skills", "list", "--format", "json"}, {"skills", "show", "--format", "json", agentskill.Name}, {"skills", "show", agentskill.Name}, {"help", "skills", "export"}} {
		var out, stderr bytes.Buffer
		calls := 0
		code := Run(args, Environment{Out: &out, Err: &stderr, ExportSkill: func(context.Context, string, string) (*agentskill.Receipt, error) {
			calls++
			return nil, errors.New("unexpected export")
		}})
		if code != 0 || calls != 0 || stderr.Len() != 0 || out.Len() == 0 {
			t.Fatal(args, code, calls, stderr.String())
		}
	}
	for _, args := range [][]string{{"skills", "export", agentskill.Name}, {"skills", "export", "--dir", "target", "../escape"}, {"skills", "export", "--dir", "target", "--format", "bad", agentskill.Name}, {"skills", "show", "missing"}, {"skills", "list", "extra"}} {
		var stderr bytes.Buffer
		calls := 0
		code := Run(args, Environment{Err: &stderr, ExportSkill: func(context.Context, string, string) (*agentskill.Receipt, error) { calls++; return nil, nil }})
		if code != ExitUsage || calls != 0 {
			t.Fatal(args, code, calls, stderr.String())
		}
	}
}
func TestSkillExportReceiptAndConflict(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		var out, stderr bytes.Buffer
		called := 0
		code := Run([]string{"skills", "export", "--dir", "chosen", "--format", "json", agentskill.Name}, Environment{Out: &out, Err: &stderr, ExportSkill: func(ctx context.Context, name, parent string) (*agentskill.Receipt, error) {
			called++
			if name != agentskill.Name || parent != "chosen" {
				t.Fatal(name, parent)
			}
			if conflict {
				return nil, agentskill.ErrConflict
			}
			return &agentskill.Receipt{SchemaVersion: 1, Info: agentskill.Info{Name: name, Version: agentskill.Version}, Directory: "chosen/" + name}, nil
		}})
		if called != 1 {
			t.Fatal(called)
		}
		if conflict {
			if code != ExitInvalid || out.Len() != 0 {
				t.Fatal(code, out.String())
			}
		} else {
			var result agentskill.Receipt
			if code != 0 || json.Unmarshal(out.Bytes(), &result) != nil || result.Info.Name != agentskill.Name {
				t.Fatal(code, out.String(), stderr.String())
			}
		}
	}
}
func TestSkillWriterAndCancelledExport(t *testing.T) {
	sentinel := errors.New("write failed")
	var stderr bytes.Buffer
	code := Run([]string{"skills", "show", agentskill.Name}, Environment{Out: extractFailWriter{sentinel}, Err: &stderr})
	if code != ExitIO || !strings.Contains(stderr.String(), sentinel.Error()) {
		t.Fatal(code, stderr.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	stderr.Reset()
	code = Run([]string{"skills", "export", "--dir", "chosen", agentskill.Name}, Environment{Context: ctx, Err: &stderr, ExportSkill: func(context.Context, string, string) (*agentskill.Receipt, error) { calls++; return nil, nil }})
	if code != ExitIO || calls != 0 {
		t.Fatal(code, calls, stderr.String())
	}
}
