package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
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
