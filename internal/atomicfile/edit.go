package atomicfile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SourceGuard bounds and hashes a captured physical input. Bytes is its original
// captured length, not an unbounded filesystem read allowance.
type SourceGuard struct {
	Path, Digest string
	Bytes        int
}
type SourceEdit struct {
	Path, Digest string
	Data         []byte
}

// WriteEdits explicitly updates existing captured sources. All original guards,
// including unmodified inputs, must match before staging/commit. Exact proposed
// new bytes are accepted for idempotent recovery retries; unrelated disk changes
// are stale. Symlinks, special files and aliases across source names are rejected.
// This reuses generator staging/sync/rollback: each rename is atomic; group
// visibility/crash atomicity is not promised. Ordinary failures restore prior
// replacements; failed restores retain backups and report recovery paths.
// Callers must provide exclusive write access for the operation's duration.
func WriteEdits(ctx context.Context, edits []SourceEdit, guards []SourceGuard) error {
	return writeEdits(ctx, edits, guards, outputOperations{})
}
func writeEdits(ctx context.Context, edits []SourceEdit, guards []SourceGuard, ops outputOperations) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(edits) > 4096 || len(guards) > 4096 {
		return fmt.Errorf("too many guarded source files")
	}
	guarded, err := canonicalGuards(guards)
	if err != nil {
		return err
	}
	totalEditBytes := 0
	byPath := map[string]SourceEdit{}
	for _, edit := range edits {
		p, err := physicalPath(edit.Path)
		if err != nil {
			return err
		}
		edit.Path = p
		guard, ok := guarded[p]
		if !ok || edit.Digest == "" || edit.Digest != guard.Digest {
			return fmt.Errorf("edit lacks matching captured source guard: %s", p)
		}
		if len(edit.Data) > (64<<20)-totalEditBytes {
			return fmt.Errorf("edited source exceeds 64 MiB")
		}
		totalEditBytes += len(edit.Data)
		if _, ok := byPath[p]; ok {
			return fmt.Errorf("duplicate source edit: %s", p)
		}
		byPath[p] = edit
	}
	if err := checkGuards(ctx, guarded, byPath); err != nil {
		return err
	}
	var outputs []File
	// Preserve caller's deterministic order, skipping already completed writes.
	for _, edit := range edits {
		p, _ := physicalPath(edit.Path)
		entry := byPath[p]
		data, err := readOutput(p, max(guarded[p].Bytes, len(entry.Data)))
		if err != nil {
			return err
		}
		if hashBytes(data) != hashBytes(entry.Data) {
			outputs = append(outputs, File{p, append([]byte(nil), entry.Data...)})
		}
	}
	if len(outputs) == 0 {
		return nil
	}
	ops.maxReadBytes = 64 << 20
	ops.validate = func(staged []stagedOutput) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, s := range staged {
			if hashBytes(s.original) != guarded[s.Path].Digest {
				return fmt.Errorf("stale source before staging: %s", s.Path)
			}
		}
		return checkGuards(ctx, guarded, byPath)
	}
	ops.ready = func() error { return checkGuards(ctx, guarded, byPath) }
	ops.check = ctx.Err
	ops.finished = func() error { return checkGuards(ctx, guarded, byPath) }
	return replaceOutputs(outputs, ops)
}

// WriteEditedOutput writes a copy to an explicit destination after verifying
// every captured source hash. It rejects aliases of all captured source paths.
func WriteEditedOutput(ctx context.Context, destination string, guards []SourceGuard, data []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(data) > 64<<20 {
		return fmt.Errorf("edited output exceeds 64 MiB")
	}
	guarded, err := canonicalGuards(guards)
	if err != nil {
		return err
	}
	if err = checkGuards(ctx, guarded, nil); err != nil {
		return err
	}
	ops := outputOperations{maxReadBytes: 64 << 20, newMode: 0600, check: ctx.Err}
	ops.validate = func(staged []stagedOutput) error {
		for _, s := range staged {
			for p := range guarded {
				if p == s.Path {
					return fmt.Errorf("output aliases input source")
				}
				info, err := os.Lstat(p)
				if err != nil {
					return err
				}
				if s.info != nil && os.SameFile(info, s.info) {
					return fmt.Errorf("output aliases input source")
				}
			}
		}
		return checkGuards(ctx, guarded, nil)
	}
	ops.ready = func() error { return checkGuards(ctx, guarded, nil) }
	ops.finished = func() error { return checkGuards(ctx, guarded, nil) }
	return replaceOutputs([]File{{Path: destination, Data: append([]byte(nil), data...)}}, ops)
}

func physicalPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty source path")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}
func canonicalGuards(guards []SourceGuard) (map[string]SourceGuard, error) {
	if len(guards) > 4096 {
		return nil, fmt.Errorf("too many source guards")
	}
	totalBytes := 0
	result := make(map[string]SourceGuard, len(guards))
	var infos []os.FileInfo
	for _, g := range guards {
		if g.Digest == "" || g.Bytes < 0 || g.Bytes > (64<<20)-totalBytes {
			return nil, fmt.Errorf("invalid source hash/length guard")
		}
		totalBytes += g.Bytes
		p, err := physicalPath(g.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := result[p]; ok {
			return nil, fmt.Errorf("duplicate physical source path: %s", p)
		}
		info, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source write guard requires regular non-symlink file: %s", p)
		}
		for _, previous := range infos {
			if os.SameFile(previous, info) {
				return nil, fmt.Errorf("captured source names alias a physical file: %s", p)
			}
		}
		infos = append(infos, info)
		g.Path = p
		result[p] = g
	}
	return result, nil
}
func checkGuards(ctx context.Context, guards map[string]SourceGuard, edits map[string]SourceEdit) error {
	paths := make([]string, 0, len(guards))
	for p := range guards {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		g := guards[p]
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source changed type: %s", p)
		}
		limit := g.Bytes
		if e, ok := edits[p]; ok {
			limit = max(limit, len(e.Data))
		}
		data, err := readOutput(p, max(limit, 1))
		if err != nil {
			return err
		}
		hash := hashBytes(data)
		if hash == g.Digest {
			continue
		}
		if e, ok := edits[p]; ok && hash == hashBytes(e.Data) {
			continue
		}
		return fmt.Errorf("stale captured source: %s", p)
	}
	return nil
}
func hashBytes(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
