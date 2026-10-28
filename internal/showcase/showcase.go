// Package showcase builds the committed examples through the command boundary.
// Sources are explicit; generation never reads includes, fonts or the network.
package showcase

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"github.com/arran4/golang-gtroff/internal/cli"
)

// Build returns named artifacts without writing to the supplied filesystem.
// The semantic model JSON and native screenshot have separate refresh commands.
func Build(source fs.FS) (map[string][]byte, error) {
	out := make(map[string][]byte)
	jobs := []struct {
		name string
		args []string
	}{
		{"man-tool.html", []string{"render", "--format", "html", "examples/man-tool.1"}},
		{"man-tool.png", []string{"render", "--format", "png", "examples/man-tool.1"}},
		{"man-tool.jpg", []string{"render", "--format", "jpg", "--theme", "dark", "examples/man-tool.1"}},
		{"man-tool.ast.json", []string{"ast", "--select", "section:OPTIONS", "examples/man-tool.1"}},
		{"man-tool.validation.json", []string{"validate", "--strict", "--format", "json", "examples/man-tool.1"}},
		{"man-tool.canonical.1", []string{"convert", "--from", "roff", "--to", "html", "--assets", "omit-fonts", "-i", "examples/man-tool.1"}},
		{"convert.1", []string{"convert", "--from", "html", "--to", "roff", "--strict", "-i", "examples/convert.html"}},
	}
	for _, job := range jobs {
		var output, diagnostics bytes.Buffer
		code := cli.Run(job.args, cli.Environment{Out: &output, Err: &diagnostics,
			Open: func(name string) (io.ReadCloser, error) { return source.Open(name) }})
		if code != 0 || diagnostics.Len() != 0 {
			return nil, fmt.Errorf("%s: exit %d: %s", job.name, code, diagnostics.String())
		}
		if job.name == "man-tool.canonical.1" {
			var canonical, notes bytes.Buffer
			code = cli.Run([]string{"convert", "--from", "html", "--to", "roff", "--strict"}, cli.Environment{In: &output, Out: &canonical, Err: &notes})
			if code != 0 || notes.Len() != 0 {
				return nil, fmt.Errorf("canonical example: exit %d: %s", code, notes.String())
			}
			out[job.name] = bytes.Clone(canonical.Bytes())
		} else {
			out[job.name] = bytes.Clone(output.Bytes())
		}
	}
	return out, nil
}
