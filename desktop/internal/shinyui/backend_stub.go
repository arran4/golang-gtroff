//go:build (!linux && !windows && !darwin) || (darwin && !cgo) || android

package shinyui

import (
	"fmt"
	"github.com/arran4/golang-gtroff/viewer"
	"runtime"
)

func BackendError() error {
	return fmt.Errorf("Shiny desktop backend unavailable for %s/%s in this build; macOS GUI requires native CGO and Apple frameworks; headless gtroff remains supported", runtime.GOOS, runtime.GOARCH)
}
func Show(*viewer.Session) error { return BackendError() }
