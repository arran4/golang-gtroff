// Command docgen checks or explicitly refreshes man pages and completions.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/cli"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	f := flag.NewFlagSet("docgen", flag.ContinueOnError)
	write := f.Bool("write", false, "explicitly regenerate command documentation")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	type job struct {
		path string
		args []string
	}
	jobs := []job{{"docs/generated/man/gtroff.1", []string{"man"}}}
	for _, c := range cli.DocumentationCommands() {
		jobs = append(jobs, job{"docs/generated/man/gtroff-" + c + ".1", []string{"man", c}})
	}
	for _, s := range []struct{ shell, name string }{{"bash", "gtroff.bash"}, {"zsh", "_gtroff"}, {"fish", "gtroff.fish"}, {"powershell", "gtroff.ps1"}} {
		jobs = append(jobs, job{"docs/generated/completions/" + s.name, []string{"completion", s.shell}})
	}
	var outputs []atomicfile.File
	for _, job := range jobs {
		var out, stderr bytes.Buffer
		code := cli.Run(job.args, cli.Environment{Out: &out, Err: &stderr})
		if code != 0 || stderr.Len() != 0 {
			return fmt.Errorf("%s: exit %d: %s", job.path, code, stderr.String())
		}
		outputs = append(outputs, atomicfile.File{Path: job.path, Data: bytes.Clone(out.Bytes())})
	}
	if *write {
		for _, out := range outputs {
			if err := os.MkdirAll(filepath.Dir(out.Path), 0755); err != nil {
				return err
			}
		}
		if err := atomicfile.WriteMany("", outputs); err != nil {
			return err
		}
		for _, out := range outputs {
			if err := os.Chmod(out.Path, 0644); err != nil {
				return err
			}
		}
	}
	for _, out := range outputs {
		data, err := os.ReadFile(out.Path)
		if err != nil || !bytes.Equal(data, out.Data) {
			return fmt.Errorf("stale %s; refresh with go run ./cmd/docgen --write", out.Path)
		}
	}
	return nil
}
