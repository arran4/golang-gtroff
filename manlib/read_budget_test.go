package manlib_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
)

var errBudgetRead = errors.New("fixture read failed")
var errBudgetClose = errors.New("fixture close failed")

type budgetFS struct {
	fs.FS
	bytes, opened, closed int
	pages                 []string
	readFault, closeFault map[string]bool
}

func (s *budgetFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	s.opened++
	return &budgetFile{File: f, owner: s, name: name}, nil
}

type budgetFile struct {
	fs.File
	owner *budgetFS
	name  string
}

func (f *budgetFile) Read(p []byte) (int, error) {
	if len(f.owner.pages) == 0 || f.owner.pages[len(f.owner.pages)-1] != f.name {
		f.owner.pages = append(f.owner.pages, f.name)
	}
	n, err := f.File.Read(p)
	f.owner.bytes += n
	if n > 0 && f.owner.readFault[f.name] {
		return n, errBudgetRead
	}
	return n, err
}
func (f *budgetFile) ReadDir(n int) ([]fs.DirEntry, error) { return f.File.(fs.ReadDirFile).ReadDir(n) }
func (f *budgetFile) Close() error {
	f.owner.closed++
	err := f.File.Close()
	if f.owner.closeFault[f.name] {
		return errors.Join(err, errBudgetClose)
	}
	return err
}
func checkBudgetCloses(t *testing.T, sys *budgetFS) {
	t.Helper()
	if sys.opened != sys.closed {
		t.Fatalf("opened=%d closed=%d", sys.opened, sys.closed)
	}
}
func budgetGzip(t *testing.T, data []byte, corrupt bool) []byte {
	t.Helper()
	var out bytes.Buffer
	z := gzip.NewWriter(&out)
	if _, err := z.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	raw := bytes.Clone(out.Bytes())
	if corrupt {
		raw[len(raw)-8] ^= 0xff
	}
	return raw
}

func TestTotalBudgetBoundsRawReads(t *testing.T) {
	for _, suffix := range []string{".1", ".1.gz"} {
		t.Run(suffix, func(t *testing.T) {
			sys := &budgetFS{FS: fstest.MapFS{"a" + suffix: {Data: bytes.Repeat([]byte("x"), 128)}, "b" + suffix: {Data: []byte("never read")}}}
			idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: 32})
			if !errors.Is(err, manlib.ErrLimit) || sys.bytes != 33 || len(idx.Entries()) != 0 || !reflect.DeepEqual(sys.pages, []string{"a" + suffix}) {
				t.Fatalf("total limit not enforced during read: bytes=%d pages=%v error=%v", sys.bytes, sys.pages, err)
			}
			checkBudgetCloses(t, sys)
		})
	}
}

func TestFailedGzipAttemptsConsumeTotalBudget(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     []byte
		decoded int
	}{
		{"header", []byte("not gzip\n"), 0},
		{"checksum", budgetGzip(t, []byte("stored payload\n"), true), len("stored payload\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cost := len(tc.raw) + tc.decoded
			sys := &budgetFS{FS: fstest.MapFS{"a.1.gz": {Data: tc.raw}, "b.1.gz": {Data: tc.raw}, "c.1.gz": {Data: tc.raw}, "d.1": {Data: []byte("never read")}}}
			idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: 2*cost + 2})
			if !errors.Is(err, manlib.ErrLimit) || len(idx.Entries()) != 0 || len(idx.Notices()) != 2 || sys.bytes != 2*len(tc.raw)+3 || !reflect.DeepEqual(sys.pages, []string{"a.1.gz", "b.1.gz", "c.1.gz"}) {
				t.Fatalf("failed decode escaped aggregate budget: bytes=%d notices=%v pages=%v error=%v", sys.bytes, idx.Notices(), sys.pages, err)
			}
			checkBudgetCloses(t, sys)
		})
	}
}

func TestTotalBudgetCountsFailedReadsAndCloseErrors(t *testing.T) {
	for _, readFault := range []bool{false, true} {
		t.Run(map[bool]string{false: "close", true: "read-and-close"}[readFault], func(t *testing.T) {
			sys := &budgetFS{FS: fstest.MapFS{"a.1": {Data: []byte("failure\n")}, "b.1": {Data: []byte("oversized payload\n")}, "c.1": {Data: []byte("never read")}}, readFault: map[string]bool{"a.1": readFault, "b.1": readFault}, closeFault: map[string]bool{"a.1": true, "b.1": true}}
			idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: 12})
			if !errors.Is(err, manlib.ErrLimit) || !errors.Is(err, errBudgetClose) || errors.Is(err, errBudgetRead) != readFault || sys.bytes != 13 || len(idx.Notices()) != 1 || len(idx.Entries()) != 0 {
				t.Fatalf("failure cost/cause lost: bytes=%d error=%v", sys.bytes, err)
			}
			if !reflect.DeepEqual(sys.pages, []string{"a.1", "b.1"}) {
				t.Fatal("read after resource failure", sys.pages)
			}
			checkBudgetCloses(t, sys)
		})
	}
}

func TestTotalBudgetGzipExpansionAndExactBoundary(t *testing.T) {
	source := []byte(".SH NAME\nvisible payload\n")
	raw := budgetGzip(t, source, false)
	cost := len(raw) + len(source)
	for _, limit := range []int{cost - 1, cost} {
		sys := &budgetFS{FS: fstest.MapFS{"page.1.gz": {Data: raw}}}
		idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: limit})
		if errors.Is(err, manlib.ErrLimit) != (limit < cost) || sys.bytes != len(raw) {
			t.Fatalf("raw and decoded budgets separated incorrectly: %v", err)
		}
		if limit == cost {
			if err != nil || len(idx.Entries()) != 1 {
				t.Fatal("exact boundary rejected", err)
			}
			doc, err := idx.Document(context.Background(), "page", "1", "")
			if err != nil || doc.Source() != string(source) {
				t.Fatal("accepted source changed", err)
			}
		}
		checkBudgetCloses(t, sys)
	}
	// A large expansion cannot spend MaxFileBytes after the shared budget runs out.
	raw = budgetGzip(t, bytes.Repeat([]byte("x"), 4096), false)
	sys := &budgetFS{FS: fstest.MapFS{"page.1.gz": {Data: raw}}}
	_, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: len(raw) + 16, MaxFileBytes: 8192})
	if !errors.Is(err, manlib.ErrLimit) {
		t.Fatal("expansion escaped shared budget", err)
	}
	checkBudgetCloses(t, sys)
}

func TestTotalBudgetSharedAcrossRootsRetainsPartialIndex(t *testing.T) {
	first := &budgetFS{FS: fstest.MapFS{"a.1": {Data: []byte("accepted\n")}, "bad.1.gz": {Data: []byte("bad gzip\n")}}}
	second := &budgetFS{FS: fstest.MapFS{"next.1": {Data: []byte(strings.Repeat("x", 128))}, "z.1": {Data: []byte("never read")}}}
	idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "first", FS: first}, {Label: "second", FS: second}}, manlib.Limits{MaxTotalBytes: 20})
	if !errors.Is(err, manlib.ErrLimit) || first.bytes+second.bytes != 21 || len(idx.Entries()) != 1 || len(idx.Notices()) != 1 || !reflect.DeepEqual(second.pages, []string{"next.1"}) {
		t.Fatalf("root budget/reset or partial-index loss: %+v %v", idx.Entries(), err)
	}
	checkBudgetCloses(t, first)
	checkBudgetCloses(t, second)
	doc, err := idx.Document(context.Background(), "a", "1", "")
	if err != nil || doc.Source() != "accepted\n" {
		t.Fatal("accepted snapshot lost", err)
	}
}

func FuzzLibraryReadBudget(f *testing.F) {
	f.Add([]byte(".SH NAME\nvisible\n"), uint16(16), uint8(0))
	f.Add([]byte("stored payload\n"), uint16(128), uint8(2))
	f.Add([]byte{}, uint16(1), uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, limit uint16, mode uint8) {
		if len(data) > 512 {
			t.Skip()
		}
		total := int(limit%2048) + 1
		raw := data
		suffix := ".1"
		cost := len(data)
		corrupt := mode%3 == 2
		if mode%3 != 0 {
			raw = budgetGzip(t, data, corrupt)
			suffix = ".1.gz"
			cost += len(raw)
		}
		sys := &budgetFS{FS: fstest.MapFS{"a" + suffix: {Data: raw}, "b" + suffix: {Data: raw}}}
		idx, err := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{MaxTotalBytes: total, MaxFileBytes: 4096})
		if sys.bytes > total+1 {
			t.Fatalf("raw read exceeded global cap: %d > %d", sys.bytes, total+1)
		}
		if 2*cost > total && err == nil {
			t.Fatal("failed or decoded bytes escaped work budget")
		}
		checkBudgetCloses(t, sys)
		if err == nil && !corrupt {
			if len(idx.Entries()) != 2 {
				t.Fatal("bounded valid pages disappeared")
			}
			doc, err := idx.Document(context.Background(), "a", "1", "")
			if err != nil || doc.Source() != string(data) {
				t.Fatal("source changed", err)
			}
		}
	})
}
