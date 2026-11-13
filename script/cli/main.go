// The optional process adapter embeds Tengo in one pure-Go executable. All
// command behaviour belongs to the root CLI, including its generated script
// declarations; this entrypoint defines no additional command grammar.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arran4/golang-gtroff/internal/cli"
	"github.com/arran4/golang-gtroff/script"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func main() { os.Exit(run()) }
func run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	environment := cli.ProcessEnvironment(ctx, os.Stdin, os.Stdout, os.Stderr, cli.BuildInfo{Version: version, Commit: commit, Date: date})
	environment.Script = script.Run
	return cli.Run(os.Args[1:], environment)
}
