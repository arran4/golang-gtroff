// Command gtroff inspects and validates roff syntax without evaluating it.
package main

import (
	"context"
	"github.com/arran4/golang-gtroff/manlib"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

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
	ctx := context.Background()
	cancel := func() {}
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		ctx, cancel = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	}
	code := cli.Run(os.Args[1:], cli.Environment{
		Context: ctx,
		OpenRoot: func(path string) (manlib.Root, io.Closer, error) {
			d, err := manlib.OpenDirectory(path)
			if err != nil {
				return manlib.Root{}, nil, err
			}
			return d.Root("configured"), d, nil
		},
		Listen: func(address string) (net.Listener, error) { return net.Listen("tcp", address) },
		SystemSearch: func() (manlib.SearchProvider, error) {
			return manlib.DetectSystemSearch(exec.LookPath, manlib.RunCommand, time.Second, 64<<10)
		},
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
	})
	cancel()
	os.Exit(code)
}
