package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteManyPreflightAndStaging(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.1")
	first := filepath.Join(dir, "page-001.png")
	second := filepath.Join(dir, "page-002.png")
	for _, p := range []string{source, first, second} {
		if err := os.WriteFile(p, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, files := range [][]File{{{first, []byte("new")}, {source, []byte("bad")}}, {{first, []byte("new")}, {first, []byte("duplicate")}}, {{first, []byte("new")}, {filepath.Join(dir, "missing", "page.png"), []byte("bad")}}} {
		if err := WriteMany(source, files); err == nil {
			t.Fatal("bad batch accepted")
		}
		data, err := os.ReadFile(first)
		if err != nil || string(data) != "original" {
			t.Fatal("preflight/staging failure damaged earlier destination")
		}
		temps, _ := filepath.Glob(filepath.Join(dir, ".gtroff-*"))
		if len(temps) > 0 {
			t.Fatal("temporary files leaked")
		}
	}
	if err := WriteMany(source, []File{{first, []byte("one")}, {second, []byte("two")}}); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{first: "one", second: "two", source: "original"} {
		data, err := os.ReadFile(p)
		if err != nil || string(data) != want {
			t.Fatalf("%s: %s %v", p, data, err)
		}
	}
}
func TestWriteManyAliases(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	alias := filepath.Join(dir, "alias.png")
	if err := os.WriteFile(source, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(source, alias); err != nil {
		t.Skip("hard links unavailable", err)
	}
	if err := WriteMany(source, []File{{alias, []byte("bad")}}); err == nil {
		t.Fatal("source alias accepted")
	}
	other := filepath.Join(dir, "other.png")
	if err := os.Link(source, other); err != nil {
		t.Fatal(err)
	}
	if err := WriteMany("", []File{{alias, []byte("one")}, {other, []byte("two")}}); err == nil {
		t.Fatal("output aliases accepted")
	}
}
