// Package agentskill exposes bundled, portable agent instructions for read-only
// roff query/extraction workflows. It never chooses an agent configuration path,
// opens a network connection or executes a bundled command.
package agentskill

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"sort"
	"strings"
)

var (
	// ErrNotFound identifies a name outside the embedded bundle registry.
	ErrNotFound = errors.New("unknown bundled agent skill")
	// ErrPath identifies an invalid artifact-relative path.
	ErrPath = errors.New("invalid skill artifact path")
	// ErrConflict identifies an existing export destination.
	ErrConflict = errors.New("skill destination already exists")
)

// Name identifies the bundled query/extraction recipe.
const Name = "gtroff-query-extract"

// Version is the content/contract version of the bundled recipe.
const Version = "1.0.0"
const description = "Locate and cite roff/man documentation fragments with structural queries and source-linked extraction records."

//go:embed builtin/gtroff-query-extract/* builtin/gtroff-query-extract/references/*
var bundled embed.FS

// Contracts names capability/schema requirements rather than claiming that a
// particular unreleased executable version implements every roadmap feature.
type Contracts struct {
	InventorySchema int      `json:"inventorySchema"`
	QuerySchema     int      `json:"querySchema"`
	ExtractSchema   int      `json:"extractSchema"`
	Capabilities    []string `json:"capabilities"`
}

// Entry identifies the exact bytes of one exported artifact.
type Entry struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest describes portable content and the CLI contracts it requires.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Description   string    `json:"description"`
	Format        string    `json:"format"`
	Entrypoint    string    `json:"entrypoint"`
	Repository    string    `json:"repository"`
	Contracts     Contracts `json:"contracts"`
	Files         []Entry   `json:"files"`
}

// Info is a discovery summary whose digest covers the complete manifest.
type Info struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Description    string `json:"description"`
	ManifestSHA256 string `json:"manifestSha256"`
}

// File contains a copy of one exported artifact, including the deterministic
// manifest.json. Manifest.Files covers authored assets, excluding itself.
type File struct {
	Path string
	Data []byte
}

// Bundle is immutable. Obtain it through Lookup; byte/slice accessors copy data.
type Bundle struct {
	manifest Manifest
	files    []File
	encoded  []byte
}

// List returns summaries of all bundled recipes in deterministic name order.
func List() ([]Info, error) {
	bundle, err := Lookup(Name)
	if err != nil {
		return nil, err
	}
	return []Info{bundle.Info()}, nil
}

// Lookup loads a known embedded recipe without opening host files.
func Lookup(name string) (*Bundle, error) {
	if name != Name {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	b := &Bundle{manifest: Manifest{
		SchemaVersion: 1,
		Name:          Name, Version: Version, Description: description,
		Format: "portable-skill-v1", Entrypoint: "SKILL.md",
		Repository: "https://github.com/arran4/golang-gtroff",
		Contracts: Contracts{
			InventorySchema: 1, QuerySchema: 1, ExtractSchema: 1,
			Capabilities: []string{"stateless-query", "structural-selectors", "record-extraction", "source-provenance", "bounded-includes"},
		},
	}}
	prefix := "builtin/" + Name + "/"
	err := fs.WalkDir(bundled, "builtin/"+Name, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative := strings.TrimPrefix(path, prefix)
		if !validPath(relative) {
			return ErrPath
		}
		data, err := bundled.ReadFile(path)
		if err != nil {
			return err
		}
		b.files = append(b.files, File{relative, data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(b.files, func(i, j int) bool { return b.files[i].Path < b.files[j].Path })
	found := false
	for _, file := range b.files {
		if file.Path == b.manifest.Entrypoint {
			found = true
		}
		b.manifest.Files = append(b.manifest.Files, Entry{file.Path, len(file.Data), digest(file.Data)})
	}
	if !found {
		return nil, fmt.Errorf("missing bundled skill entrypoint")
	}
	b.encoded, err = json.MarshalIndent(b.manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	b.encoded = append(b.encoded, '\n')
	return b, nil
}

// Info returns the recipe identity and deterministic manifest digest.
func (b *Bundle) Info() Info {
	return Info{b.manifest.Name, b.manifest.Version, b.manifest.Description, digest(b.encoded)}
}

// Manifest returns a copy of the content/contract manifest.
func (b *Bundle) Manifest() Manifest {
	m := b.manifest
	m.Files = append([]Entry(nil), m.Files...)
	m.Contracts.Capabilities = append([]string(nil), m.Contracts.Capabilities...)
	return m
}

// Files returns sorted copies of all artifacts, including manifest.json.
func (b *Bundle) Files() []File {
	out := make([]File, 0, len(b.files)+1)
	for _, file := range b.files {
		out = append(out, File{file.Path, append([]byte(nil), file.Data...)})
	}
	out = append(out, File{"manifest.json", append([]byte(nil), b.encoded...)})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ReadFile returns a copy of one clean relative artifact path.
func (b *Bundle) ReadFile(path string) ([]byte, error) {
	if !validPath(path) {
		return nil, ErrPath
	}
	if path == "manifest.json" {
		return append([]byte(nil), b.encoded...), nil
	}
	for _, file := range b.files {
		if file.Path == path {
			return append([]byte(nil), file.Data...), nil
		}
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, path)
}

// WriteFile writes one explicitly named artifact to a caller-owned writer. It
// performs no file opens. A downstream writer may retain a prefix on I/O failure.
func (b *Bundle) WriteFile(ctx context.Context, w io.Writer, path string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := b.ReadFile(path)
	if err != nil {
		return err
	}
	if w == nil || nilWriter(w) {
		return fmt.Errorf("nil skill writer")
	}
	n, err := w.Write(data)
	if n < 0 || n > len(data) {
		return fmt.Errorf("invalid skill writer count")
	}
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return ctx.Err()
}
func validPath(path string) bool {
	return fs.ValidPath(path) && path != "." && !strings.Contains(path, "\\")
}
func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func nilWriter(w io.Writer) bool {
	v := reflect.ValueOf(w)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
