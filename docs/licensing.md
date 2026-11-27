# Licensing and attribution

Project-authored Go code, CLI, tools and tests are licensed under **GPL-3.0-or-later**. Third-party material retains its own terms; this project does not relicense it.

| Component | Terms and notices |
| --- | --- |
| Project-authored code | [GPL version 3](../LICENSE), or later |
| GNU-derived descriptions in `cmd/gen/in.json`, `catalogue.go` and generated Go comments | GPL-3.0-or-later where covered by the upstream grant or dual licence; not all historical passages have established source/permission coverage |
| `groff.txt`, `groff_man.txt`, `groff_ms.txt` | Original GNU manual notices and applicable grants; [GFDL 1.3](../LICENSES/GFDL-1.3.txt) where applicable |
| Bundled Go Fonts | [BSD-3-Clause](../fontset/LICENSE-Go-Fonts.txt) |
| Adapted Go Image rasterizer code | [BSD-3-Clause and modification attribution](../renderimage/LICENSE-Go-Image.txt) |
| `testdata/environ_snippet.7` | GPL-2.0-or-later; adapted from Linux environ(7), © 1993 Michael Haardt and Andries Brouwer |

## GNU groff material

The catalogue includes text extracted from the GNU troff HTML manual and later adapted descriptions. The exact initial manual edition is not established. The three formatted references identify GNU groff 1.23.0; `groff_man.txt` has a 31 October 2025 footer and the other two have 2 July 2023 footers.

The GNU troff manual credits Trent A. Fisher, Werner Lemberg, G. Branden Robinson and contributors, including adaptations from James Clark, Eric Allman and Larry Kollar. GNU groff 1.22.4's manual is © 1994–2018 Free Software Foundation, Inc., under GFDL 1.3 or later with no invariant sections and specified front/back cover texts. The 1.23.0 manual is © 1994–2023 Free Software Foundation, Inc., under GFDL 1.3 or later with no invariant sections or cover texts. The man-page sources carry their own copying notices; consult the original source-specific terms.

The [GNU groff 1.23.0 source distribution](https://ftp.gnu.org/gnu/groff/groff-1.23.0.tar.gz) offers its files under GPLv3 or later with alternative terms possible. See the [upstream licence clarification](https://www.mail-archive.com/groff-commit@gnu.org/msg13204.html), [manual dual-licensing clarification](https://www.mail-archive.com/groff-commit@gnu.org/msg14955.html), [man-page copying notice discussion](https://www.mail-archive.com/groff-commit@gnu.org/msg13201.html) and [manual copying notice](https://www.gnu.org/software/groff/manual/groff.html#Copying-This-Manual). Source-specific copyright and licence notices take precedence over this summary.

The source and permission coverage of all descriptions embedded in generated Go and binaries has not been established. A text match alone is not permission to distribute under another licence. No MIT/BSD grant is made for the library as a whole. See [licensing issue #67](https://github.com/arran4/golang-gtroff/issues/67) for unresolved source rights.
