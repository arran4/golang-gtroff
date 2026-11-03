package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGeneratedPreflight(t *testing.T) {
	for _, kind := range []string{"directory", "missing-parent", "duplicate", "empty"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			first, second := filepath.Join(dir, "first.go"), filepath.Join(dir, "second.go")
			mustGeneratedWrite(t, first, "original")
			switch kind {
			case "directory":
				if err := os.Mkdir(second, 0700); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				second = filepath.Join(dir, "missing", "second.go")
			case "duplicate":
				second = first
			case "empty":
				second = ""
			}
			if err := WriteGenerated([]File{{first, []byte("new")}, {second, []byte("new")}}); err == nil {
				t.Fatal("invalid output batch accepted")
			}
			assertGeneratedBytes(t, first, "original")
			assertNoGenerationStaging(t, dir)
		})
	}
}

func TestGeneratedStagingFailure(t *testing.T) {
	dir := t.TempDir()
	files := []File{{filepath.Join(dir, "a"), []byte("new-a")}, {filepath.Join(dir, "b"), []byte("new-b")}, {filepath.Join(dir, "c"), []byte("new-c")}}
	for _, file := range files {
		mustGeneratedWrite(t, file.Path, "old")
	}
	failure := errors.New("staging unavailable")
	calls := 0
	err := replaceOutputs(files, outputOperations{mkdirTemp: func(parent, pattern string) (string, error) {
		calls++
		if calls == 3 {
			return "", failure
		}
		return os.MkdirTemp(parent, pattern)
	}})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	for _, file := range files {
		assertGeneratedBytes(t, file.Path, "old")
	}
	assertNoGenerationStaging(t, dir)
}

func TestGeneratedCommitFailureRollsBackExistingAndNew(t *testing.T) {
	dir := t.TempDir()
	first, second, third := filepath.Join(dir, "old"), filepath.Join(dir, "new"), filepath.Join(dir, "last")
	mustGeneratedWrite(t, first, "original first")
	mustGeneratedWrite(t, third, "original last")
	when := time.Unix(1600000000, 0)
	if err := os.Chtimes(first, when, when); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("third rename failed")
	commits := 0
	err = replaceOutputs([]File{{first, []byte("changed")}, {second, []byte("created")}, {third, []byte("changed last")}}, outputOperations{rename: func(source, dest string) error {
		if filepath.Base(source) == "new" {
			commits++
			if commits == 3 {
				return failure
			}
		}
		return os.Rename(source, dest)
	}})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	assertGeneratedBytes(t, first, "original first")
	assertGeneratedBytes(t, third, "original last")
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Fatalf("new output survived rollback: %v", err)
	}
	after, err := os.Stat(first)
	if err != nil || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("rollback lost metadata: before=%v after=%v err=%v", before, after, err)
	}
	assertNoGenerationStaging(t, dir)
}

func TestGeneratedRollbackFailureRetainsOriginal(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	mustGeneratedWrite(t, first, "original a")
	mustGeneratedWrite(t, second, "original b")
	commitErr, restoreErr := errors.New("commit failure"), errors.New("restore failure")
	err := replaceOutputs([]File{{first, []byte("new a")}, {second, []byte("new b")}}, outputOperations{rename: func(source, dest string) error {
		if filepath.Base(source) == "original" {
			return restoreErr
		}
		if dest == second {
			return commitErr
		}
		return os.Rename(source, dest)
	}})
	if !errors.Is(err, commitErr) || !errors.Is(err, restoreErr) {
		t.Fatalf("failure identity lost: %v", err)
	}
	staging, errGlob := filepath.Glob(filepath.Join(dir, ".gtroff-gen-*"))
	if errGlob != nil || len(staging) != 1 || !strings.Contains(err.Error(), staging[0]) {
		t.Fatalf("recovery location not retained/reported: %v %v %v", staging, errGlob, err)
	}
	assertGeneratedBytes(t, filepath.Join(staging[0], "original"), "original a")
	assertGeneratedBytes(t, first, "new a")
	assertGeneratedBytes(t, second, "original b")
}

func TestGeneratedDestinationChangeDuringStaging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output")
	mustGeneratedWrite(t, path, "initial")
	err := replaceOutputs([]File{{path, []byte("new")}}, outputOperations{mkdirTemp: func(parent, pattern string) (string, error) {
		if err := os.WriteFile(path, []byte("external change"), 0600); err != nil {
			return "", err
		}
		return os.MkdirTemp(parent, pattern)
	}})
	if err == nil || !strings.Contains(err.Error(), "changed during staging") {
		t.Fatal(err)
	}
	assertGeneratedBytes(t, path, "external change")
	assertNoGenerationStaging(t, dir)
}

func TestGeneratedSuccessAndOwnedTemporaryPaths(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
	mustGeneratedWrite(t, first, "old")
	// The old fixed .tmp convention must never overwrite this unrelated file.
	mustGeneratedWrite(t, first+".tmp", "user-owned temporary file")
	for i := 0; i < 2; i++ {
		if err := WriteGenerated([]File{{first, []byte("one")}, {second, []byte("two")}}); err != nil {
			t.Fatal(err)
		}
		assertGeneratedBytes(t, first, "one")
		assertGeneratedBytes(t, second, "two")
		assertGeneratedBytes(t, first+".tmp", "user-owned temporary file")
		assertNoGenerationStaging(t, dir)
	}
	info, err := os.Stat(second)
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0644 {
		t.Fatalf("new generated mode: %v %v", info, err)
	}
}

func TestGeneratedAliases(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "parent-alias"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			first, alias := filepath.Join(dir, "first"), filepath.Join(dir, "alias")
			mustGeneratedWrite(t, first, "original")
			var err error
			switch kind {
			case "hardlink":
				err = os.Link(first, alias)
			case "symlink":
				err = os.Symlink(first, alias)
			case "parent-alias":
				link := filepath.Join(dir, "parent")
				err = os.Symlink(dir, link)
				alias = filepath.Join(link, "first")
			}
			if err != nil {
				t.Skipf("OS cannot create %s: %v", kind, err)
			}
			if err := WriteGenerated([]File{{first, []byte("new")}, {alias, []byte("other")}}); err == nil {
				t.Fatal("alias accepted")
			}
			assertGeneratedBytes(t, first, "original")
			assertNoGenerationStaging(t, dir)
		})
	}
}

func TestGeneratedWindowsNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows filename interpretation")
	}
	for _, name := range []string{"data:stream", "data.", "data ", "NUL"} {
		if err := WriteGenerated([]File{{filepath.Join(t.TempDir(), name), []byte("x")}}); err == nil {
			t.Errorf("accepted ambiguous filename %q", name)
		}
	}
}

func mustGeneratedWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertGeneratedBytes(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("%s: got %q, %v; want %q", path, data, err, want)
	}
}

func assertNoGenerationStaging(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, ".gtroff-gen-*"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("staging leaked: %v %v", paths, err)
	}
}
