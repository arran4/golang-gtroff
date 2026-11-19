package atomicfile

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAtomicOutputAndSourceProtection(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input.1")
	dest := filepath.Join(dir, "output.html")
	original := []byte(".TH DEMO 1\n")
	if err := os.WriteFile(source, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Write(dest, source, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := Write(dest, source, []byte("second")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "second" {
		t.Fatal("replacement failed")
	}
	if err := Write(source, source, []byte("must not overwrite")); err == nil {
		t.Fatal("input source was replaceable")
	}
	raw, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(raw, original) {
		t.Fatal("source changed")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dest)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("new output mode")
		}
	}
	// Alias checks use file identity, not only spelling.
	hardlink := filepath.Join(dir, "hardlink.1")
	if err := os.Link(source, hardlink); err == nil {
		if err := Write(hardlink, source, []byte("bad")); err == nil {
			t.Fatal("source hardlink accepted")
		}
	}
	symlink := filepath.Join(dir, "symlink.html")
	if err := os.Symlink(dest, symlink); err == nil {
		if err := Write(symlink, source, []byte("bad")); err == nil {
			t.Fatal("symlink output accepted")
		}
	}
	if err := Write(dir, source, []byte("bad")); err == nil {
		t.Fatal("directory output accepted")
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range files {
		if len(entry.Name()) >= 8 && entry.Name()[:8] == ".gtroff-" {
			t.Fatal("temporary file leaked")
		}
	}
}

func TestAtomicOutputProtectsIncludedSources(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root.1")
	child := filepath.Join(dir, "child.roff")
	original := []byte("Included source.\n")
	if err := os.WriteFile(root, []byte(".so child.roff\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, original, 0600); err != nil {
		t.Fatal(err)
	}
	inputs := []string{root, child}
	if err := WriteSources(child, inputs, []byte("bad")); err == nil {
		t.Fatal("included source replaceable")
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Link(child, alias); err == nil {
		if err := WriteSources(alias, inputs, []byte("bad")); err == nil {
			t.Fatal("included hardlink replaceable")
		}
	}
	data, err := os.ReadFile(child)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatal("source changed")
	}
	dest := filepath.Join(dir, "out")
	if err := WriteSources(dest, inputs, []byte("export")); err != nil {
		t.Fatal(err)
	}
	if err := WriteSources(dest, []string{filepath.Join(dir, "missing")}, []byte("bad")); err == nil {
		t.Fatal("missing guard source ignored")
	}
	data, err = os.ReadFile(dest)
	if err != nil || string(data) != "export" {
		t.Fatal("failed preflight changed destination")
	}
}
