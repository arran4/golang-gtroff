package manlib_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/manlib"
)

type countedFS struct {
	fs.FS
	opened, closed *int
	badRead        bool
}

func (c countedFS) Open(name string) (fs.File, error) {
	f, e := c.FS.Open(name)
	if e != nil {
		return nil, e
	}
	*c.opened++
	return &countedFile{File: f, closed: c.closed, bad: c.badRead && name == "page.1"}, nil
}

type countedFile struct {
	fs.File
	closed *int
	bad    bool
}

func (c *countedFile) Close() error { *c.closed++; return c.File.Close() }
func (c *countedFile) Read(p []byte) (int, error) {
	if c.bad {
		return 0, errors.New("read denied")
	}
	return c.File.Read(p)
}
func (c *countedFile) ReadDir(n int) ([]fs.DirEntry, error) {
	return c.File.(fs.ReadDirFile).ReadDir(n)
}
func TestOwnedFilesCloseOnSuccessAndReadFailure(t *testing.T) {
	for _, bad := range []bool{false, true} {
		opened, closed := 0, 0
		sys := countedFS{FS: fstest.MapFS{"page.1": {Data: []byte("visible source\n")}}, opened: &opened, closed: &closed, badRead: bad}
		idx, e := manlib.Build(context.Background(), []manlib.Root{{Label: "test", FS: sys}}, manlib.Limits{})
		if e != nil {
			t.Fatal(e)
		}
		if opened != closed {
			t.Fatalf("file leak opened=%d closed=%d", opened, closed)
		}
		if bad && len(idx.Notices()) != 1 {
			t.Fatal("read failure not diagnosed")
		}
	}
}
