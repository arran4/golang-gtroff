package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// File is an explicitly named encoded output, never a pathname derived from
// document content.
type File struct {
	Path string
	Data []byte
}

// WriteMany preflights every destination and stages/syncs all files before any
// replacement. Each rename is atomic; the collection is not a filesystem-wide
// transaction. A rename failure can leave earlier pages replaced, and is reported.
func WriteMany(source string, files []File) error {
	if len(files) == 0 {
		return fmt.Errorf("no output files")
	}
	var inputInfo os.FileInfo
	var inputAbs string
	var err error
	if source != "" {
		inputAbs, err = filepath.Abs(source)
		if err != nil {
			return err
		}
		inputInfo, err = os.Stat(source)
		if err != nil {
			return fmt.Errorf("stat input: %w", err)
		}
	}
	type staged struct {
		dest, temp string
		mode       os.FileMode
		info       os.FileInfo
	}
	outputs := make([]staged, len(files))
	paths := map[string]bool{}
	for i, f := range files {
		dest, err := filepath.Abs(f.Path)
		if err != nil {
			return err
		}
		if paths[dest] {
			return fmt.Errorf("duplicate output path")
		}
		paths[dest] = true
		if source != "" && dest == inputAbs {
			return fmt.Errorf("output aliases input source")
		}
		entry := staged{dest: dest, mode: 0600}
		info, err := os.Lstat(dest)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("output must be a regular file, not a symlink or special file")
			}
			if inputInfo != nil && os.SameFile(inputInfo, info) {
				return fmt.Errorf("output aliases input source")
			}
			for _, previous := range outputs[:i] {
				if previous.info != nil && os.SameFile(previous.info, info) {
					return fmt.Errorf("output files alias one another")
				}
			}
			entry.mode = info.Mode().Perm()
			entry.info = info
		} else if !os.IsNotExist(err) {
			return err
		}
		outputs[i] = entry
	}
	defer func() {
		for _, f := range outputs {
			if f.temp != "" {
				_ = os.Remove(f.temp)
			}
		}
	}()
	for i := range outputs {
		f, err := os.CreateTemp(filepath.Dir(outputs[i].dest), ".gtroff-*")
		if err != nil {
			return err
		}
		outputs[i].temp = f.Name()
		n, writeErr := f.Write(files[i].Data)
		if writeErr == nil && n != len(files[i].Data) {
			writeErr = fmt.Errorf("short output file write")
		}
		if writeErr == nil {
			writeErr = f.Chmod(outputs[i].mode)
		}
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	for _, f := range outputs {
		if err := os.Rename(f.temp, f.dest); err != nil {
			return fmt.Errorf("replace %q: %w", f.dest, err)
		}
	}
	return nil
}
