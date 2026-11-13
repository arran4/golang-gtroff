# Reproducible generation

The parser generator owns `macros.go`, `parseline.go`, `catalogue.go` and
`docs/generated/catalogue-support.json`. Its inputs are the embedded canonical
`cmd/gen/in.json`, normalized schema, generation plan and `*.template` files.
Handwritten parser/handler code stays outside those outputs.
[Catalogue documentation references and package bindings](provenance/corrections.md)
are validated before the generation plan is built.

```sh
# Read-only: validate inputs, render candidates, compare committed bytes.
go run ./cmd/gen -check

# Explicitly regenerate parser files, check examples, and refresh CLI docs.
go generate ./...
go generate ./...
git diff --exit-code
```

When changing a template/schema intentionally, regeneration will produce a diff;
review and commit that change, then run the clean-tree check. Do not repair a
generated Go file by hand. Go outputs have standard leading generated-file
headers and pass through `go/format`. Sorted plans and stable JSON avoid dates,
temporary names and machine paths in emitted bytes. Runtime route tests are
separate evidence from generation determinism.

`-check` never writes destinations, creates directories or stages files. Missing
or stale output is an error. The optional `-macrosloc`, `-parselineloc`,
`-catalogueloc` and `-supportloc` flags select all four destinations explicitly.
Their parent directories must exist when writing.

Before mutation, `cmd/gen` validates the entire catalogue and renders/formats all
outputs. A failure in a later template or Go formatter leaves every destination
unchanged. It then preflights every destination and stages/syncs all new bytes
and backups in private, uniquely named directories beside the target files.
Symlink/nonregular destinations, duplicate paths and file aliases are refused.
Existing permissions are retained; new generated assets use 0644. A fixed
`OUTPUT.tmp` is never opened or removed.

The shared `atomicfile.WriteGenerated` boundary handles three distinct batches:
the parser's four outputs, `cmd/docgen --write`'s documentation set, and
`cmd/examples --write`'s explicitly approved example refresh. Both documentation
and example tools render their full set before replacement. Normal example and
documentation checks remain read-only. CLI render/export output uses its existing
file-writing contract independently of this tooling helper.

Each replacement is an atomic same-filesystem rename where supported. If an
ordinary later replacement fails, earlier existing files are restored (bytes,
permissions and modification time), and newly created files are removed.
Successful/rolled-back staging is cleaned. If the filesystem also refuses a
restore, the returned error preserves both failures and identifies the retained
staging directory. For a previously existing destination it contains `original`;
backups are not deleted on that path. An originally absent destination has no
original backup.

This is a **single-writer** operation. Staging-time destination changes are
checked, but there is no lock against concurrent editors/generators. Readers may
observe individual renames. The files are not one atomic filesystem object, and
process termination/power loss has no automatic journal recovery guarantee.
Separate commands within `go generate ./...` do not form one global transaction.

TXTAR cases cover late template execution/format errors, malformed inputs and
deterministic output. OS tests exercise aliases, staging/commit failures, mixed
existing/new rollback, changed destinations, retained recovery bytes and cleanup.
Special-identifier tests inspect Unicode, quote/backslash/backtick spelling in
generated Go. Real command tests check generator `-check`, documentation output
failure and example refresh failure. CI runs the read-only parser check, actual
regeneration/clean-tree check, tests and vet; hosted execution status remains
separate from these configured gates.
