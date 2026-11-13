//go:build !js

package groff_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// BenchmarkGeneratorCheck measures the compiled generator's complete read-only
// CLI pipeline, including process startup, normalize/render/gofmt and disk reads.
// Binary compilation is outside timing. B/op and allocs/op describe the parent
// command scaffolding, NOT allocations in the generator child. In-process
// generator-argument grammar allocations are measured separately.
func BenchmarkGeneratorCheck(b *testing.B) {
	binary := filepath.Join(b.TempDir(), "gtroff-gen")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(b.Context(), "go", "build", "-o", binary, "./cmd/gen")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, e := build.CombinedOutput(); e != nil {
		b.Fatalf("build generator: %v\n%s", e, output)
	}
	generatorEnv := append(os.Environ(), "GOMAXPROCS="+strconv.Itoa(runtime.GOMAXPROCS(0)))
	check := func() ([]byte, error) {
		command := exec.CommandContext(b.Context(), binary, "-check")
		command.Env = generatorEnv
		return command.CombinedOutput()
	}
	// Check once before timing so setup/drift failures cannot masquerade as data.
	if output, e := check(); e != nil {
		b.Fatalf("generator fixture: %v\n%s", e, output)
	}
	b.ReportAllocs()
	for b.Loop() {
		if output, e := check(); e != nil {
			b.Fatalf("generator check: %v\n%s", e, output)
		}
	}
}
