package agentskill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Receipt describes a completed explicit directory export. The deterministic
// manifest/files have no host paths or timestamps; Directory is host context.
type Receipt struct {
	SchemaVersion int     `json:"schemaVersion"`
	Info          Info    `json:"skill"`
	Directory     string  `json:"directory"`
	Files         []Entry `json:"files"`
}

// Export copies one known bundled skill into parent/name. parent must be explicit;
// missing parents are created. Existing destinations (including symlinks and
// identical skills) are refused. Files stage/sync in an owned sibling directory,
// then one rename publishes the bundle. This is an exclusive-writer operation,
// not protection against hostile concurrent filesystem mutation or crash recovery.
// It never selects an agent config path or executes an artifact.
func Export(ctx context.Context, name, parent string) (receipt *Receipt, err error) {
	return export(ctx, name, parent, os.Rename)
}
func export(ctx context.Context, name, parent string, rename func(string, string) error) (receipt *Receipt, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bundle, err := Lookup(name)
	if err != nil {
		return nil, err
	}
	if parent == "" {
		return nil, fmt.Errorf("explicit skill export directory required")
	}
	root, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	destination := filepath.Join(root, name)
	if _, err := os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrConflict, destination)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(root, ".gtroff-skill-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if stage != "" {
			if cleanupErr := os.RemoveAll(stage); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("clean owned skill staging %s: %w", stage, cleanupErr))
			}
		}
	}()
	files := bundle.Files()
	receipt = &Receipt{SchemaVersion: 1, Info: bundle.Info(), Directory: destination}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !validPath(file.Path) {
			return nil, ErrPath
		}
		path := filepath.Join(stage, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		if err := writeStaged(path, file.Data); err != nil {
			return nil, err
		}
		receipt.Files = append(receipt.Files, Entry{file.Path, len(file.Data), digest(file.Data)})
	}
	if err := os.Chmod(stage, 0755); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Refuse an ordinary concurrently created destination too. Final rename
	// atomicity and exclusion ultimately belong to the trusted host filesystem.
	if _, err := os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrConflict, destination)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := rename(stage, destination); err != nil {
		return nil, fmt.Errorf("publish skill bundle: %w", err)
	}
	stage = ""
	return receipt, nil
}
func writeStaged(path string, data []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	if n, err := file.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("short skill file write")
	}
	return file.Sync()
}
