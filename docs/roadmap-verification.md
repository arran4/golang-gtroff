# Roadmap verification record

This is an evidence map for the feature series under issue #31. Implementation
and local verification are recorded separately from native/hosted acceptance.
It is not a declaration that the roadmap, parent parser backlog or release gate
has passed. The issue bodies and live PR state remain authoritative.

## Branch and review boundaries

PR #19 is still open/unmerged at `75d60db446e4aeb9ea1d16488fd818438492158a`.
All feature PRs remain draft, open and unmerged. Each names its parent and is
based on that parent's head; no existing parent branch was overwritten.

| Issue | Draft PR | Parent | Implementation and primary executable evidence |
| --- | --- | --- | --- |
| #20 AST | #32 | #19 | `ast*.go`, `testdata/ast`, `ast_test.go`, `ast_example_test.go` |
| #21 CLI | #33 | #32 | `cmd/gtroff`, `internal/cli`, `testdata/cli`, executable process tests |
| #22 semantics | #34 | #33 | `man`, `testdata/man`, `examples/man-tool.1` / semantic JSON |
| #23 HTML | #35 | #34 | `renderhtml`, `testdata/render/html`, actual standalone HTML example |
| #24 conversion | #36 | #35 | `convert`, `testdata/convert`, conversion loop and escaping tests |
| #25 images | #37 | #36 | `renderimage`, `testdata/image`, separate binary goldens and PNG/JPEG examples |
| #27 server | #38 | #37 | `manlib`, `testdata/serve`, MapFS/HTTP/process/root-boundary tests |
| #26 viewer | #39 | #38 | `viewer`, isolated `desktop` module, `testdata/viewer`, real X11 smoke |
| #28 integration | #40 | #39 | `integration_scenarios_test.go`, `testdata/scenarios`, `cmd/examples`, README smoke |
| #29 release | #41 | #40 | two GoReleaser configs, `cmd/docgen`, `cmd/releasecheck`, owner workflow, policy scenarios |

No PR was merged and no GitHub release was published. The only test tag was
created in a disposable local clone; it was never pushed. Other repositories,
including md2png, were consulted read-only.

## Feature acceptance coverage

### AST and compatibility (#20)

- Kinds, typed attributes/tokens, parent/children/sibling links, ordered IDs/paths,
  raw spans and diagnostics: `ast.go`, `ast_parse.go`, `ast_search.go` and AST
  scenarios. `Node` and sibling links use direct indexes; iterative preorder
  walking supports pruning/stopping. Public inspection returns copied state.
- Document/line/inline, man/ms controls, explicit and opaque nested blocks,
  preprocessors, unknown/malformed input: named AST scenarios and parser recovery
  tests. Physical-line/raw-source cardinality is asserted independently.
- Unicode/rune columns and byte offsets, CRLF/no-final-LF/invalid UTF-8, extraction,
  literal/normalized searches, immutable token inspection, reader failures and
  depth/node/byte/line limits: focused `ast_test.go` assertions and fuzz seeds.
- Legacy ParseLine/ParseDocument/token APIs remain available. Narrow tab and
  unterminated-block repairs have positive/negative recovery tests. Full legacy
  fidelity/evaluation/include/generator contracts remain under #3–#17; see
  [specific dependencies](ast.md#parser-dependencies). Those issues were not closed.
- The separate #12 containment/lifecycle repair rejects escaping symlinks with
  `os.Root`, closes resolver-owned readers on success/failure, reports close errors,
  forwards per-source line limits, preserves nested diagnostics and bounds custom
  resolver recursion. Repeated siblings and resolver reuse release reservations.
  A probe against parent `bf43d21` reproduced both escaping reads and an unclosed
  reader; the same probe passed on the repair. Include TXTAR scenarios, OS boundary
  tests and focused race tests cover these contracts. This does not complete the
  rest of #12 or replace native Windows/macOS evidence.
- The separate #5 registry repair wires previously ignored custom-handler options
  through document/AST dispatch and inherited includes. Explicit registries isolate
  callers; registration snapshots keep active parses stable. TXTAR cases cover
  package/precedence, source vs. token formatting, custom errors and invalid or
  ambiguous results. Race tests cover registration/dispatch and isolated parsers;
  a subprocess verifies the legacy global API without polluting other tests. A
  pre/post probe changed both document/AST handler call counts from 0 (rejected)
  to 1 (accepted). Catalogue-wide implementation validation remains under #5.

### CLI and semantic model (#21–#22)

- `validate`, versioned JSON/text `ast`, selectors, package/strict options,
  file/stdin input, diagnostics and stable 0/1/2/3 exits: CLI TXTAR and injected
  stream/FS/error tests. Built-process tests cover real files, filenames with
  spaces/dashes, stdin, stdout, negative status, alias protection and atomic output.
- Help/validation explicitly say syntax acceptance does not prove typesetting.
  The legacy `cmd/lex` remains documented and tested.
- The semantic model retains metadata/provenance and groups physical flow into
  sections/subsections, paragraphs, definitions/items, indentation, preformatted
  blocks, breaks, styled runs, links/references and unsupported literal fallback.
- Common man macros, line-spanning flow, repeated headings, inherited state,
  section/AST/range/term selection, raw/normalized extraction, malformed input,
  budgets and invalid models have named scenarios and focused model tests.
- Semantic JSON round-trip/canonical comparison is tested separately from raw
  source. The realistic synthetic fixture is embedded/reused; no installed man
  database or evaluator is needed.

### HTML and conversions (#23–#24)

- Full/fragment/section/node rendering, title/themes/CSS/typography/fonts/assets,
  responsive semantic headings/lists/preformatted text and anchors: renderer and
  CLI scenarios, parsed DOM assertions and the actual HTML example.
- Source escaping, strict output buffering, safe URL policy, unsupported warnings,
  explicit custom fonts, output/writer/budget errors: positive and hostile tests.
  Neither rendering nor import runs a browser, script, include or resource fetch.
- HTML import uses x/net/html. Supported headings/flow/styles/lists/definitions/
  preformatted/links map to the man model; unsupported markup/CSS stays observable.
- Both semantic loops compare normalized intermediate models. Second canonical
  roff is stable. Integration also compares canonical HTML rendered from stable
  canonical roff, avoiding unrelated source-offset differences. Real exporter
  fragments/full bundled-font output pass through the importer.
- Control-line, argument, quote/backslash and Unicode escaping have independent
  safety assertions. Negative cases require retained content rather than allowing
  a successful conversion that discarded unknown nodes.

### Raster and desktop (#25–#26)

- Pure-Go font measurement/painting, flow/wrapping/styles/baselines, headings,
  definitions/indent/preformatted text, glyph fallback and diagnostics: layout and
  decoded-image tests, readable TXTAR layouts and separate PNG/JPEG goldens.
- Defaults bundle Go Regular/Bold/Italic/Mono and needed variants. Explicit custom
  paths are optional. Width/margin/font-size/DPI/theme/page-height/single-mode/
  pixel/page/glyph/output limits, JPEG quality, selectors, page streams/writers,
  numbered paths, stdout-one-page and staged per-file replacement are tested.
- Content/bounds/nonblank assertions supplement frozen bytes. Local visual checks
  covered full HTML, both example raster formats, pagination and a real viewer
  screenshot. Existing image/state goldens were unchanged by the ARM64 rounding
  repair; ARM64 representative image bytes match exactly.
  Current image-layout fuzzing also passed 149,126 executions over 15 seconds,
  including actual page painting after successful bounded layout.
- Viewer reuses the raster plan, with no alternate layout engine. Pure-Go model
  scenarios cover pan/scroll/drag/anchored zoom, fit/reset, page controls, actual
  viewport geometry, notes/help/close, invalid input and allocation bounds.
- Desktop dependencies are isolated in their own module. Linux amd64/X11/CGO=0
  actually opened full/selected sources, handled real input/resize/page changes,
  matched server pixels and closed cleanly, including an extracted release binary.
  Windows GUI is cross-built only; Apple requires its native toolchain and remains
  unverified. Only the verified Linux viewer is packaged.

### HTTP library (#27)

- Explicit roots, deterministic catalogue/duplicates/section suffixes, bounded
  pure-Go gzip, built-in name/description/text search and correct heading links:
  MapFS/TXTAR, httptest and executable HTTP tests.
- Optional apropos/whatis detection is injected and opt-in, with argument arrays,
  time/output limits, escaped results and visible fallback. No external utility
  is required for tests or normal internal search.
- Loopback default/explicit remote opt-in, GET/HEAD only, host/query/traversal
  rejection, escaped HTML/CSP, snapshot behavior, concurrency/response/source/
  decompression/depth/node limits, cancellation, reader closure and root symlinks
  have focused tests. No request opens an arbitrary source path or follows includes.
- The OS-root/symlink contract passed on Linux. Wine cannot validate it: its
  symlink API reported success while the link was absent from all subsequent
  file APIs. The native Windows lane retains the full test without exceptions.

### Shared scenarios and user documentation (#28)

- Recursive sorted fs.WalkDir discovery, standard TXTAR, explicit inputs/options,
  AST walks/extraction, validator statuses, normalized semantics, both loops,
  layout/diagnostics and binary decode are exercised in one harness. Missing input
  or unknown/duplicate members fail; missing diagnostics means none.
- Cases cover typical NAME/SYNOPSIS/DESCRIPTION/OPTIONS/EXAMPLES, repeated sections,
  controls/escaping/URLs/Unicode/CRLF/no-final-newline/empty/malformed input,
  inline formatting, preformatted pagination and unsupported literal retention.
- The synthetic fixture drives CLI, AST, validation, HTML, PNG/JPEG, conversion,
  in-memory server and viewer. Normal tests never update goldens. Explicit update
  controls, deterministic example checks/generation and actual outputs are documented.
- README is a user guide with honest unreleased installation instructions, commands,
  selection/fonts/pagination, conversion/server/search/viewer, support matrix and
  limitations. Marked fences execute as argv against the real binary; outputs are
  checked and images decoded. Library examples are executable Go examples.
- AST and HTML fuzz boundaries were executed; budgets and invalid-model checks are
  independent of golden comparisons. Root tests also ran under ARM64 QEMU and as
  Windows executables under Wine, with their scope identified below.

### Release and CI (#29)

- GoReleaser check and full snapshot builds passed for six headless targets and
  the separate Linux viewer. Default names/checksums, trimpath/CGO metadata,
  version/commit/date and required archive contents were inspected.
- Both Linux architectures produce deb/rpm/apk/Arch packages with executable,
  generated man pages/completions, README/licenses/example at defined paths.
  A Debian package actually installed and rendered HTML/PNG as nobody in a
  network-disabled container. RPM/APK/Arch contents were structurally inspected;
  their package-manager installs are not claimed.
- Clean clone tidy checks for both modules, generation drift, vet, unit/integration/
  examples, race, CGO-free builds and actionlint passed. Isolated non-snapshot
  prerelease builds passed with publication disabled, including installed-package
  and extracted-viewer execution. No remote release/tag was created.
- One workflow owns useful PR/main/tag/manual paths, with no duplicate feature
  pushes, schedule, automatic tags or auto-merge. PRs avoid the six-target matrix;
  full/main/tag routes require it. Source/generation/candidate/native gates are
  explicit and the always-run aggregate rejects failed/cancelled/skipped checks.
- Seventeen TXTAR cases execute the actual routing/gate shell blocks. Only the
  final publisher receives contents:write, is tag-serialized and bound to the
  release environment, and rechecks the immutable tag/main ancestry. GoReleaser
  only builds; the sole publisher uploads those same inspected artifacts.
- Before first publication, environment reviewer/tag protections require repository
  owner configuration. No protection policy or successful publication is inferred
  from the YAML. Full hosted/native acceptance is still missing.

## Platform evidence and remaining acceptance

| Surface | Linux amd64 | Linux arm64 | Windows amd64 | Windows arm64 | macOS amd64/arm64 |
| --- | --- | --- | --- | --- | --- |
| Headless archives | Built, inspected and executed | Built, inspected and QEMU-executed | Built, inspected and Wine-executed | Built/inspected | Built/inspected |
| Root scenarios/tests | Native pass | QEMU pass | Wine pass | No runtime evidence | No runtime evidence |
| Internal CLI tests | Native pass | Native gate pending | Wine pass | No runtime evidence | No runtime evidence |
| HTTP/root filesystem | Native pass | Native gate pending | Wine suite fails the symlink case; other tests pass in isolation | Native gate pending | Native gate pending |
| Viewer | Real X11 input/pixels/close | Cross-build only, not packaged | Cross-build only, not packaged | Not packaged | Not packaged |

Wine and QEMU are compatibility/emulation evidence. They do not replace native
OS/architecture results. Native Windows/macOS and the hosted release graph are
not established, so the complete release acceptance remains unproven.

Hosted runs #4, #5 and #6 failed before any steps ran. Run #6 failed source
verification and the aggregate with empty step lists/no log blobs; candidate,
native, viewer and publisher jobs were skipped. Check output summaries are empty.
The current connector rejects annotation URLs and the direct authenticated API
returns HTTP 403. The cause is unknown: billing, capacity, policy or test failure
is not inferred. A repository owner must inspect the job annotations and restore
execution before the full native verification can supply missing evidence.

See [release gates and commands](releasing.md), [scenario vocabulary](scenarios.md)
and the exact run/commit evidence recorded in each draft PR.
