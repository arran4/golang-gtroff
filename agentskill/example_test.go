package agentskill_test

import (
	"context"
	"fmt"
	"os"

	"github.com/arran4/golang-gtroff/agentskill"
)

func ExampleLookup() {
	bundle, err := agentskill.Lookup("gtroff-query-extract")
	if err != nil {
		panic(err)
	}
	info := bundle.Info()
	fmt.Println(info.Name, info.Version)
	manifest := bundle.Manifest()
	fmt.Println(manifest.Entrypoint, manifest.Contracts.QuerySchema, manifest.Contracts.ExtractSchema)
	// Output:
	// gtroff-query-extract 1.0.0
	// SKILL.md 1 1
}
func ExampleExport() {
	parent, err := os.MkdirTemp("", "gtroff-skills-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(parent)
	receipt, err := agentskill.Export(context.Background(), "gtroff-query-extract", parent)
	if err != nil {
		panic(err)
	}
	fmt.Println(receipt.Info.Name, receipt.Info.Version, len(receipt.Files))
	// The caller explicitly chose this temporary parent; no agent config was used.
	// Output: gtroff-query-extract 1.0.0 4
}
