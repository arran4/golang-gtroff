package cli

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/arran4/golang-gtroff/agentskill"
	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/manlib"
)

// ProcessEnvironment supplies the explicit process capabilities used by the
// generated core entrypoint and optional scripting build. Constructing it does
// not open roots, search the system, listen or execute another program.
func ProcessEnvironment(ctx context.Context, in io.Reader, out, stderr io.Writer, build BuildInfo) Environment {
	return Environment{
		Context: ctx, In: in, Out: out, Err: stderr, Build: build,
		Open: func(path string) (io.ReadCloser, error) { return os.Open(path) },
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
		WriteOutput: atomicfile.Write, WriteSourceOutput: atomicfile.WriteSources,
		ExportSkill: agentskill.Export,
		CommitEdits: atomicfile.WriteEdits, WriteEditedOutput: atomicfile.WriteEditedOutput,
		WriteOutputs: func(source string, files []OutputFile) error {
			outputs := make([]atomicfile.File, len(files))
			for i, f := range files {
				outputs[i] = atomicfile.File{Path: f.Path, Data: f.Data}
			}
			return atomicfile.WriteMany(source, outputs)
		},
	}
}
