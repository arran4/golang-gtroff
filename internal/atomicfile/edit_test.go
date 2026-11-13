package atomicfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGuardedEditsAndRetry(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	mustGeneratedWrite(t, a, "old-a")
	mustGeneratedWrite(t, b, "old-b")
	guards := []SourceGuard{{a, hashBytes([]byte("old-a")), 5}, {b, hashBytes([]byte("old-b")), 5}}
	edits := []SourceEdit{{a, guards[0].Digest, []byte("new-a")}, {b, guards[1].Digest, []byte("new-b")}}
	if err := WriteEdits(context.Background(), edits, guards); err != nil {
		t.Fatal(err)
	}
	assertGeneratedBytes(t, a, "new-a")
	assertGeneratedBytes(t, b, "new-b")
	if err := WriteEdits(context.Background(), edits, guards); err != nil {
		t.Fatal("idempotent retry", err)
	}
	mustGeneratedWrite(t, b, "unrelated")
	if err := WriteEdits(context.Background(), edits, guards); err == nil {
		t.Fatal("stale disk accepted")
	}
	assertGeneratedBytes(t, a, "new-a")
	assertGeneratedBytes(t, b, "unrelated")
}
func TestGuardedEditRollbackAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		aliased, restoreFailure bool
	}{
		{"restored", false, false}, {"recovery", false, true},
		{"restored-parent-alias", true, false}, {"recovery-parent-alias", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restoreFailure := tc.restoreFailure
			dir := t.TempDir()
			if tc.aliased {
				alias := filepath.Join(t.TempDir(), "parent-alias")
				if err := os.Symlink(dir, alias); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("directory symlinks unavailable: %v", err)
					}
					t.Fatal(err)
				}
				dir = alias
			}
			canonicalDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				t.Fatal(err)
			}
			a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
			mustGeneratedWrite(t, a, "old-a")
			mustGeneratedWrite(t, b, "old-b")
			guards := []SourceGuard{{a, hashBytes([]byte("old-a")), 5}, {b, hashBytes([]byte("old-b")), 5}}
			edits := []SourceEdit{{a, guards[0].Digest, []byte("new-a")}, {b, guards[1].Digest, []byte("new-b")}}
			failure := errors.New("injected rename failure")
			calls := 0
			err = writeEdits(context.Background(), edits, guards, outputOperations{rename: func(from, to string) error {
				calls++
				if calls == 2 || restoreFailure && calls == 3 {
					return failure
				}
				return os.Rename(from, to)
			}})
			if !errors.Is(err, failure) {
				t.Fatal(err)
			}
			assertGeneratedBytes(t, b, "old-b")
			if restoreFailure {
				assertGeneratedBytes(t, a, "new-a")
				dirs, e := filepath.Glob(filepath.Join(canonicalDir, ".gtroff-gen-*"))
				if e != nil || len(dirs) != 1 {
					t.Fatalf("recovery dirs %v %v", dirs, e)
				}
				backup := filepath.Join(dirs[0], "original")
				assertGeneratedBytes(t, backup, "old-a")
				if !strings.Contains(err.Error(), dirs[0]) {
					t.Fatal("missing recovery path", err)
				}
				// Retrying the same captured batch recognizes the exact already-written a.
				if e := WriteEdits(context.Background(), edits, guards); e != nil {
					t.Fatal("recover by retry", e)
				}
				assertGeneratedBytes(t, b, "new-b")
			} else {
				assertGeneratedBytes(t, a, "old-a")
				assertNoGenerationStaging(t, dir)
			}
		})
	}
}
func TestGuardedSourcesAndCopyFailures(t *testing.T) {
	dir := t.TempDir()
	a, b, out := filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "out")
	mustGeneratedWrite(t, a, "old")
	mustGeneratedWrite(t, b, "guarded")
	mustGeneratedWrite(t, out, "untouched")
	guards := []SourceGuard{{a, hashBytes([]byte("old")), 3}, {b, hashBytes([]byte("guarded")), 7}}
	mustGeneratedWrite(t, b, "changed")
	if e := WriteEditedOutput(context.Background(), out, guards, []byte("new")); e == nil {
		t.Fatal("unguarded copy")
	}
	assertGeneratedBytes(t, out, "untouched")
	if e := WriteEdits(context.Background(), []SourceEdit{{a, guards[0].Digest, []byte("new")}}, guards); e == nil {
		t.Fatal("unchanged dependency not guarded")
	}
	assertGeneratedBytes(t, a, "old")
	mustGeneratedWrite(t, b, "guarded")
	if e := WriteEditedOutput(context.Background(), a, guards, []byte("new")); e == nil {
		t.Fatal("copy overwrote original")
	}
	if e := WriteEditedOutput(context.Background(), out, guards, []byte("new")); e != nil {
		t.Fatal(e)
	}
	assertGeneratedBytes(t, out, "new")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := WriteEdits(ctx, []SourceEdit{{a, guards[0].Digest, []byte("new")}}, guards); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	assertGeneratedBytes(t, a, "old")
}
func TestGuardedAliasDenial(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
			mustGeneratedWrite(t, a, "old")
			var e error
			if kind == "symlink" {
				e = os.Symlink(a, b)
			} else {
				e = os.Link(a, b)
			}
			if e != nil {
				t.Skip(e)
			}
			guards := []SourceGuard{{a, hashBytes([]byte("old")), 3}, {b, hashBytes([]byte("old")), 3}}
			if e := WriteEdits(context.Background(), []SourceEdit{{a, guards[0].Digest, []byte("new")}}, guards); e == nil {
				t.Fatal("alias accepted")
			}
			assertGeneratedBytes(t, a, "old")
		})
	}
}
func TestGuardedStagingRaceAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed", true: "cancelled"}[cancelled], func(t *testing.T) {
			dir := t.TempDir()
			a := filepath.Join(dir, "a")
			mustGeneratedWrite(t, a, "old")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			guard := SourceGuard{a, hashBytes([]byte("old")), 3}
			err := writeEdits(ctx, []SourceEdit{{a, guard.Digest, []byte("new")}}, []SourceGuard{guard}, outputOperations{mkdirTemp: func(parent, pattern string) (string, error) {
				if cancelled {
					cancel()
				} else {
					mustGeneratedWrite(t, a, "external")
				}
				return os.MkdirTemp(parent, pattern)
			}})
			if err == nil {
				t.Fatal("staging change accepted")
			}
			if cancelled {
				assertGeneratedBytes(t, a, "old")
			} else {
				assertGeneratedBytes(t, a, "external")
			}
			assertNoGenerationStaging(t, dir)
		})
	}
}

func TestGuardedCopyModeAndCancelledCopy(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	out := filepath.Join(dir, "copy")
	mustGeneratedWrite(t, source, "old")
	guards := []SourceGuard{{source, hashBytes([]byte("old")), 3}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := WriteEditedOutput(ctx, out, guards, []byte("new")); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("cancelled copy created output", e)
	}
	if e := WriteEditedOutput(context.Background(), out, guards, []byte("new")); e != nil {
		t.Fatal(e)
	}
	info, e := os.Stat(out)
	if e != nil {
		t.Fatal(e)
	}
	if runtime.GOOS == "windows" {
		// Go reports Windows' readonly attribute, not its access-control list.
		if info.Mode().Perm()&0200 == 0 {
			t.Fatal("new copy is unexpectedly readonly")
		}
	} else if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("new copy grants group/other access: %v", info.Mode())
	}
	assertGeneratedBytes(t, source, "old")
	if e := os.Chmod(out, 0640); e != nil {
		t.Fatal(e)
	}
	before, e := os.Stat(out)
	if e != nil {
		t.Fatal(e)
	}
	if runtime.GOOS != "windows" && before.Mode().Perm() != 0640 {
		t.Fatal("requested POSIX mode not set", before.Mode())
	}
	if e := WriteEditedOutput(context.Background(), out, guards, []byte("again")); e != nil {
		t.Fatal(e)
	}
	info, e = os.Stat(out)
	if e != nil {
		t.Fatal(e)
	}
	if info.Mode().Perm() != before.Mode().Perm() {
		t.Fatal("existing reported mode changed", before.Mode(), info.Mode())
	}
}
