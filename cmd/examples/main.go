// Command examples checks deterministic CLI artifacts; --write explicitly refreshes them.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
	"github.com/arran4/golang-gtroff/internal/showcase"
)

func run(args []string, out io.Writer) error {
	f := flag.NewFlagSet("examples", flag.ContinueOnError)
	f.SetOutput(out)
	write := f.Bool("write", false, "explicitly refresh committed CLI examples")
	root := f.String("root", ".", "repository checkout")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	artifacts, err := showcase.Build(os.DirFS(*root))
	if err != nil {
		return err
	}
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	if *write {
		outputs := make([]atomicfile.File, 0, len(names))
		for _, name := range names {
			outputs = append(outputs, atomicfile.File{Path: filepath.Join(*root, "examples", name), Data: artifacts[name]})
		}
		if err := atomicfile.WriteGenerated(outputs); err != nil {
			return err
		}
	}
	stale := false
	for _, name := range names {
		path := filepath.Join(*root, "examples", name)
		want, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(want, artifacts[name]) {
			fmt.Fprintf(out, "stale example: %s\n", name)
			stale = true
		}
	}
	if stale {
		return fmt.Errorf("refresh explicitly: go run ./cmd/examples --write")
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
