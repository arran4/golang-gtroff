package releasecheck

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksumFailures(t *testing.T) {
	for _, tc := range []struct {
		name, index, payload string
		fail                 bool
	}{
		{"valid", "", "candidate", false},
		{"changed archive", "", "corrupted", true},
		{"unlisted artifact", "", "candidate", true},
		{"traversal", "%s  ../candidate.tar.gz\n", "candidate", true},
		{"duplicate", "%s  candidate.tar.gz\n%s  candidate.tar.gz\n", "candidate", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sum := fmt.Sprintf("%x", sha256.Sum256([]byte("candidate")))
			index := sum + "  candidate.tar.gz\n"
			if tc.index != "" {
				index = strings.ReplaceAll(tc.index, "%s", sum)
			}
			if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(index), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "candidate.tar.gz"), []byte(tc.payload), 0600); err != nil {
				t.Fatal(err)
			}
			want := map[string]artifact{"candidate.tar.gz": {}}
			if tc.name == "unlisted artifact" {
				want["missing.deb"] = artifact{}
			}
			err := checkHashes(dir, "checksums.txt", want)
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v want failure=%v", err, tc.fail)
			}
		})
	}
}
func TestArchiveNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		fail  bool
	}{
		{"valid", []string{"gtroff", "docs/man.1"}, false},
		{"valid directory", []string{"gtroff", "docs/"}, false},
		{"parent traversal", []string{"../gtroff"}, true},
		{"absolute", []string{"/gtroff"}, true},
		{"Windows traversal", []string{`..\gtroff.exe`}, true},
		{"Windows drive", []string{`C:gtroff.exe`}, true},
		{"Windows stream", []string{`docs/man.1:payload`}, true},
		{"unsafe directory", []string{"../"}, true},
		{"duplicate", []string{"gtroff", "gtroff"}, true},
	} {
		for _, format := range []string{"zip", "tar.gz"} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "candidate."+format)
				writeArchive(t, file, tc.files)
				got, err := archive(file)
				if (err != nil) != tc.fail {
					t.Fatalf("error=%v want failure=%v", err, tc.fail)
				}
				if !tc.fail && string(got["gtroff"]) != "visible" {
					t.Fatal("archive lost content")
				}
			})
		}
	}
}

func writeArchive(t *testing.T, filename string, names []string) {
	t.Helper()
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if strings.HasSuffix(filename, ".zip") {
		z := zip.NewWriter(f)
		for _, name := range names {
			w, err := z.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(name, "/") {
				if _, err := w.Write([]byte("visible")); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(f)
		tr := tar.NewWriter(gz)
		for _, name := range names {
			h := &tar.Header{Name: name, Mode: 0644, Size: int64(len("visible")), Typeflag: tar.TypeReg}
			if strings.HasSuffix(name, "/") {
				h.Typeflag = tar.TypeDir
				h.Size = 0
			}
			if err := tr.WriteHeader(h); err != nil {
				t.Fatal(err)
			}
			if h.Typeflag == tar.TypeReg {
				if _, err := tr.Write([]byte("visible")); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := tr.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestChecksumWindowsPathCannotEscape(t *testing.T) {
	// On Unix this is a literal filename. On Windows it resolves outside dist.
	// Reject it on every host before opening the file so validation is portable.
	root := t.TempDir()
	dir := filepath.Join(root, "dist")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	name := `..\candidate.tar.gz`
	payload := []byte("must not be accepted outside the artifact directory")
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0600); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(payload))
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(sum+"  "+name+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkHashes(dir, "checksums.txt", map[string]artifact{name: {}}); err == nil {
		t.Fatal("accepted a Windows traversal checksum")
	}
}
