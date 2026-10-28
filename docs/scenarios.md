# Shared scenarios and reproducible examples

`TestIntegrationScenarios` walks embedded `testdata/scenarios` with `fs.WalkDir`,
sorts paths, and runs named subtests. The existing `ast`, `cli`, `man`,
`render/html`, `convert`, `image`, `serve` and `viewer` suites keep detailed
feature contracts. This shared suite checks that those public boundaries agree.

## Archive vocabulary

Every archive has a comment explaining intent, `options.json`, and exactly one
of `input.1`, `input.html`, or `"example":true`. The example option selects only
the fixed embedded `examples/man-tool.1`; it cannot open an arbitrary filename.
Unknown members, duplicate names and unknown options fail. Missing input fails;
empty input is valid. Missing `expected-diagnostics.json` means no diagnostics
at any recorded stage, rather than an assertion being skipped.

`status`, `validateCode` and `strictCode` express expected syntax status and
permissive/strict exit codes separately from diagnostic goldens. HTML cases
validate their generated canonical roff, while original HTML diagnostics remain
under `input-html`. `noFinalNewline` removes the archive-added terminal LF;
`crlf` converts the readable source to CRLF. These are explicit byte transforms,
verified against raw AST/model snapshots. Existing AST/CLI suites also contain
literal CRLF archives.

Expectations include a deterministic AST preorder table with syntax extraction,
normalized semantics, both validation results, HTML, canonical roff, and page
geometry/ordered source text. Diagnostics are keyed by pipeline stage. Each node's
raw slice, identity, parent-before-child ordering and tree lookup are asserted.
The pipeline must not mutate the source or semantic model.

Supported cases run both `roff → HTML → roff` and `HTML → roff → HTML`, compare
normalized semantics at every boundary and check second canonical roff stability.
For HTML byte stability, both passes render from canonical roff: exported source
spans/IDs refer to the current input snapshot, so bytes from unrelated input
origins are not compared. The complete HTML remains a golden too.

Negative cases name a `skipLoopReason` and explicit `retain` literals. Those
literals must survive in the semantic text, escaped HTML and reimported canonical
roff. Unsupported provenance may become safe literal text; it may not disappear.
Malformed syntax, unknown requests, includes, unsafe HTML and missing glyphs have
visible diagnostic expectations. No include is opened and no resource is fetched.

The realistic example also builds an index from `fstest.MapFS` and exercises
search, catalogue, page navigation, OPTIONS selection and missing-page handling
through `httptest`. The actual executable suite uses that same source for CLI
and HTTP process checks. No installed man database is involved.

## Raster snapshots

All scenarios check visible source content and glyph bounds, lay out and encode
images, decode them through the standard library, and verify page count and
dimensions. Ordered fragment source text stays in the
TXTAR layout summary. Selected PNG/JPEG bytes live separately in
`testdata/golden/scenarios`; normal tests compare exact bytes. Existing image
contracts additionally compare lossless pixels, check nonblank painting, bounds,
unsupported glyph labels and content preservation. Keeping a small selection of
binary snapshots avoids duplicating every feature golden.

## Explicit updates

Ordinary tests never write expected files. Refresh only the suite being changed:

```sh
go test . -run '^TestIntegrationScenarios$' -update-integration
go test . -run '^TestManScenarios$' -update-man
go test . -run '^TestHTMLScenarios$' -update-html
go test . -run '^TestConvertScenarios$' -update-convert
go test . -run '^TestImageScenarios$' -update-images
go test . -run '^TestServeScenarios$' -update-serve
go test . -run '^TestViewerScenarios$' -update-viewer
```

Review text diffs and visually inspect affected images before committing. Updates
still run semantic/status/content assertions; they do not approve lost content.

## Example generator and executable documentation

`go run ./cmd/examples` checks the committed HTML, PNG, dark JPEG, selected AST,
strict validation JSON, canonical man page and the small HTML conversion output.
`go generate ./...` performs that check after regenerating parser tables. Both
commands are deterministic and work without a display or installed fonts.

```sh
go run ./cmd/examples --write
go run ./cmd/examples
go test ./...
```

The generator builds every artifact through explicit CLI input/output capabilities
before writing any file. A source error prevents a refresh. `--write` is the only
refresh mode; default checks report missing/stale outputs without modifying them.
Existing full-font HTML is about 830 KiB because it embeds four licensed fonts;
the new AST is about 15 KiB and the new raster snapshots total about 40 KiB.
The original input files are never generated. Semantic model JSON has its own
`-update-man` contract. Native viewer captures have a separate opt-in procedure
in [viewer.md](viewer.md), because window-system pixels are platform dependent.

README shell fences beginning with `# smoke` run against a newly built CGO-free
executable in a temporary workspace, including redirection and pagination. The
test invokes argv directly rather than requiring a POSIX shell, decodes generated
images and checks expected output files. External installation/font paths and
interactive viewer/server commands have separate targeted integration checks.
Library examples are executable Go `Example` tests.

Fuzz boundaries already exist for source AST, semantic model, CLI inspection,
HTML import/export, raster layout, HTTP and viewer controller. For example:

```sh
go test . -run '^$' -fuzz '^FuzzParseAST$' -fuzztime 15s
go test ./convert -run '^$' -fuzz '^FuzzHTMLImport$' -fuzztime 15s
```

These conventions follow [TXTAR v2](https://arran4.github.io/blog/post/2026/018-txtar-patterns-for-agents-v2/),
[executable scenarios](https://arran4.github.io/blog/post/2026/046-scenarios-as-executable-application-state/)
and [explicit golden updates](https://arran4.github.io/blog/post/2026/034-golden-test-update-convention/).
