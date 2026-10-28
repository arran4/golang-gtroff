// Package atomicfile implements the CLI's explicit output-file boundary.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write replaces a regular destination through a same-directory temporary file.
// It never deletes the destination as a rename fallback. Source aliases and
// symlink/nonregular destinations are rejected. New files have mode 0600;
// existing regular-file permissions are retained. An empty source denotes stdin.
// Rename atomicity is provided
// by the host filesystem; unsupported replacements return an error unchanged.
func Write(destination, source string, data []byte) error {
	destAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	var inputInfo os.FileInfo
	if source != "" {
		inputAbs, err := filepath.Abs(source)
		if err != nil {
			return err
		}
		if inputAbs == destAbs {
			return fmt.Errorf("output aliases input source")
		}
		inputInfo, err = os.Stat(source)
		if err != nil {
			return fmt.Errorf("stat input: %w", err)
		}
	}
	mode := os.FileMode(0600)
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output must be a regular file, not a symlink or special file")
		}
		if inputInfo != nil && os.SameFile(inputInfo, info) {
			return fmt.Errorf("output aliases input source")
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(destAbs), ".gtroff-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if n, err := file.Write(data); err != nil {
		file.Close()
		return err
	} else if n != len(data) {
		file.Close()
		return fmt.Errorf("short output file write")
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, destAbs); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	return nil
}
