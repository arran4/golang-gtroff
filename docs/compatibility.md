# Scope and compatibility

gtroff is a pure-Go toolkit for a subset of roff and Linux man pages. It is not a complete GNU groff implementation. APIs and CLI formats may change before a stable release.

| Area | Supported use |
| --- | --- |
| Parsing | Typed requests and man/ms macros, original source, AST and diagnostics |
| Inspection | Bounded source sets, structural queries, extraction, feature analysis and lint |
| Semantic model | Common man headings, paragraphs, lists, definitions and preformatted text |
| Output | HTML, paginated PNG/JPEG, canonical man source and a supported HTML/man conversion subset |
| Reading | Read-only local man-page HTTP library; optional desktop viewer |
| Authoring | Construct new canonical man pages using Go |

Headless builds need Go 1.26 or newer and no C toolchain, system groff or installed fonts. The [optional viewer](viewer.md) has separate platform requirements.

Syntax recognition does not imply evaluation, rendering or complete grammar support. The AST preserves original source; canonical conversion does not promise byte-identical round trips. Tables, equations, full macro execution and GNU layout equivalence are outside the supported profile. The [opt-in evaluator](evaluation.md) has its own limited language and resource budgets.

See [getting started](getting-started.md), [Go API examples](go-api.md) and [CLI](cli.md).
