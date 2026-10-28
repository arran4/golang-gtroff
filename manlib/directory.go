package manlib

import "os"

// Directory owns an OS root handle. Close it after Build; the resulting Index
// retains source snapshots and never opens files while answering HTTP requests.
type Directory struct{ root *os.Root }

func OpenDirectory(path string) (*Directory, error) {
	r, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &Directory{r}, nil
}
func (d *Directory) Root(label string) Root { return Root{Label: label, FS: d.root.FS()} }
func (d *Directory) Close() error           { return d.root.Close() }
