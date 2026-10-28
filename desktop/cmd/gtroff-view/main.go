// Command gtroff-view is the optional Shiny desktop entrypoint.
package main

import (
	"io"
	"os"

	"github.com/arran4/golang-gtroff/desktop/internal/app"
	"github.com/arran4/golang-gtroff/desktop/internal/shinyui"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func main() {
	os.Exit(app.Run(os.Args[1:], app.Environment{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Open: func(path string) (io.ReadCloser, error) { return os.Open(path) }, Show: shinyui.Show, BackendError: shinyui.BackendError(), Version: version, Commit: commit, Date: date}))
}
