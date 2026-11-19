package edit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/query"
)

func TestCreateCommitAndFailedWriteRecovery(t *testing.T) {
	w, e := Create(context.Background(), "new.1", man.Metadata{Title: "NEW", Section: "1"}, Options{})
	if e != nil {
		t.Fatal(e)
	}
	execute(t, w, Command{Select: Root(), Operation: AppendSection, Heading: "NAME", Text: "new - example"})
	before := data(t, w)
	failure := errors.New("disk unavailable")
	calls := 0
	commit := func(ctx context.Context, b CommitBatch) error {
		calls++
		if len(b.Changes) != 1 || len(b.Guards) != 1 {
			t.Fatal(b)
		}
		b.Changes[0].After[0] = 'z'
		if calls == 1 {
			return failure
		}
		return nil
	}
	if e := w.Commit(context.Background(), commit); !errors.Is(e, failure) {
		t.Fatal(e)
	}
	if len(w.Changes()) != 1 || data(t, w) != before {
		t.Fatal("lost changes on failed write")
	}
	if e := w.Commit(context.Background(), commit); e != nil {
		t.Fatal(e)
	}
	if len(w.Changes()) != 0 || data(t, w) != before {
		t.Fatal("commit baseline")
	}
	var out bytes.Buffer
	if e := w.WriteSource(context.Background(), &out, w.root); e != nil {
		t.Fatal(e)
	}
	if out.String() != before {
		t.Fatal(out.String())
	}
	if e := w.Commit(context.Background(), nil); !errors.Is(e, ErrTarget) {
		t.Fatal(e)
	}
}
func TestPreviewExactInvalidBytesAndWriterFailures(t *testing.T) {
	w := working(t, ".SH X\r\n\xff old\r\n", Options{})
	p, e := w.Plan(context.Background(), Command{Select: Root(), Operation: FindReplaceSource, Find: "old", Text: "new"})
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e = p.WritePreview(context.Background(), &out, "json", 0); e != nil {
		t.Fatal(e)
	}
	var report struct{ Changes []Change }
	if e = json.Unmarshal(out.Bytes(), &report); e != nil {
		t.Fatal(e)
	}
	if string(report.Changes[0].Before) != ".SH X\r\n\xff old\r\n" {
		t.Fatal("invalid bytes lost")
	}
	out.Reset()
	if e = p.WritePreview(context.Background(), &out, "diff", 0); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), `- "old"`) || !strings.Contains(out.String(), `+ "new"`) {
		t.Fatal(out.String())
	}
	out.Reset()
	if e = p.WritePreview(context.Background(), &out, "json", 8); !errors.Is(e, ErrLimit) || out.Len() != 0 {
		t.Fatal(e)
	}
	if e = w.WriteSource(context.Background(), badWriter{count: 0}, w.root); !errors.Is(e, io.ErrShortWrite) {
		t.Fatal(e)
	}
	failure := errors.New("writer failed")
	if e = w.WriteSource(context.Background(), badWriter{err: failure}, w.root); !errors.Is(e, failure) {
		t.Fatal(e)
	}
	if e = w.WriteSource(context.Background(), badWriter{count: -1}, w.root); e == nil {
		t.Fatal("invalid count")
	}
	var nilBuffer *bytes.Buffer
	if e = w.WriteSource(context.Background(), nilBuffer, w.root); e == nil {
		t.Fatal("typed nil writer")
	}
	if len(w.Changes()) != 0 {
		t.Fatal("output mutated source")
	}
}

type badWriter struct {
	count int
	err   error
}

func (w badWriter) Write(p []byte) (int, error) { return w.count, w.err }
func TestCommandFileValidation(t *testing.T) {
	for _, s := range []string{`{}`, `{"schemaVersion":1,"commands":[]}`, `{"schemaVersion":1,"commands":[{"selector":"source","operation":"delete","unknown":1}]}`, `{"schemaVersion":1,"commands":[{"selector":"source","operation":"exec"}]}`, `{"schemaVersion":1,"commands":[{"selector":"source","operation":"delete"}]} {}`, strings.Repeat("x", 65537)} {
		if _, e := ParseCommands([]byte(s)); e == nil {
			t.Fatal("accepted", s[:min(len(s), 80)])
		}
	}
	commands, e := ParseCommands([]byte(`{"schemaVersion":1,"commands":[{"selector":"source[source-id=1]","operation":"append-section","heading":"NAME","text":"sample"}]}`))
	if e != nil {
		t.Fatal(e)
	}
	w := working(t, "", Options{})
	p, e := w.Plan(context.Background(), commands...)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Apply(p); e != nil {
		t.Fatal(e)
	}
	r, e := w.Query(context.Background(), query.Select(query.Section), query.Options{})
	if e != nil || len(r.Matches) != 1 {
		t.Fatal(e)
	}
}
