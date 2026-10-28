package manlib_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/manlib"
)

func fixtureIndex(t *testing.T) *manlib.Index {
	t.Helper()
	sys := fstest.MapFS{
		"man1/tool.1":   {Data: []byte(".TH TOOL 1\n.SH NAME\ntool \\- useful utility\n.SH OPTIONS\nUnique content here.\n")},
		"man3/tool.3":   {Data: []byte(".TH TOOL 3\n.SH NAME\ntool \\- library function\n")},
		"man3/ssl.3ssl": {Data: []byte(".TH SSL 3ssl\n.SH NAME\nssl \\- TLS helper\n")},
	}
	idx, e := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	return idx
}
func TestIndexSearchAndSnapshots(t *testing.T) {
	idx := fixtureIndex(t)
	entries := idx.Entries()
	if len(entries) != 3 || entries[0].Name != "ssl" || entries[1].Section != "1" || entries[2].Section != "3" {
		t.Fatal(entries)
	}
	for query, want := range map[string]int{"tool": 2, "useful": 1, "unique content": 1, "TLS": 1, "missing": 0} {
		hits, e := idx.Search(context.Background(), query, 100)
		if e != nil || len(hits) != want {
			t.Fatalf("query %q %+v %v", query, hits, e)
		}
	}
	entries[0].Name = "mutated"
	if idx.Entries()[0].Name != "ssl" {
		t.Fatal("entries shared")
	}
	doc, e := idx.Document(context.Background(), "tool", "1", "")
	if e != nil {
		t.Fatal(e)
	}
	source := doc.Source()
	doc.Nodes[0].Children = nil
	again, e := idx.Document(context.Background(), "tool", "1", "")
	if e != nil || again.Source() != source || !strings.Contains(again.Text(1, man.TextOptions{}), "Unique") {
		t.Fatal("caller model mutation changed index")
	}
}
func TestGzipAndLimits(t *testing.T) {
	var compressed bytes.Buffer
	z := gzip.NewWriter(&compressed)
	_, _ = z.Write([]byte(".TH ZIP 1\n.SH NAME\nzip \\- compressed utility\n"))
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	sys := fstest.MapFS{"man1/zip.1.gz": {Data: compressed.Bytes()}, "man1/bad.1.gz": {Data: []byte("not gzip")}}
	idx, e := manlib.Build(context.Background(), []manlib.Root{{Label: "root", FS: sys}}, manlib.Limits{})
	if e != nil || len(idx.Entries()) != 1 || len(idx.Notices()) != 1 {
		t.Fatal(idx, e)
	}
	for _, l := range []manlib.Limits{{MaxFileBytes: 10}, {MaxTotalBytes: 10}, {MaxScanEntries: 1}, {MaxEntries: 1, MaxFileBytes: 1}, {MaxNodes: 4}} {
		if _, e := manlib.Build(context.Background(), []manlib.Root{{Label: "root", FS: sys}}, l); !errors.Is(e, manlib.ErrLimit) {
			t.Fatalf("budget %+v: %v", l, e)
		}
	}
	var bomb bytes.Buffer
	z = gzip.NewWriter(&bomb)
	_, _ = z.Write(bytes.Repeat([]byte("x"), 4096))
	_ = z.Close()
	if _, e := manlib.Build(context.Background(), []manlib.Root{{Label: "root", FS: fstest.MapFS{"bomb.1.gz": {Data: bomb.Bytes()}}}}, manlib.Limits{MaxFileBytes: 1024}); !errors.Is(e, manlib.ErrLimit) {
		t.Fatal("gzip expansion escaped budget", e)
	}
}
func TestDirectorySymlinksAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if e := os.Mkdir(filepath.Join(dir, "man1"), 0700); e != nil {
		t.Fatal(e)
	}
	source := []byte(".TH SAFE 1\n.SH NAME\nsafe \\- original snapshot\n")
	if e := os.WriteFile(filepath.Join(dir, "man1", "safe.1"), source, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(outside, "secret.1"), []byte("SECRET OUTSIDE ROOT"), 0600); e != nil {
		t.Fatal(e)
	}
	linked := os.Symlink(filepath.Join(outside, "secret.1"), filepath.Join(dir, "man1", "escape.1")) == nil
	d, e := manlib.OpenDirectory(dir)
	if e != nil {
		t.Fatal(e)
	}
	// Keep the root closed even when an assertion fails before the explicit
	// close below. Windows otherwise retains a handle during TempDir cleanup.
	t.Cleanup(func() { _ = d.Close() })
	root := d.Root("safe")
	idx, e := manlib.Build(context.Background(), []manlib.Root{root}, manlib.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	if len(idx.Entries()) != 1 {
		t.Fatal("symlink page escaped", idx.Entries())
	}
	if linked && len(idx.Notices()) == 0 {
		t.Fatal("symlink omission invisible")
	}
	if linked {
		if _, e := root.FS.Open("man1/escape.1"); e == nil {
			t.Fatal("OS root allowed symlink escape")
		}
	}
	if e = d.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(dir, "man1", "safe.1")); e != nil {
		t.Fatal(e)
	}
	doc, e := idx.Document(context.Background(), "safe", "1", "")
	if e != nil || doc.Source() != string(source) {
		t.Fatal("request reopened source after snapshot", e)
	}
}
func TestBuildCancellationAndDuplicatePrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := manlib.Build(ctx, []manlib.Root{{Label: "root", FS: fstest.MapFS{}}}, manlib.Limits{}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	roots := []manlib.Root{{Label: "first", FS: fstest.MapFS{"x.1": {Data: []byte("first root\n")}}}, {Label: "second", FS: fstest.MapFS{"x.1": {Data: []byte("second root\n")}}}}
	idx, e := manlib.Build(context.Background(), roots, manlib.Limits{})
	if e != nil {
		t.Fatal(e)
	}
	for root, want := range map[string]string{"": "first root", "second": "second root"} {
		d, e := idx.Document(context.Background(), "x", "1", root)
		if e != nil || !strings.Contains(d.Source(), want) {
			t.Fatal(root, e)
		}
	}
	if _, e := idx.Document(context.Background(), "../x", "1", ""); !errors.Is(e, fs.ErrNotExist) {
		t.Fatal("unknown lookup not confined", e)
	}
}
