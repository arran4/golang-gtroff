package edit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/arran4/golang-gtroff/convert"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

// Create constructs a new in-memory document with an optional canonical .TH.
// Name labels source ownership; no file is created. Empty metadata gives an empty
// document. Supported literal content can then be appended with normal commands.
func Create(ctx context.Context, name string, metadata man.Metadata, o Options) (*Working, error) {
	if name == "" {
		return nil, ErrOptions
	}
	for _, s := range []string{metadata.Title, metadata.Section, metadata.Date, metadata.Source, metadata.Manual} {
		if err := literal(s, true); err != nil {
			return nil, err
		}
	}
	var buffer bytes.Buffer
	_, err := convert.WriteMan(&buffer, &man.Document{Version: 1, Metadata: metadata, Nodes: []man.Node{{ID: 1, Kind: man.DocumentNode}}}, convert.Options{Strict: true, MaxOutputBytes: o.MaxBytes})
	if err != nil {
		return nil, err
	}
	options := o.Source
	options.FS = nil
	set, err := source.Read(ctx, &buffer, name, options)
	if err != nil {
		return nil, err
	}
	return New(set, o)
}

// Guard describes the unchanged captured baseline, including files not edited.
// A host maps Names to explicitly authorized physical paths and verifies hashes
// before writing. Bytes bounds reads; digest comparison uses exact bytes.
type Guard struct {
	Name   string
	Digest string
	Bytes  int
}
type CommitBatch struct {
	Guards  []Guard
	Changes []Change
}
type Committer func(context.Context, CommitBatch) error

// Commit invokes one explicit, trusted output capability. A failure retains all
// in-memory changes and original baselines for inspection/retry. Success adopts
// the current bytes as the new baseline and invalidates outstanding plans. The
// host owns atomic replacement, alias checks, disk staleness and recovery policy;
// a callback that partially writes must report recovery details.
func (w *Working) Commit(ctx context.Context, commit Committer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if commit == nil {
		return fmt.Errorf("%w: no explicit commit capability", ErrTarget)
	}
	batch := CommitBatch{Changes: w.Changes()}
	for _, n := range w.Names() {
		s := w.original[n]
		batch.Guards = append(batch.Guards, Guard{n, digest(s), len(s)})
	}
	if err := commit(ctx, batch); err != nil {
		return err
	}
	w.original = cloneFiles(w.files)
	w.version++
	return nil
}

// WriteSource writes one current physical file, never an include expansion.
// It does not acknowledge changes to original files. The explicit writer owns
// its I/O behaviour; a writer error may leave a prefix on that writer.
func (w *Working) WriteSource(ctx context.Context, out io.Writer, name string) error {
	s, ok := w.files[name]
	if !ok {
		return ErrTarget
	}
	return writeBytes(ctx, out, []byte(s))
}

// WritePreview emits a deterministic review report. "diff" is a byte-range
// review format, not a patch(1) unified-diff program. Quoting and JSON base64
// preserve invalid bytes. MaxBytes bounds materialized output, default 32 MiB.
// Construction/limit errors leave the writer untouched.
func (p *Plan) WritePreview(ctx context.Context, out io.Writer, format string, maxBytes int) error {
	if p == nil {
		return ErrStale
	}
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	if maxBytes > 128<<20 {
		return ErrOptions
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var b bytes.Buffer
	finalChanges := p.Changes()
	if format == "json" {
		// Account conservatively before JSON allocation (base64 plus escaping).
		size := 0
		for _, c := range finalChanges {
			if len(c.Before)+len(c.After) > maxBytes/2-size {
				return ErrLimit
			}
			size += len(c.Before) + len(c.After)
		}
		for _, patch := range p.patches {
			size += 2*len(patch.Data) + 6*len(patch.Source) + 1024
			if size > maxBytes {
				return ErrLimit
			}
		}
		for _, d := range p.diagnostics {
			size += 6*(len(d.Source)+len(d.Message)+len(d.Code)) + 512
			if size > maxBytes {
				return ErrLimit
			}
		}
		for _, d := range p.set.Diagnostics() {
			size += 6*(len(d.Message)+len(d.Position.Filename)) + 512
			if size > maxBytes {
				return ErrLimit
			}
		}
		err := json.NewEncoder(&b).Encode(struct {
			SchemaVersion     int                 `json:"schemaVersion"`
			Version           uint64              `json:"version"`
			Changes           []Change            `json:"changes"`
			Patches           []Patch             `json:"patches"`
			Diagnostics       []Diagnostic        `json:"diagnostics,omitempty"`
			SourceDiagnostics []source.Diagnostic `json:"sourceDiagnostics,omitempty"`
		}{1, p.version, finalChanges, p.Patches(), p.Diagnostics(), p.set.Diagnostics()})
		if err != nil {
			return err
		}
	} else if format == "diff" {
		for _, c := range finalChanges {
			if err := ctx.Err(); err != nil {
				return err
			}
			// One minimal enclosing changed byte interval per physical file. This
			// compares initial/final bytes even across sequential command steps.
			a, z := 0, 0
			for a < len(c.Before) && a < len(c.After) && c.Before[a] == c.After[a] {
				a++
			}
			for z < len(c.Before)-a && z < len(c.After)-a && c.Before[len(c.Before)-1-z] == c.After[len(c.After)-1-z] {
				z++
			}
			removed, added := c.Before[a:len(c.Before)-z], c.After[a:len(c.After)-z]
			if len(c.Source) > (maxBytes-b.Len())/12 {
				return ErrLimit
			}
			if len(removed)+len(added) > (maxBytes-b.Len())/4 {
				return ErrLimit
			}
			fmt.Fprintf(&b, "--- %q sha256:%s\n+++ %q sha256:%s\n@@ bytes [%d,%d) -> [%d,%d) @@\n- %q\n+ %q\n", c.Source, c.BeforeDigest, c.Source, c.AfterDigest, a, len(c.Before)-z, a, len(c.After)-z, removed, added)
		}
	} else {
		return fmt.Errorf("%w: preview format must be json or diff", ErrOptions)
	}
	if b.Len() > maxBytes {
		return ErrLimit
	}
	return writeBytes(ctx, out, b.Bytes())
}

func writeBytes(ctx context.Context, out io.Writer, data []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if nilWriter(out) {
		return fmt.Errorf("nil edit output writer")
	}
	n, err := out.Write(data)
	if n < 0 || n > len(data) {
		return fmt.Errorf("invalid edit writer count")
	}
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return ctx.Err()
}

func nilWriter(out io.Writer) bool {
	if out == nil {
		return true
	}
	v := reflect.ValueOf(out)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
