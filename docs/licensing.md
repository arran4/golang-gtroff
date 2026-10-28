# Licensing and Provenance

`golang-gtroff` is free and open-source software. This document summarizes the copyright, licensing terms, and provenance of the repository's components.

## Core Codebase

- **License**: [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.html) (GPL-3.0-only or later).
- **Details**: Full text is provided in [LICENSE](../LICENSE).
- All Go packages, CLI commands, tests, and build tooling developed as part of `golang-gtroff` are governed by this license.

## GNU groff Documentation and Catalogue Derivations

Portions of the macro and request documentation embedded in `cmd/gen/in.json` and generated into `catalogue.go`, `macros.go`, and parser docstrings are derived from the official [GNU groff manual](https://www.gnu.org/software/groff/manual/groff.html#Copying-This-Manual).

- **Original Authors**: Trent A. Fisher, Werner Lemberg, G. Branden Robinson, James Clark, Larry Kollar, and GNU groff contributors.
- **License**: [GNU Free Documentation License](https://www.gnu.org/licenses/fdl-1.3.html) (GFDL v1.3 or later), with no Invariant Sections, no Front-Cover Texts, and no Back-Cover Texts.
- The reference text files in the repository root (`groff.txt`, `groff_man.txt`, `groff_ms.txt`) are plain-text renderings of GNU groff 1.23.0 manual pages (`groff(7)`, `groff_man(7)`, `groff_ms(7)`) included for offline verification and behavioral conformance reference.

## Bundled Go Fonts

The `fontset` package and standalone rendered HTML documents bundle or embed the Go font family (Go Regular, Go Bold, Go Italic, Go Mono):

- **Copyright**: © 2016 Bigelow & Holmes Inc. All rights reserved.
- **License**: 3-Clause BSD License.
- **Notice**: Preserved in [fontset/LICENSE-Go-Fonts.txt](../fontset/LICENSE-Go-Fonts.txt) and exposed programmatically via `fontset.License()`.

## Test Fixtures and Snippets

- `testdata/environ_snippet.7` is adapted from the Linux `environ(7)` manual page (Copyright © 1993 Michael Haardt, Andries Brouwer; licensed under GPLv2 or later).
- All other test scenarios and TXTAR fixtures under `testdata/` are authored for `golang-gtroff` test verification and released under the repository's GPLv3 license.
