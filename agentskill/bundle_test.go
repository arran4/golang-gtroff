package agentskill

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestAndArtifactCopies(t *testing.T) {
	b, e := Lookup(Name)
	if e != nil {
		t.Fatal(e)
	}
	again, e := Lookup(Name)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := b.ReadFile("manifest.json")
	z, _ := again.ReadFile("manifest.json")
	if !bytes.Equal(a, z) {
		t.Fatal("manifest not reproducible")
	}
	var manifest Manifest
	if e = json.Unmarshal(a, &manifest); e != nil || manifest.SchemaVersion != 1 || manifest.Version != Version {
		t.Fatal(e, manifest)
	}
	for _, entry := range manifest.Files {
		data, e := b.ReadFile(entry.Path)
		if e != nil || len(data) != entry.Bytes || digest(data) != entry.SHA256 {
			t.Fatal(entry, e)
		}
	}
	files := b.Files()
	files[0].Data[0] ^= 255
	data, _ := b.ReadFile(files[0].Path)
	if bytes.Equal(data, files[0].Data) {
		t.Fatal("mutable file alias")
	}
	m := b.Manifest()
	m.Contracts.Capabilities[0] = "changed"
	m.Files[0].Path = "changed"
	if b.Manifest().Files[0].Path == "changed" || b.Manifest().Contracts.Capabilities[0] == "changed" {
		t.Fatal("mutable manifest alias")
	}
	license, e := os.ReadFile("../LICENSE")
	if e != nil {
		t.Fatal(e)
	}
	bundledLicense, e := b.ReadFile("LICENSE")
	if e != nil || !bytes.Equal(license, bundledLicense) {
		t.Fatal("license copy drift; go generate ./...", e)
	}
}
func TestInvalidPathsAndUnknownNames(t *testing.T) {
	b, e := Lookup(Name)
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"../SKILL.md", "/SKILL.md", "references/../../file", "a\\..\\SKILL.md", ".", ""} {
		if _, e = b.ReadFile(path); !errors.Is(e, ErrPath) {
			t.Fatal(path, e)
		}
	}
	for _, name := range []string{"../escape", Name + "/../escape", "/tmp/escape", "unknown"} {
		root := filepath.Join(t.TempDir(), "not-created")
		if _, e = Export(context.Background(), name, root); !errors.Is(e, ErrNotFound) {
			t.Fatal(name, e)
		}
		if _, e = os.Stat(root); !os.IsNotExist(e) {
			t.Fatal("unknown bundle created parent", e)
		}
	}
	if _, e = b.ReadFile("missing.md"); !errors.Is(e, fs.ErrNotExist) {
		t.Fatal(e)
	}
}
func TestExportProtectionAndDeterminism(t *testing.T) {
	firstRoot, secondRoot := t.TempDir(), t.TempDir()
	first, e := Export(context.Background(), Name, firstRoot)
	if e != nil {
		t.Fatal(e)
	}
	second, e := Export(context.Background(), Name, secondRoot)
	if e != nil {
		t.Fatal(e)
	}
	if first.Info.ManifestSHA256 != second.Info.ManifestSHA256 {
		t.Fatal("host path changed manifest")
	}
	for _, file := range first.Files {
		a, e := os.ReadFile(filepath.Join(first.Directory, filepath.FromSlash(file.Path)))
		if e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile(filepath.Join(second.Directory, filepath.FromSlash(file.Path)))
		if e != nil || !bytes.Equal(a, b) {
			t.Fatal("host-dependent artifact", file, e)
		}
	}
	custom := filepath.Join(first.Directory, "SKILL.md")
	if e = os.WriteFile(custom, []byte("custom local skill"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Export(context.Background(), Name, firstRoot); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	b, e := os.ReadFile(custom)
	if e != nil || string(b) != "custom local skill" {
		t.Fatal("existing skill changed", e)
	}
}
func TestExportSymlinkCancellationAndPublishFailure(t *testing.T) {
	root, target := t.TempDir(), t.TempDir()
	destination := filepath.Join(root, Name)
	if e := os.Symlink(target, destination); e == nil {
		if _, e = Export(context.Background(), Name, root); !errors.Is(e, ErrConflict) {
			t.Fatal(e)
		}
		entries, e := os.ReadDir(target)
		if e != nil || len(entries) != 0 {
			t.Fatal("symlink target changed", e)
		}
	}
	root = t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Export(ctx, Name, root); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	failure := errors.New("publish unavailable")
	if _, e := export(context.Background(), Name, root, func(string, string) error { return failure }); !errors.Is(e, failure) {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) != 0 {
		t.Fatal("failed export left partial bundle", entries, e)
	}
}

type artifactWriter struct {
	n   int
	err error
}

func (w artifactWriter) Write(p []byte) (int, error) { return w.n, w.err }

func TestWriteArtifactFailures(t *testing.T) {
	b, err := Lookup(Name)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("writer failed")
	for _, tc := range []struct {
		writer io.Writer
		want   error
	}{
		{artifactWriter{0, nil}, io.ErrShortWrite},
		{artifactWriter{0, sentinel}, sentinel},
	} {
		if err := b.WriteFile(context.Background(), tc.writer, "SKILL.md"); !errors.Is(err, tc.want) {
			t.Fatal(err)
		}
	}
	for _, writer := range []io.Writer{nil, (*bytes.Buffer)(nil), artifactWriter{-1, nil}, artifactWriter{1 << 30, nil}} {
		if err := b.WriteFile(nil, writer, "SKILL.md"); err == nil {
			t.Fatal("accepted invalid writer")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	if err := b.WriteFile(ctx, &out, "SKILL.md"); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
	if err := b.WriteFile(nil, &out, "SKILL.md"); err != nil {
		t.Fatal(err)
	}
	expected, _ := b.ReadFile("SKILL.md")
	if !bytes.Equal(out.Bytes(), expected) {
		t.Fatal("artifact bytes changed")
	}
}
