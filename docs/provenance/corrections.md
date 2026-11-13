# Catalogue association correction

This records the correction for [#99](https://github.com/arran4/golang-gtroff/issues/99)
against #100, `0d4064b04f760a556cb34929249aa8e6ec85d5a1`. It repairs feature
metadata and generated comments. It does not decide the licensing questions in
[#67](https://github.com/arran4/golang-gtroff/issues/67) or
[#81](https://github.com/arran4/golang-gtroff/issues/81).

## Defect and correction

The historical extraction assigned group 356's pre-grohtml description to 153
items. Only the first two concern that feature. It also stored the next request
or escape heading as the description in 24 other groups. Package inference by
group position mislabelled 26 items, including core registers and the first
`.QS` entry.

The input retains all **366 groups and 518 entries**, in their existing order.
Kinds, names, arguments, argument grammars, block policies, indices, Go names
and declared support status are preserved. There are **179 changed entries**:
175 description corrections and 26 package corrections, with overlap.

| Description correction | Entries |
| --- | ---: |
| Reference an existing description literal | 166 |
| New, project-authored factual summary literal | 7 |
| Reference one of those summaries | 2 |

The seven new summaries concern relative register assignment, device control
(`.device`/`.devicem`), string definition, numeric font selection, adjusted line
breaking, and plain register assignment. They replace lost bodies or wrong
argument-variant associations; no new upstream prose is imported. The signed
`.nr` forms share one summary. `.ds` and `.ds1` share another. `.ft nnn` does
not reuse the named-font description, and plain `.nr` does not claim that an
increment parameter was supplied.

The [change inventory](association-corrections.json) records every affected
entry, previous/current decoded-description SHA-256, effective JSON pointer,
package changes, and **288 source-heading citations**. Source labels and HTML
hashes identify the same explicitly downloaded editions described in the
[historical provenance checkpoint](README.md). This inventory was assembled by
reviewing definition headings, argument forms, shared bodies and feature
families. Its evidence bases distinguish:

- `whitespace_identical_definition_body`: the selected heading's body matches
  the reused literal under the checkpoint's whitespace normalization.
- `shared_definition_body`: the selected heading and the existing literal's
  feature heading share one `dd` in that edition. This supports the association;
  it does not assert identical prose. Consecutive `dt` headings must be collected
  through their following shared `dd`, including escape headings excluded from
  the catalogue. Looking only at `nextElementSibling` loses these relationships.
- `feature_heading`: the cited heading identifies the corrected feature family
  or the feature summarized in new project prose. This is not a text-match claim.

Argument forms and package context disambiguate `.DS` placements and `.Ds`.
For example, the bounding-box registers and `.psbb` share a definition body;
`.NH curr-level` and the alternate `.NH` heading share another. Package names
on core requests identify roff even when a tutorial section discusses ms.
These semantic associations do not establish the exact HTML served in 2023,
original authorship, copyright ownership or permission to relicense.

## Input schema and compatibility

A group owns `Description` by default. An item may own its own `Description`
or select `DescriptionRef`. A group can also use `DescriptionRef`. Literals
may have stable `ID` values. References must point directly to a nonempty
literal: duplicate/invalid IDs, unresolved references, reference-plus-literal,
reference-plus-ID, chains and cycles are errors. IDs are explicit names, not
indices inferred at runtime. The same internal resolver serves generation and
provenance verification. It retains literal bytes and the owner's JSON position;
the generator still trims outer whitespace when emitting runtime descriptions.

An item's optional `Package` overrides the legacy group-position inference.
It identifies the documented feature family. Optional `ParserPackage` preserves
an existing constructor namespace where required. Correcting the first `.QS`
from man to ms must not collapse `TokenManQS` and `TokenMsQS`, remove aliases,
or change `ParseLineWithPackage` dispatch. Consequently both catalogue entries
now describe ms, while their historical parser bindings and token
`PackageName()` results remain available. Source-package collision metadata is
computed separately from constructor variants. The syntax support report exposes
`parser_package` only when its executable context differs from `package`.

`CatalogueDefinition.Provenance` still contains reference-family hints such as
`groff(7)`, not verified passage citations. The explicit overrides repair the
identified 26 errors; they do not replace the entire legacy inference scheme
with an authoritative source-location database.

Regeneration changes `catalogue.go`, comments in `macros.go` and package metadata
in the syntax support report. `parseline.go` remains byte-identical. A Go syntax
comparison against the parent, excluding comments, verified all declarations,
aliases, methods and dispatch code in both generated parser files unchanged.
Repeated identical shared descriptions are emitted once per concrete token.

## Effective-description provenance

[`catalogue-corrections.json`](catalogue-corrections.json) is schema **2**:
one row per effective entry, with its literal owner's line and JSON pointer.
Unused group defaults are not attributed to overriding items. It records 518
entries, **464 text matches and 54 unmatched entries**, including the nine entries
using the seven new summaries. The remaining 45 source-derived unmatched entries
still require source/grant research. Matches can overlap: 426 entries match the
1.22.4 manual, 27 the 1.23.0 manual, 25 the bundled man render and 13 the bundled
ms render; none match the bundled groff render.

These are entry counts, unlike the historical schema 1's group counts. They
must not be compared as a measure of licensing clearance. The comparator retains
its conservative whitespace-only text matching and direct `dt`/`dd` rule;
the separately reviewed association inventory supplies shared-heading evidence.
The frozen historical `catalogue.json` is reproduced using the explicitly
exported #64 input, rather than today's changed default input.

After preparing `PROVENANCE_DIR` and historical JSON inputs as documented in
[the checkpoint reproduction](README.md#reproduce-the-catalogue-comparison), run
from this checkout:

```sh
go run ./internal/provenanceaudit \
  -revision input-blob:4a84768f7e433d9473194b4fdaf19c8b098a16eb \
  -manual "groff-1.22.4=$PROVENANCE_DIR/groff-1.22.4/doc/groff.html" \
  -manual "groff-1.23.0=$PROVENANCE_DIR/groff-1.23.0/doc/groff.html" \
  -reference groff_man.txt=groff_man.txt \
  -reference groff_ms.txt=groff_ms.txt -reference groff.txt=groff.txt \
  -history "df43c9a51aa0ecb9aef25ce833cbe040ae742efb=$PROVENANCE_DIR/df43c9a.json" \
  -history "1938f96e3e73c17f9a222024a4d528658644a877=$PROVENANCE_DIR/1938f96.json" \
  -history "f44379d77df5b0c3a72ee1dd9a0f00c85b830dc6=$PROVENANCE_DIR/f44379d.json" \
  > "$PROVENANCE_DIR/catalogue-corrections-rerun.json"
cmp docs/provenance/catalogue-corrections.json "$PROVENANCE_DIR/catalogue-corrections-rerun.json"
git hash-object cmd/gen/in.json
```

The input-blob label avoids a self-referential commit identifier. The expected
Git blob is `4a84768f7e433d9473194b4fdaf19c8b098a16eb`; the report also records the
input's SHA-256. The tool records the caller's revision label without verifying
it. A different input, source edition or HTML rendering requires a new receipt.
The source headings in the change inventory are independently reviewable at
their recorded physical lines and anchors; ordinary tests check inventory drift
against the current input, not the legal meaning of external source text.
