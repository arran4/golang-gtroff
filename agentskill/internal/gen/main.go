// Command gen copies the repository's existing license verbatim into the bundled
// portable artifact. It does not choose or change licensing terms.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/arran4/golang-gtroff/internal/atomicfile"
)

func main() {
	data, err := os.ReadFile("../LICENSE")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	const target = "builtin/gtroff-query-extract/LICENSE"
	old, err := os.ReadFile(target)
	if err == nil && bytes.Equal(old, data) {
		return
	}
	if err = atomicfile.WriteGenerated([]atomicfile.File{{Path: target, Data: data}}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
