package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed .github/workflows/ci.yaml testdata/release/*.txtar
var releaseCases embed.FS

// These cases execute the actual policy and gate blocks in the owning workflow,
// rather than reimplementing their decisions in the test language.
func TestReleasePolicyScenarios(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable; policy is verified on the Linux owner job")
	}
	workflow, err := releaseCases.ReadFile(".github/workflows/ci.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	if err := fs.WalkDir(releaseCases, "testdata/release", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".txtar") {
			paths = append(paths, p)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no release policy scenarios")
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/release/"), func(t *testing.T) {
			raw, err := releaseCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for _, f := range txtar.Parse(raw).Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate fixture member")
				}
				files[f.Name] = f.Data
			}
			var tc struct {
				Block   string
				Env     map[string]string
				Exit    int
				Outputs map[string]string
				Error   string
			}
			if err := json.Unmarshal(files["scenario.json"], &tc); err != nil {
				t.Fatal(err)
			}
			if tc.Block != "policy" && tc.Block != "gate" {
				t.Fatal("unknown workflow block")
			}
			begin := "# release-" + tc.Block + "-begin"
			end := "# release-" + tc.Block + "-end"
			_, block, ok := strings.Cut(string(workflow), begin)
			if !ok {
				t.Fatal("missing policy block")
			}
			block, _, ok = strings.Cut(block, end)
			if !ok {
				t.Fatal("unterminated policy block")
			}
			lines := strings.Split(block, "\n")
			for i := range lines {
				lines[i] = strings.TrimPrefix(lines[i], "          ")
			}
			block = strings.Join(lines, "\n")
			output := filepath.Join(t.TempDir(), "outputs")
			cmd := exec.Command(bash, "-c", block)
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output)
			for _, name := range []string{"EVENT_NAME", "INPUT_MODE", "GITHUB_REF", "VERIFY_RESULT", "CANDIDATES_RESULT", "NATIVE_RESULT", "FULL"} {
				cmd.Env = append(cmd.Env, name+"="+tc.Env[name])
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()
			code := 0
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok {
					code = e.ExitCode()
				} else {
					t.Fatal(err)
				}
			}
			if code != tc.Exit || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.Error) {
				t.Fatalf("exit %d want %d; stdout=%s stderr=%s", code, tc.Exit, stdout.String(), stderr.String())
			}
			data, err := os.ReadFile(output)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			got := map[string]string{}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				if line == "" {
					continue
				}
				k, v, ok := strings.Cut(line, "=")
				if !ok {
					t.Fatal("malformed policy output")
				}
				got[k] = v
			}
			if len(got) != len(tc.Outputs) {
				t.Fatalf("unexpected outputs %v", got)
			}
			for key, value := range tc.Outputs {
				if got[key] != value {
					t.Fatalf("%s=%s want %s", key, got[key], value)
				}
			}
		})
	}
}

func TestNativeVerificationStopsAfterFailedTests(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable; workflow scripts are verified on the Linux owner job")
	}
	workflow, err := releaseCases.ReadFile(".github/workflows/ci.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, block, ok := strings.Cut(string(workflow), "# native-verification-begin")
	if !ok {
		t.Fatal("missing native verification block")
	}
	block, _, ok = strings.Cut(block, "# native-verification-end")
	if !ok {
		t.Fatal("unterminated native verification block")
	}
	// Substitute only the matrix label, as the Actions runner does. Execute the
	// actual checked-in script with a Go stand-in that records both invocations.
	block = strings.ReplaceAll(block, "${{ matrix.target }}", "linux/amd64")
	const fakeGo = `
go() {
  printf '%s\n' "$1" >> "$CALLS_FILE"
  if [[ "$1" == test ]]; then return "$TEST_STATUS"; fi
  return 0
}
`
	for _, status := range []string{"0", "23"} {
		t.Run(status, func(t *testing.T) {
			calls := filepath.Join(t.TempDir(), "calls")
			cmd := exec.Command(bash, "-c", fakeGo+block)
			cmd.Env = append(os.Environ(), "CALLS_FILE="+calls, "TEST_STATUS="+status)
			out, err := cmd.CombinedOutput()
			want := "test\nrun\n"
			if status != "0" {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 23 {
					t.Fatalf("test failure exit lost: %v %s", err, out)
				}
				want = "test\n"
			} else if err != nil {
				t.Fatalf("successful native verification: %v %s", err, out)
			}
			got, err := os.ReadFile(calls)
			if err != nil || string(got) != want {
				t.Fatalf("native invocation order: %q, want %q; %v", got, want, err)
			}
		})
	}
}
