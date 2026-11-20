package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

const withheld = "withheld test marker"

func TestCandidateFieldsOnly(t *testing.T) {
	root := t.TempDir()
	for _, scanner := range []string{"gitleaks", "trufflehog"} {
		t.Run(scanner, func(t *testing.T) {
			var raw []byte
			if scanner == "gitleaks" {
				raw, _ = json.Marshal([]any{map[string]any{"RuleID": "generic-api-key", "File": filepath.Join(root, "docs", "guide.md"), "StartLine": 35, "Commit": strings.Repeat("a", 40), "Secret": withheld, "Match": withheld, "Email": withheld, "Message": withheld, "FutureField": withheld}})
			} else {
				raw, _ = json.Marshal(map[string]any{"DetectorName": "PrivateKey", "Raw": withheld, "RawV2": withheld, "Redacted": withheld, "ExtraData": withheld, "VerificationError": withheld, "FutureField": withheld, "SourceMetadata": map[string]any{"Data": map[string]any{"Git": map[string]any{"file": filepath.Join(root, "docs", "guide.md"), "line": 35, "commit": strings.Repeat("a", 40), "repository": withheld}}}})
			}
			rows, err := redact(scanner, root, raw)
			if err != nil || len(rows) != 1 {
				t.Fatalf("candidate parse: rows=%+v error=%v", rows, err)
			}
			b, _ := json.Marshal(rows)
			if bytes.Contains(b, []byte(withheld)) || bytes.Contains(b, []byte(root)) {
				t.Fatal("raw value or host path leaked")
			}
			if rows[0].File != "docs/guide.md" || rows[0].Line != 35 || rows[0].Commit != strings.Repeat("a", 40) {
				t.Fatalf("source metadata lost: %+v", rows)
			}
		})
	}
	filesystem := []byte(`{"DetectorName":"PrivateKey","SourceMetadata":{"Data":{"Filesystem":{"file":"example.txt","line":4}}}}`)
	rows, err := redact("trufflehog", root, append(append(filesystem, '\n'), filesystem...))
	if err != nil || len(rows) != 2 || rows[0].File != "example.txt" || rows[0].Line != 4 {
		t.Fatalf("JSONL/filesystem metadata wrong: %+v %v", rows, err)
	}
	for _, input := range []string{"not JSON " + withheld, `[{"RuleID":"bad rule","File":"file"}]`, `null`} {
		if _, err := redact("gitleaks", root, []byte(input)); err == nil || strings.Contains(err.Error(), withheld) {
			t.Fatal("invalid scanner output accepted or echoed")
		}
	}
}

func TestRunBoundaries(t *testing.T) {
	for _, scanner := range []string{"gitleaks", "trufflehog"} {
		for _, kind := range []string{"git", "filesystem"} {
			t.Run(scanner+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				var command []string
				invoke := func(_ context.Context, _ string, args []string) (result, error) {
					if len(args) == 1 {
						if scanner == "gitleaks" {
							return result{stdout: []byte("8.30.1\n")}, nil
						}
						return result{stdout: []byte("trufflehog 3.99.9\n")}, nil
					}
					command = append([]string{}, args...)
					if scanner == "gitleaks" {
						return result{stdout: []byte(`[{"RuleID":"generic-api-key","File":"docs/guide.md","StartLine":35,"Secret":"` + withheld + `"}]`), code: 10}, nil
					}
					return result{stderr: []byte(`{"level":"info-0","msg":"` + withheld + `"}`)}, nil
				}
				var out bytes.Buffer
				if err := run(context.Background(), []string{"-scanner", scanner, "-executable", "explicit-tool", "-kind", kind, "-source", root, "-label", "test-scope"}, &out, invoke); err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(out.Bytes(), []byte(withheld)) || bytes.Contains(out.Bytes(), []byte(root)) {
					t.Fatal("subprocess data/host path leaked")
				}
				var r receipt
				if json.Unmarshal(out.Bytes(), &r) != nil || !r.Complete || r.Label != "test-scope" {
					t.Fatal("missing completed receipt")
				}
				joined := strings.Join(command, " ")
				if scanner == "gitleaks" {
					if !strings.Contains(joined, "--ignore-gitleaks-allow") || !strings.Contains(joined, "--config") || !strings.Contains(joined, "--redact=100") {
						t.Fatal("audit lost suppression override or redaction")
					}
					if kind == "git" && (!strings.Contains(joined, "--all --full-history") || !strings.Contains(joined, "--diff-merges=separate")) {
						t.Fatal("history/merge coverage flags lost")
					}
				} else {
					if !strings.Contains(joined, "--no-verification") || !strings.Contains(joined, "--no-update") || !strings.Contains(joined, "--fail-on-scan-errors") || !strings.Contains(joined, "--no-ignore-tag") {
						t.Fatal("audit enabled verification/update or hid diagnostics/ignored findings")
					}
					if kind == "git" && (!strings.Contains(joined, "--bare") || !strings.Contains(joined, "--trust-local-git-config")) {
						t.Fatal("controlled mirror scan would lose additional refs")
					}
				}
			})
		}
	}
}

func TestIncompleteScansStayIncomplete(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, scanner string
		output        result
		err           error
	}{
		{"bad-json", "gitleaks", result{stdout: []byte(withheld)}, nil},
		{"process-failure", "gitleaks", result{stdout: []byte("[]"), code: 2}, nil},
		{"missing-process", "gitleaks", result{}, errors.New(withheld)},
		{"error-diagnostic", "trufflehog", result{stderr: []byte(`{"level":"error","msg":"` + withheld + `"}`)}, nil},
		{"warning-diagnostic", "trufflehog", result{stderr: []byte(`{"level":"warn","msg":"` + withheld + `"}`)}, nil},
		{"unknown-diagnostic", "trufflehog", result{stderr: []byte(withheld)}, nil},
		{"unknown-level", "trufflehog", result{stderr: []byte(`{"level":"` + withheld + `"}`)}, nil},
		{"inconsistent-success", "gitleaks", result{stdout: []byte(`[{"RuleID":"generic-api-key","File":"a"}]`)}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invoke := func(_ context.Context, _ string, args []string) (result, error) {
				if len(args) == 1 {
					v := "8.30.1"
					if tc.scanner == "trufflehog" {
						v = "trufflehog 3.99.9"
					}
					return result{stdout: []byte(v)}, nil
				}
				return tc.output, tc.err
			}
			var out bytes.Buffer
			err := run(context.Background(), []string{"-scanner", tc.scanner, "-executable", "explicit-tool", "-kind", "filesystem", "-source", root, "-label", "test"}, &out, invoke)
			var r receipt
			if !errors.Is(err, errIncomplete) || json.Unmarshal(out.Bytes(), &r) != nil || r.Complete {
				t.Fatal("failed scan presented as complete")
			}
			if bytes.Contains(out.Bytes(), []byte(withheld)) || strings.Contains(err.Error(), withheld) {
				t.Fatal("scanner diagnostic/value leaked")
			}
		})
	}
	var out bytes.Buffer
	err := run(context.Background(), []string{"-scanner", "gitleaks", "-executable", "explicit-tool", "-kind", "filesystem", "-source", root, "-label", "test"}, &out, func(context.Context, string, []string) (result, error) {
		return result{stdout: []byte(withheld)}, nil
	})
	if err == nil || strings.Contains(err.Error(), withheld) || out.Len() != 0 {
		t.Fatal("unrecognised version accepted or echoed")
	}
}

func TestCaptureLimitCannotBeBypassedByIOCopy(t *testing.T) {
	cancelled := false
	b := &cappedBuffer{remaining: 4, cancel: func() { cancelled = true }}
	if _, err := io.Copy(b, strings.NewReader("more than four bytes")); err == nil || !cancelled || b.data.Len() != 0 {
		t.Fatal("output cap bypassed or cancellation lost")
	}
}

func TestPathMetadata(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", ".", "../outside", "bad\npath"} {
		if got := relativeFile(root, name); got != "[unreportable path]" {
			t.Errorf("unsafe path %q became %q", name, got)
		}
	}
	if got := relativeFile(root, filepath.Join(root, "..", "outside")); got != "[unreportable path]" {
		t.Errorf("outside source path became %q", got)
	}
}
