# Catalogue provenance evidence

This is a factual checkpoint for [#67](https://github.com/arran4/golang-gtroff/issues/67)
and [#81](https://github.com/arran4/golang-gtroff/issues/81), reviewed on
2026-11-13 against `a5359412d0363ade8b7dd611e365a79412397b98` (#64).
It changes no license grant and provides no publication clearance. The
[relicensing assessment](relicensing.md) identifies the decisions still needed.

## What the evidence establishes

The catalogue has **366 description groups and 518 items**. The
[machine-readable comparison](catalogue.json) inventories every group, its
decoded-description SHA-256, input line, items and observed matches. It does
not contain additional copies of upstream description prose.

| Comparison | Groups matched |
| --- | ---: |
| groff 1.22.4 manual definition bodies, whitespace normalized | 276 |
| groff 1.23.0 manual definition bodies, whitespace normalized | 25 |
| Either upstream manual | 278 |
| Bundled `groff_man.txt`, contiguous whitespace-normalized text | 25 |
| Bundled `groff_ms.txt`, contiguous whitespace-normalized text | 12 |
| Bundled `groff.txt` | 0 |
| Any of the above | 313 |
| No match in this comparison | **53** |

These sets overlap. In particular, a match in a bundled render is not an
independent upstream-source or copyright-grant verification. The 53 unmatched
groups are precisely the rows with an empty `matches` array. Do not label them
GFDL-only or independently authored on the basis of a failed text comparison.

Whitespace normalization joins Unicode whitespace runs with one ASCII space.
HTML entities are decoded and the declared ISO-8859-1 encoding is honored.
Punctuation, escape spelling, footnote markers and words are retained; there
is no fuzzy matching or typographic substitution. Definition matches require
a roff `dt` heading whose next element sibling is a nonempty `dd`. The
reported HTML line is the heading anchor's physical line, not a rendered
line number. Reference matches are whole-description substrings and do not
establish correct token association. No byte-for-byte source identity is
claimed for either kind of normalized match.

`byte_identical_history` compares decoded description bytes, not JSON file
encoding or token membership. `whitespace_identical_history` records the
weaker reflow comparison separately. Hashes distinguish the complete input
JSON and individual decoded descriptions. Input labels, rather than local
absolute paths or timestamps, appear in the report. `revision` is the caller's
recorded review target; the tool does not independently verify a Git revision.

## Actual import history and transformations

1. [`df43c9a51aa0ecb9aef25ce833cbe040ae742efb`](https://github.com/arran4/golang-gtroff/commit/df43c9a51aa0ecb9aef25ce833cbe040ae742efb),
   2023-01-13, Arran Ubels: the commit message explicitly identifies
   `https://www.gnu.org/software/groff/manual/groff.html` and includes the
   browser JavaScript used to extract `dt`/next-element text, trim surrounding
   whitespace, replace nonbreaking spaces, and group entries. This is the
   broader GNU troff manual, not merely the three bundled man-page renders.
   The exact HTML revision served that day has not been recovered. The date
   alone does not identify a released version or its exact grant.
2. [`1938f96e3e73c17f9a222024a4d528658644a877`](https://github.com/arran4/golang-gtroff/commit/1938f96e3e73c17f9a222024a4d528658644a877),
   later that day, Arran Ubels: changes extraction/group ordering. The
   `reduceRight`/`unshift` grouping is material to the metadata defect below.
3. [`f44379d77df5b0c3a72ee1dd9a0f00c85b830dc6`](https://github.com/arran4/golang-gtroff/commit/f44379d77df5b0c3a72ee1dd9a0f00c85b830dc6),
   2025-11-19, Arran Ubels, “Description rework”: introduces the three
   formatted reference files and changes descriptions. All 366 current
   decoded descriptions are byte-identical to descriptions in this commit;
   316 also match `1938f96` after whitespace normalization, and 90 are
   byte-identical to descriptions in the initial import. Thus 50 current
   groups have descriptions added or changed beyond whitespace since
   `1938f96`. A commit author's name does not establish authorship of copied
   prose or permission to relicense it.
4. Subsequent generator-input commits `0a279a4`, `9168208` and `3e9fb48`
   declare argument grammars and block policies. The description comparison
   above verifies that their current prose remains identical to `f44379d`.
   The generator trims descriptions in `cmd/gen/schema.go:103`, emits runtime
   strings through `catalogue.go.template:38`, and emits documentation
   comments through `macros.go.template:13`. Removing only Go comments would
   leave the catalogue strings.

### Incorrect description and provenance associations

`cmd/gen/in.json:4188` starts zero-based group 356. Its pre-grohtml description
is assigned to **153** items, including unrelated `.warn`, `.sy` and `.cp`.
Those item keys intersect 104 groups in the initial JSON. This does not prove
the initial extractor's associations were correct: its next element could
itself be another `dt`, rather than a definition body. This is tracked as
[#99](https://github.com/arran4/golang-gtroff/issues/99), separately from the
rights assessment. This PR does not change generator input or parser behavior.

`cmd/gen/schema.go:128–151` assigns `Provenance` strings using kind and
container-index ranges. Values such as `groff_man(7)` are therefore package
or reference hints, not verified source citations. For example, the first
`.QS` entry gets the man label although its description concerns an ms
paragraph region. A successful prose match in this audit does not validate
the package, token association or argument grammar.

## Examined source artifacts and grants

The following official HTTPS source archives were downloaded for comparison.
Hashes identify the observed artifacts; a detached-signature verification was
not performed. Neither archive nor generated HTML is added to this repository.

| Artifact | SHA-256 |
| --- | --- |
| [groff-1.22.4.tar.gz](https://ftp.gnu.org/gnu/groff/groff-1.22.4.tar.gz) | `e78e7b4cb7dec310849004fa88847c44701e8d133b5d4c13057d876c1bad0293` |
| [groff-1.23.0.tar.gz](https://ftp.gnu.org/gnu/groff/groff-1.23.0.tar.gz) | `6b9757f592b7518b4902eb6af7e54570bdccba37a871fddb2d30ae3863511c13` |

| Source within artifact | Notice and credit evidence | Limit of the evidence |
| --- | --- | --- |
| 1.22.4 `doc/groff.texi:26–43` | FSF copyright 1994–2018; FDL 1.3 or later, no invariant sections, specified front and back cover texts | Its cover-text terms differ from the 1.23.0 notice. Do not replace this with a generic “no cover texts” attribution. |
| 1.22.4 `LICENSES:10–28` | GPL v3 or later for the program; wording allows files under GPL **or compatible licenses** | This wording is not the later explicit all-files GPL grant. An exact text match in this edition alone is not recorded as GPL clearance. |
| 1.23.0 `doc/groff.texi:25–39` | FSF copyright 1994–2023; FDL 1.3 or later without invariant sections or cover texts | Applies to this source edition; does not identify every older imported passage. |
| 1.23.0 `LICENSES:10–29` | All files in the distribution offered under GPL v3 or later, with alternative terms possible; FSF assignments subject to stated exceptions | Supports the GPL pathway for passages actually covered by this distribution. Does not authorize MIT/BSD or prove all historical text falls within it. |
| 1.23.0 `doc/groff.texi:468–470,815–820` | Authors Trent A. Fisher, Werner Lemberg and G. Branden Robinson; credits adaptations from James Clark's man pages, Eric Allman's me introduction and Larry Kollar's ms material | Credits are source evidence, not a transfer of these authors' rights to this repository. |
| 1.23.0 `tmac/groff_man.7.man.in:60–84` | FSF copyright 1999–2018, 2020–2021; source-specific verbatim/modified/translated-manual notice | Do not describe this notice as BSD/MIT or simply FDL. Preserve its text if relying on it. |
| 1.23.0 `tmac/groff_ms.7.man:10–33` | FSF copyright 1989–2023; corresponding manual-copying notice | As above; archive-wide GPL alternative is separate evidence. |

Upstream later clarified the distribution statement in
[commit a083c87](https://www.mail-archive.com/groff-commit@gnu.org/msg13204.html)
and made the manual's dual licensing explicit in
[commit ea590458](https://www.mail-archive.com/groff-commit@gnu.org/msg14955.html).
The [man-page notice discussion](https://www.mail-archive.com/groff-commit@gnu.org/msg13201.html)
also distinguishes these terms. Those clarifications are useful grant evidence,
but do not themselves match the remaining historical passages.

`groff_man.txt` reports 1.23.0 with a **31 October 2025** footer;
`groff_ms.txt` and `groff.txt` report 1.23.0 with **2 July 2023** footers.
Consequently the first render cannot be assumed byte-identical to the July
2023 release sources simply because it shares the version number. Exact
source revisions, transformations and retained notices for the three full
renders still require verification. The bundled renders are comparison
evidence here, not authority for reassigning copyright terms.

## Reproduce the catalogue comparison

Use a separate evidence directory and obtain the two hash-checked archives
above. Extract them there. The 1.23.0 archive supplies `doc/groff.html`.
For 1.22.4, run GNU Texinfo **7.1.1** against its extracted source:

```sh
makeinfo --html --no-split --force \
  --output=groff-1.22.4/doc/groff.html groff-1.22.4/doc/groff.texi
```

The observed conversion succeeded with five warnings (macro argument commas
and redundant `@noindent`). It honors ISO-8859-1. Another Texinfo version may
change markup, anchors, line numbers and the HTML hash. Record such a rerun as
new evidence rather than silently substituting its bytes for this checkpoint.
The observed HTML SHA-256 values are recorded in `catalogue.json`.

From the reviewed repository checkout, with `PROVENANCE_DIR` set to that
explicit evidence directory:

```sh
git show df43c9a51aa0ecb9aef25ce833cbe040ae742efb:cmd/gen/in.json > "$PROVENANCE_DIR/df43c9a.json"
git show 1938f96e3e73c17f9a222024a4d528658644a877:cmd/gen/in.json > "$PROVENANCE_DIR/1938f96.json"
git show f44379d77df5b0c3a72ee1dd9a0f00c85b830dc6:cmd/gen/in.json > "$PROVENANCE_DIR/f44379d.json"
go run ./internal/provenanceaudit \
  -revision a5359412d0363ade8b7dd611e365a79412397b98 \
  -manual "groff-1.22.4=$PROVENANCE_DIR/groff-1.22.4/doc/groff.html" \
  -manual "groff-1.23.0=$PROVENANCE_DIR/groff-1.23.0/doc/groff.html" \
  -reference groff_man.txt=groff_man.txt \
  -reference groff_ms.txt=groff_ms.txt -reference groff.txt=groff.txt \
  -history "df43c9a51aa0ecb9aef25ce833cbe040ae742efb=$PROVENANCE_DIR/df43c9a.json" \
  -history "1938f96e3e73c17f9a222024a4d528658644a877=$PROVENANCE_DIR/1938f96.json" \
  -history "f44379d77df5b0c3a72ee1dd9a0f00c85b830dc6=$PROVENANCE_DIR/f44379d.json" \
  > "$PROVENANCE_DIR/catalogue-rerun.json"
cmp docs/provenance/catalogue.json "$PROVENANCE_DIR/catalogue-rerun.json"
```

The comparison tool is a verification-only program with no network access or
generator writes. It uses the already pinned Go HTML dependency, requires
explicit evidence inputs, bounds each file to 32 MiB and writes one JSON report
only after all inputs validate. Its tests cover encoding/entities, alias
boundaries, punctuation differences, history versus reflow, deterministic
output and invalid/missing/duplicate evidence. It is not a legal decision
engine or part of ordinary parsing. GNU Texinfo is needed only to reproduce
the older comparison artifact, never for library/CLI operation.

## Remaining decisions and runnable follow-up

- Recover or otherwise establish covered-source evidence for the 53 unmatched
  descriptions and the 1.22.4-only passages; match actual words, not merely
  request names. Preserve per-edition notices and distinguish unchanged text
  from adaptations. The live 2023 HTML revision remains unidentified.
- Establish the exact source revisions and complete attribution/grants of
  the three formatted references, especially the 2025-dated man render.
- Have the maintainer review the evidence supporting retention under the
  existing mixed GPL policy. If a grant remains unestablished, choose a
  separately reviewed permission, independent rewrite or real separation;
  do not mass-rewrite prose merely because a normalized match failed.
- Carry the accepted notices and modification dates into generator inputs,
  generated distribution notices and source/binary packages through the
  documented generation/release process. This checkpoint does not assert
  that the current notices satisfy every selected alternative.
- Repair the independently tracked metadata associations in #99. Re-run the
  inventory after any catalogue changes and on the final planned release HEAD.
- Keep #67 open. Full reachable-history/artifact checks under
  [#75](https://github.com/arran4/golang-gtroff/issues/75) and release acceptance
  under [#29](https://github.com/arran4/golang-gtroff/issues/29) remain separate.

This checkpoint establishes neither an actual violation nor absence of one.
