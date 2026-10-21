package releasecheck

import (
	"archive/zip"
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
	}{{"valid", []string{"gtroff", "docs/man.1"}, false}, {"parent traversal", []string{"../gtroff"}, true}, {"absolute", []string{"/gtroff"}, true}, {"duplicate", []string{"gtroff", "gtroff"}, true}} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "candidate.zip")
			f, err := os.Create(file)
			if err != nil {
				t.Fatal(err)
			}
			z := zip.NewWriter(f)
			for _, name := range tc.files {
				w, err := z.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = w.Write([]byte("visible")); err != nil {
					t.Fatal(err)
				}
			}
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
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
