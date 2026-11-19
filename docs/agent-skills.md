# Portable agent recipes

`gtroff skills` exposes static, versioned instructions for locating, citing and
extracting roff/man documentation. The bundled `gtroff-query-extract` skill uses
the existing stateless query and extraction interfaces. It is not a runtime
extension; other installed commands remain discoverable through generated help
and `gtroff commands --format json`.

```sh
gtroff skills list --format json
gtroff skills show gtroff-query-extract
gtroff skills show --format json gtroff-query-extract
gtroff help skills export
```

`list` reports schemaVersion 1, name, version, description and manifest SHA-256.
Text `show` writes SKILL.md; JSON `show` writes the complete deterministic manifest.
Neither command opens files. The portable entrypoint links to its focused
`references/query-extract.md`; no vendor-specific UI files or invocation protocol
are required.

## Explicit local export

Choose a parent directory yourself:

```sh
gtroff skills export --dir ./chosen-skills --format json gtroff-query-extract
```

This creates `./chosen-skills/gtroff-query-extract/` containing SKILL.md,
references/query-extract.md, LICENSE and manifest.json. Export is the explicit
local installation mechanism; there is no separate installer, remote marketplace,
registration, implicit home-directory discovery or command execution. An agent
may read the exported entrypoint using its own supported loading mechanism.

Every existing destination is refused, including an identical bundle, file or
symlink. Local customizations are preserved. To examine an updated recipe,
export to a fresh parent and compare it yourself. There is no overwrite flag.
Missing parent directories may be created; unknown names are rejected before any
filesystem mutation.

All files stage and sync in an owned sibling directory before one rename publishes
the bundle. Failure removes that owned staging directory where possible. Parent
directories may remain. Cancellation is checked before publication. This assumes
an exclusive writer and trusted local filesystem; it is not a concurrent
filesystem transaction or a promise of power-loss durability. A stdout failure
while printing the receipt can occur after a successful export. Inspect the
chosen destination before retrying; existing files remain protected.

Exit codes: 0 success, 1 existing destination, 2 invalid arguments/name/format,
3 I/O, cancellation or unavailable host export capability. Embedders must inject
`cli.Environment.ExportSkill` explicitly to permit filesystem exports; the
normal generated executable supplies `agentskill.Export`.

## Version and source contract

Recipe version 1.0.0 describes inventory/query/extraction schema version 1 and
names the actual required capability IDs. It does not infer support from a `dev`
version or promise every roadmap command exists. The manifest uses
`portable-skill-v1`, lists sorted authored-file lengths and SHA-256 hashes, and
excludes its own bytes from that list. The list command hashes the entire
manifest. Export receipts also list manifest.json. Artifacts contain no host paths
or timestamps; only the receipt's directory reflects the chosen destination.

Source and CLI capability changes require reviewing the recipe and its version.
The Go tests execute documented selectors against source-linked reports and
verify the named capabilities. Generated CLI help, inventory and executable
scenarios cover discovery. `go generate ./...` copies the repository's actual
LICENSE verbatim into the bundle; tests reject drift. This packaging makes no
new licensing decision.

## Go API

```go
bundle, err := agentskill.Lookup(agentskill.Name)
if err != nil { return err }
if err := bundle.WriteFile(ctx, output, "SKILL.md"); err != nil { return err }
receipt, err := agentskill.Export(ctx, agentskill.Name, chosenParent)
```

Import `github.com/arran4/golang-gtroff/agentskill`. `List`, `Lookup`, `Manifest`,
`Files` and `ReadFile` expose immutable snapshots through copied slices/bytes.
`WriteFile` only uses the supplied writer; an I/O failure can retain a prefix.
`Export` is the sole filesystem operation. Artifact paths must be relative clean
slash paths; traversal, absolute paths and backslashes are rejected. Unknown
bundle names cannot become filesystem paths. Working external-package examples
are in `agentskill/example_test.go`.
