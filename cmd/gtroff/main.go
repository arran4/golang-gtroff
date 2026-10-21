// Command gtroff inspects and validates roff syntax without evaluating it.
package main

import (
	"io"
	"os"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/cli"
)

// Values may be supplied by release builds with -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Environment{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
		Open:        func(path string) (io.ReadCloser, error) { return os.Open(path) },
		WriteOutput: atomicfile.Write,
		WriteOutputs: func(source string, files []cli.OutputFile) error {
			outputs := make([]atomicfile.File, len(files))
			for i, f := range files {
				outputs[i] = atomicfile.File{Path: f.Path, Data: f.Data}
			}
			return atomicfile.WriteMany(source, outputs)
		},
		Build: cli.BuildInfo{Version: version, Commit: commit, Date: date},
	}))
}
