// Command releasecheck inspects and smoke-tests actual GoReleaser artifacts.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/arran4/golang-gtroff/internal/releasecheck"
)

func main() {
	dir := flag.String("dist", "dist", "GoReleaser artifact directory")
	viewer := flag.Bool("viewer", false, "verify the optional Linux viewer distribution")
	native := flag.Bool("native", false, "execute this host's archived binary after inspection")
	target := flag.String("target", "", "require the smoke host to match OS/architecture")
	flag.Parse()
	if *target != "" && *target != runtime.GOOS+"/"+runtime.GOARCH {
		fmt.Fprintln(os.Stderr, "native host does not match required target", *target)
		os.Exit(1)
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(2)
	}
	project := "gtroff"
	if *viewer {
		project = "gtroff-view"
	}
	if err := releasecheck.Check(*dir, project, *native, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
