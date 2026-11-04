# Scope, limitations and release status

**Work in progress:** gtroff implements a subset of roff and Linux man-page behaviour. It is **not** a full GNU groff clone, nor a stable substitute for the GNU formatter.

## Current merged functionality

| Area | Scope |
| --- | --- |
| Source parser | Typed requests and man/ms macros, source positions and diagnostics |
| AST | Original source and line endings, traversal, search and selection |
| CLI | `validate`, `ast`, `render`, `convert`, `help` and `version` |
| Semantic man | Supported headings, paragraphs, tags and preformatted text |
| HTML export | Standalone HTML for the documented semantic subset |
| Conversion | Semantic HTML/man profile, not arbitrary HTML or lossless conversion |
| Build | Pure-Go headless CLI, no system groff or CGO |

The capabilities available on your installation depend on the specific revision. Check the [default branch](https://github.com/arran4/golang-gtroff) and its open PRs.

## What it does not guarantee

- Complete GNU groff syntax, layout, macro packages, preprocessor or drawing equivalence.
- Executing arbitrary input or automatically opening source include paths during AST inspection.
- Exact source fidelity after an HTML semantic round trip.
- Stable Go APIs, CLI output formats or binary release support during active development.
- Native GUI support on platforms not tested with a real application session.

## In-progress integrations

PNG/JPEG rendering, man-library HTTP browsing, the optional Shiny viewer, generated man pages/completions and release packaging were developed in stacked feature branches. **These are not all automatically part of `main` merely because their individual PRs show merged.** See [integration PR #62](https://github.com/arran4/golang-gtroff/pull/62) for their proposed combined entry into main, and [issue #31](https://github.com/arran4/golang-gtroff/issues/31) for release criteria.

Linux/X11 native interaction is the initial GUI acceptance baseline; Windows/macOS remain preview/unverified.

## Public source versus a released product

Public Git history, a GitHub Pages documentation site and downloadable stable binaries are distinct milestones. Full-history leak scanning, attribution/licensing review, native validation and release gates are tracked separately.

See [getting started](getting-started.md), [workflows](workflows.md) and [documentation deployment](documentation-site.md).
