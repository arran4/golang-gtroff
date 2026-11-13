package atomicfile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Only operations needed for deterministic failure injection are replaceable.
// All other filesystem work is intentionally exercised at the real OS boundary.
type outputOperations struct {
	mkdirTemp func(string, string) (string, error)
	rename    func(string, string) error
	remove    func(string) error
}

type stagedOutput struct {
	File
	info                os.FileInfo
	original            []byte
	dir, next, previous string
	keep                bool
}

// WriteGenerated preflights all paths, stages and syncs every file and backup,
// then replaces each file by same-filesystem rename. Ordinary commit errors
// restore earlier files; failed restores retain their backup and report its path.
// This is a single-writer operation, not atomic group visibility/crash recovery.
// Existing permissions are retained; new generated assets have mode 0644.
func WriteGenerated(outputs []File) error {
	return replaceOutputs(outputs, outputOperations{})
}

func replaceOutputs(outputs []File, ops outputOperations) (err error) {
	if ops.mkdirTemp == nil {
		ops.mkdirTemp = os.MkdirTemp
	}
	if ops.rename == nil {
		ops.rename = os.Rename
	}
	if ops.remove == nil {
		ops.remove = os.Remove
	}
	staged, err := preflightOutputs(outputs)
	if err != nil {
		return err
	}
	defer func() {
		for _, output := range staged {
			if output.dir != "" && !output.keep {
				if cleanupErr := os.RemoveAll(output.dir); cleanupErr != nil {
					err = errors.Join(err, fmt.Errorf("clean staging %s: %w", output.dir, cleanupErr))
				}
			}
		}
	}()
	for i := range staged {
		output := &staged[i]
		output.dir, err = ops.mkdirTemp(filepath.Dir(output.Path), ".gtroff-gen-*")
		if err != nil {
			return fmt.Errorf("stage %s: %w", output.Path, err)
		}
		output.next = filepath.Join(output.dir, "new")
		mode := os.FileMode(0644)
		if output.info != nil {
			mode = output.info.Mode().Perm()
			output.previous = filepath.Join(output.dir, "original")
			if err := stageBytes(output.previous, output.original, mode); err != nil {
				return err
			}
			if err := os.Chtimes(output.previous, output.info.ModTime(), output.info.ModTime()); err != nil {
				return err
			}
		}
		if err := stageBytes(output.next, output.Data, mode); err != nil {
			return err
		}
	}
	for i := range staged {
		if err := outputUnchanged(staged[i]); err != nil {
			return rollbackOutputs(staged[:i], ops, err)
		}
		if err := ops.rename(staged[i].next, staged[i].Path); err != nil {
			return rollbackOutputs(staged[:i], ops, fmt.Errorf("replace %s: %w", staged[i].Path, err))
		}
	}
	return nil
}

func preflightOutputs(outputs []File) ([]stagedOutput, error) {
	if len(outputs) == 0 {
		return nil, fmt.Errorf("no generated outputs")
	}
	paths := map[string]bool{}
	staged := make([]stagedOutput, 0, len(outputs))
	for _, output := range outputs {
		if output.Path == "" {
			return nil, fmt.Errorf("empty generated output path")
		}
		// Windows Abs uses GetFullPathName, which can erase trailing dots and
		// spaces. Validate the caller's basename before that normalization.
		base := filepath.Base(output.Path)
		if runtime.GOOS == "windows" && (!filepath.IsLocal(base) || strings.Contains(base, ":") || strings.TrimRight(base, ". ") != base) {
			return nil, fmt.Errorf("ambiguous Windows generated filename: %s", base)
		}
		abs, err := filepath.Abs(output.Path)
		if err != nil {
			return nil, err
		}
		base = filepath.Base(abs)
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return nil, fmt.Errorf("resolve output directory for %s: %w", output.Path, err)
		}
		output.Path = filepath.Join(parent, base)
		key := output.Path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if paths[key] {
			return nil, fmt.Errorf("duplicate generated output %s", output.Path)
		}
		paths[key] = true
		entry := stagedOutput{File: output}
		entry.info, err = os.Lstat(output.Path)
		if err == nil {
			if !entry.info.Mode().IsRegular() {
				return nil, fmt.Errorf("generated output must be a regular file: %s", output.Path)
			}
			for _, previous := range staged {
				if previous.info != nil && os.SameFile(previous.info, entry.info) {
					return nil, fmt.Errorf("generated outputs alias each other: %s", output.Path)
				}
			}
			entry.original, err = os.ReadFile(output.Path)
			if err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		staged = append(staged, entry)
	}
	return staged, nil
}

func stageBytes(path string, data []byte, mode os.FileMode) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create staged file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	if n, err := file.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("short generated file write")
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	return file.Sync()
}

func outputUnchanged(output stagedOutput) error {
	info, err := os.Lstat(output.Path)
	if output.info == nil && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if output.info == nil || !info.Mode().IsRegular() || !os.SameFile(output.info, info) || info.Mode() != output.info.Mode() || !info.ModTime().Equal(output.info.ModTime()) {
		return fmt.Errorf("generated destination changed during staging: %s", output.Path)
	}
	data, err := os.ReadFile(output.Path)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, output.original) {
		return fmt.Errorf("generated destination changed during staging: %s", output.Path)
	}
	return nil
}

func rollbackOutputs(applied []stagedOutput, ops outputOperations, failure error) error {
	for i := len(applied) - 1; i >= 0; i-- {
		output := &applied[i]
		var err error
		if output.info == nil {
			err = ops.remove(output.Path)
		} else {
			err = ops.rename(output.previous, output.Path)
		}
		if err != nil {
			output.keep = true
			failure = errors.Join(failure, fmt.Errorf("restore %s failed: %w; recovery staging retained at %s", output.Path, err, output.dir))
		}
	}
	return failure
}
