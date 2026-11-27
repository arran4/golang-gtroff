# Library licensing feasibility

Factual assessment for [#81](https://github.com/arran4/golang-gtroff/issues/81),
2026-11-13. MIT/BSD is the maintainer's preference, not a grant already made.
This document requests no license replacement. The existing GPL-3.0-or-later
declaration and all third-party terms remain in force.

## Rights and import boundaries

The examined integration baseline is #64 at
`a5359412d0363ade8b7dd611e365a79412397b98`. New workbench APIs were also
inspected on their existing feature heads, without merging them:

| Component / reviewed feature head | Evidence and rights question |
| --- | --- |
| Root parser, source AST, registry, tokens and semantic `man` package; #78 at `2c5e34d18bbef6f564da7ff43c2abc290e048f77` | Project-authored implementation candidates under the existing GPL grant. Root package also contains GNU-derived generated catalogue strings and token comments. Ownership confirmation and catalogue resolution are prerequisites to a permissive root package. |
| `source`, #82 at `78e19455e1e38713de7cd0966d34d30aba5ba53e`; `query`, #68 at `6519677ae833293cb5471cc6eeb5823209e969c9` | Project-authored candidates that import the root AST/parser. Moving directories or adding a license to these packages would not remove that dependency. |
| `analyze`, #73 at `79067ff69053ef10c7d1c6a6b4d921d1b539d2af`; `lint`, #70 at `581f37e3dbf677e2666b5f46aa788be7a74ff52a` | Classification consumes catalogue/AST data; lint consumes analyze/source/semantic models. Same unresolved root boundary. |
| `extract`, #69 at `1f56ebb12bbf9e6b01e18ed9072bfb0837b317b7`; `diff`, #71 at `f953017135f495fd4dd2a2f5ec1a1088d1eb2714` | Project-authored code candidates, with query/man/source imports. Output can reproduce input prose; its rights do not automatically become the program's license. |
| `edit`, #72 at `35155fdaed4008a26b10e6b786da8b68a65241ad`; `author`, #79 at `95145863b263f83f2aec95764aec1e076db86e50` | Project-authored candidates that depend on root types/serialization. Independently authored synthetic templates differ from third-party source supplied by callers. |
| `scriptapi`, optional `script/` module, #74/#92 at `3dc1d364847f19a8cac0340eb590a71fe246fbac` | Adapter/session code is project-authored candidate code. Tengo v2.17.0 is MIT, copyright Daniel Kang; preserve its notice. Optional module imports the GPL root. Engine isolation is a dependency/size boundary, not a licensing exception. |
| `agentskill`, #80 at `b7e899933cea2e42eb6aee770929b85c655580ab` | Portable recipe text/templates are project-authored candidates. Export currently carries a verbatim GPL text and manifest hashes; a later authorized change would need source-template and manifest regeneration. No implicit relicensing by export. |
| `cmd/gen/in.json`, generated `catalogue.go`, `macros.go` | GNU prose and project-authored schema/generator code have different provenance. See the complete [description inventory](catalogue.json) and [source evidence](README.md). GPL permission does not confer MIT/BSD permission. Root compile-time data includes descriptions; removing only comments or reference files is insufficient. |
| `groff.txt`, `groff_man.txt`, `groff_ms.txt` | Separate reference files, still requiring exact source/notice verification. A separate file is not sufficient separation if its prose is also compiled into Go. |
| `fontset/`, `renderimage/stable_face.go` | Bundled Go Fonts and adapted Go Image implementation retain BSD-3-Clause notices in their respective directories. Project-authored wrappers/layout code remain under the project grant pending owner approval. |
| `testdata/environ_snippet.7` and any derived expectations | Actual GPL-2.0-or-later notice credits Michael Haardt and Andries Brouwer and retains modification history. Keep that notice and applicable terms; do not label the whole test corpus MIT/BSD. |
| CLI, conversion/rendering/viewer helpers, docs and synthetic tests | Project-authored candidates subject to contributor/import review. A GPL CLI remains an implementable route for existing code. Binary distributions must also include notices for embedded third-party content/dependencies. |
| `docs/themes/book` gitlink `b7f9c8cb0f51523f316777a27abba64b10e247f5` | [Hugo Book license at this exact revision](https://github.com/alex-shpak/hugo-book/blob/b7f9c8cb0f51523f316777a27abba64b10e247f5/LICENSE): MIT, copyright 2018 Alex Shpak. Theme assets/templates keep their notice when distributed; site output requires its own artifact review. |

These are ownership candidates, not findings that no external expression
exists in the remaining files. No assignment or contributor permission to
relicense was located. Git shortlog at the baseline has Arran Ubels, Codex and
arran4 identities (103, 47 and 3 author records). Feature history also uses
these names. The shared emails/account are not proof of one legal rights
holder, assignment, employer consent or the provenance of generated/assisted
code. The maintainer must confirm authority for the included implementation,
templates, examples, documentation and test contributions, or obtain consent
from other holders. Existing third-party grants are independent of that
confirmation.

The root license was introduced in `9d5f2fd` in 2022 as GPLv3; subsequent
`4df41ef` and `99a7636` altered its declaration/presentation. This assessment
found no historical MIT/BSD project grant to substitute for approval. It does
not determine legal ownership from commit metadata. Existing recipients'
rights to previously GPL-licensed versions must not be represented as revoked.

## Dependencies and generated/distributed material

The [module evidence inventory](modules.json) records all **22 unique module
versions** in the examined root/desktop/script graphs and SHA-256 hashes of
their observed top-level notices. The desktop graph also includes the two
deprecated x/tools test-helper modules. A module graph is broader than a
binary's actual imports, and this inventory is not a per-file binary SBOM.

Observed notices are BSD-3-Clause for go-subcommand, strings2, go-cmp, the Go
x/ modules, XGB, Metal and the Go GLFW wrapper; MIT for Tengo and Goldmark.
GLFW also bundles its own Zlib notice at `glfw/LICENSE.md`, separately recorded.
Keep component-level notices and adapted-code credits, not just the top-level
module license. No GPL dependency was established by this module-notice check;
the GNU catalogue and project grant remain the significant copyleft sources
in the library. A full file/artifact review can reveal additional notices.

On the baseline, `go list -deps .` finds no external nonstandard module for
the root parser alone. That fact does not remove the GNU data already in the
root package. The optional script's actual imports include Tengo and image,
net, sys and text modules; it still imports root code. Tests/tools and
platform-specific desktop dependencies have separate build scopes.

To reproduce the inventory, run `go list -m -json all` in root, desktop and
the stated optional-script checkout. Fetch missing module archives into Go's
cache for inspection; inspect/hash the named license paths for each exact
version. `go mod download all` can add unused checksum entries to `go.sum`:
perform that inspection in an isolated checkout and do not commit incidental
manifest changes. An absent `Dir` field is not a license finding; do not treat
the current directory as a missing module's source directory. Confirm actual
binary imports separately using `go list -deps` for its target/build tags.

An authorized license change would need a scoped review of README/GoDoc,
package notices, generator input and templates, generated Go/support reports,
portable-skill payload/manifest, examples/derived fixture expectations,
desktop/script module distributions, Hugo sources/output, archive/nfpm
contents, embedded font notices and source offers. `go.mod` has no field that
can grant a new license. Do not hand-edit generated files or replace full
license documents with an SPDX label.

## Options and required authorization

| Option | Technical feasibility and conditions |
| --- | --- |
| Repository-wide MIT or BSD | Not established. Requires authority for all included project contributions and removal/replacement or independently sufficient permissive permission for GNU prose. Existing reference/font/fixture/dependency exceptions must keep their own notices; “repository-wide” cannot erase them. |
| Owner-controlled code dual licensed GPL and MIT/BSD | Feasible for identified owner-controlled files after explicit approval. Does not cover GNU descriptions or other holders. Record exact covered files and exceptions, retain third-party terms, regenerate affected declarations. No practical permissive root import while it still contains unresolved copyleft data. |
| Permissive parser/AST/query core, separate GNU data | Recommended direction **if** permissive embedding is selected. Requires verified contributor authority and a real code/data boundary: replace or move the compiled prose, decouple root initialization/metadata and review every importing package. Keep syntax facts distinct from protected expression without presuming all generated material is uncopyrightable. Verify builds/imports and exported output after separation. This is a prospective design proposal, not an implemented or legally approved split. |
| Retain current GPL library and CLI | Smallest change under the existing maintainer decision, subject to #67 and notice/release acceptance. The CLI already provides documented formats; independent callers can invoke it as a process. Whether a particular combined product is a derivative work is a legal question; no blanket process-boundary exemption is promised. |
| LGPL fallback | Needs the same authority over project code and resolution of GNU material that cannot be offered under LGPL. GPL permission alone does not authorize a weaker LGPL grant. LGPL also introduces real downstream compliance work for static Go distributions; it is not a license-header-only alternative. |

For an LGPLv3 design, [section 4](https://www.gnu.org/licenses/lgpl-3.0.html)
requires notices/license copies and either a suitable shared-library mechanism
or materials/terms allowing recombination with a modified library; installation
information can also apply. Ordinary static Go imports do not by themselves
provide such a user relinking workflow. Before selecting this option, validate
a supported, reproducible application/library rebuild or relink route and have
its source/object distribution and user rights reviewed. Cgo/plugin tricks
would conflict with this project's ordinary pure-Go/headless direction and
must not be silently imposed. This is an engineering compliance question for
the selected license version, not a claim that LGPL is inherently impossible
with Go.

## Decision packet

Before any license-implementation PR, the maintainer needs to choose a license
and exact scope, confirm the rights above, and accept a source/notice strategy
for GNU material based on #67. If permissive embedding is chosen, a separate
technical proposal should define the smallest real parser/data boundary and
its compatibility tests; independently authored replacement descriptions need
reviewable source notes. Seek a qualified rights review where the grants or
ownership remain uncertain. No outside rights holder has been contacted and
no assignment or permission has been inferred.

This assessment leaves #81 open for that decision. It does not block unrelated
technical work, authorize relicensing, resolve #67 or sign off a release.
