# Licensing and Provenance

> **Publication review:** This records the maintainer's mixed-license choice; it is not by itself a completed copyright-source audit or approval to make the repository public. The remaining checks are tracked by [#67](https://github.com/arran4/golang-gtroff/issues/67).

The license of a component is determined by its own copyright grant, not merely by its location in this repository. No third-party material is relicensed by this project.

| Component | License / distribution terms | Provenance |
| --- | --- | --- |
| Independently authored Go implementation, CLI, tooling and project-authored tests | **GPL-3.0-or-later** | [Root GPL license](../LICENSE) |
| GNU groff-derived macro definitions and descriptions in `cmd/gen/in.json`, `catalogue.go` and generated Go comments | Use **GPL-3.0-or-later only for passages covered by GNU groff's upstream GPL grant or explicit dual-license permission**; retain copyright attribution. Any unmatched GFDL-only passages require separation, replacement or permission before public redistribution as part of compiled Go. | GNU groff man-page and GNU troff manual sources, detailed below |
| Independent GNU groff reference documentation (`groff.txt`, `groff_man.txt`, `groff_ms.txt`) | Original source-specific GNU license terms, including **GFDL-1.3-or-later** where applicable, and any upstream GPL or permissive alternative | GNU groff 1.23.0 reference renders |
| Bundled Go Fonts and independently exported font data | **BSD-3-Clause** | [Go Fonts license notice](../fontset/LICENSE-Go-Fonts.txt), `fontset.License()` |
| `testdata/environ_snippet.7` | **GPL-2.0-or-later** | Adapted Linux `environ(7)` manual page, © 1993 Michael Haardt and Andries Brouwer |

## GNU upstream licensing and generated code

The catalogue contains prose derived from the GNU groff 1.23.0 man-page reference material. The upstream GNU groff project's `LICENSES` statement grants GPLv3-or-later for its distribution, and explicitly provides an FDL/GPL dual-license pathway for the GNU troff Texinfo manual. This is **permission from upstream**, not a purported relicensing by this project. The original GNU manual is also available under GFDL 1.3 or later (without invariant sections or cover texts in the cited modern edition).

- [Upstream GNU groff licensing statement and GPL clarification](https://www.mail-archive.com/groff-commit@gnu.org/msg13204.html)
- [Upstream dual-license clarification for manual material](https://www.mail-archive.com/groff-commit@gnu.org/msg14955.html)
- [GNU groff's man-page licensing discussion](https://www.mail-archive.com/groff-commit@gnu.org/msg13201.html)
- [GNU troff manual and its copying notice](https://www.gnu.org/software/groff/manual/groff.html#Copying-This-Manual)
- [GFDL v1.3 full license text](../LICENSES/GFDL-1.3.txt)

**Source verification still required before publication:** Confirm the origin and actual license grant for the substantial descriptions copied into `cmd/gen/in.json` and then into generated Go. Match the copied text against appropriately licensed upstream versions; record source version, authors, copyright notice and transformations. Do not infer that the presence of a newer dual-license declaration automatically covers every historical or third-party passage. If a passage is not covered by GPL-compatible permission, replace it with independently authored factual documentation or distribute it as genuinely separate GFDL reference material, **not** embedded into generated Go or binaries. Regenerate and run tests only if changing the catalogue source.

GNU groff documentation credits include Trent A. Fisher, Werner Lemberg, G. Branden Robinson, James Clark, Larry Kollar, and contributors. Source-specific credits and the actual upstream notices take precedence over this summary. The three bundled `.txt` renders are reference material, not project-authored Go code; preserve their provenance and applicable permissions.

## License boundaries

The root `LICENSE` is an unmodified copy of GPLv3. The project's own code is offered under **GPL-3.0-or-later**; the full text itself need not be edited to express the “or later” grant. Third-party components retain the terms stated above. A generated source file or compiled binary containing copyrighted reference prose is not automatically a separate aggregate merely because the original prose also exists in a reference document.

Do not infer publication clearance, final secret-scanning completion, or permission to enable GitHub Pages solely from this license inventory.
