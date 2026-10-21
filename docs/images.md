# Paginated PNG/JPEG export

```sh
gtroff render --format png -o page.png examples/man-tool.1
gtroff render --format jpg --theme dark --quality 90 -o page.jpg examples/man-tool.1
gtroff render --format png --section OPTIONS --single -o options.png examples/man-tool.1
gtroff render --format png --width 800 --page-height 1000 -o page.png long-page.1
```

A single page writes `page.png`; multiple pages write `page-001.png`,
`page-002.png`, etc. The base file is not a stitched/cropped substitute. Standard
output accepts exactly one image and rejects a multi-page plan before writing.
`--single` derives a content-sized height within `--max-height` and `--max-pixels`;
it errors if the whole document cannot fit. `--node` selects the AST ID shown by
`gtroff ast`, mapping to its complete semantic block, as for HTML.

See the [light PNG](../examples/man-tool.png) and [dark JPEG](../examples/man-tool.jpg)
from the same [synthetic man page](../examples/man-tool.1). Both files are checked
against the CLI in tests and were visually inspected. Neither generation nor
viewing requires installed man pages. Generation uses no groff, browser, fontconfig,
external image process, network or CGO.

## Layout and fonts

The renderer measures styled runs with `x/image/font/opentype`, then paginates a
bounded display list before allocating RGBA images. Painting uses the measured
positions and checks actual glyph bounds; a glyph that cannot fit returns an
error instead of producing a clipped image. Each image allocation is one page.
Widths account for glyph ink/overhang as well as advance. Line ascent/descent
include actual fonts and glyphs; line height is a ratio applied to those metrics.
Word wrapping crosses source/run boundaries without losing emphasis or text.

Headings, metadata, flowing paragraphs, hard breaks, definitions, nested indents,
preformatted/source whitespace and safe link coloring/underlining are supported.
Definition tags occupy their own line with bodies indented by the tag width.
Selected bodies retain actual ancestor item/indent widths. Headings reserve space
for a following body line when possible; paragraphs may split at line boundaries.
Inter-block spacing collapses by the larger default gap; explicit `.sp` values
add spacing. `.PD` paragraph distances, including zero, are honored. Top-of-page
vertical glue is suppressed. Preformatted tabs use four-column source tab stops.
Long preformatted lines and unbreakable words wrap with diagnostics. Unsupported
syntax remains visible as literal mono content with diagnostics.

Default faces are bundled Go Regular, Bold, Italic, Bold Italic, Mono and the Go
Mono bold/italic combinations. The [Go font license](../fontset/LICENSE-Go-Fonts.txt)
remains in the repository. `--font` / `--font-regular`, `--font-bold`,
`--font-italic` and `--font-mono` load only explicitly requested TTF/OTF files,
up to 2 MiB per face. Other faces retain bundled defaults. Custom mono bytes apply
to all mono styles; bold/italic use of that custom face diagnoses the absent style
variants. The API also accepts explicit proportional bold/italic bytes.

Go font coverage is finite. A missing glyph becomes a readable `[U+XXXX]` label
in a bundled mono face, with a diagnostic; original semantic/source text remains
independent. Controls/format characters become visible labels. Combining marks,
Arabic and Hebrew diagnose unsupported shaping/bidi and show logical-order glyphs.
This renderer does not implement a browser shaping engine, full roff evaluation,
complex-script typography or pixel-perfect groff typesetting.

## Units and bounds

| Setting | Default | Allowed |
| --- | --- | --- |
| `--width` | 1000 | 128..8192 device pixels |
| `--margin` | 48 | nonnegative device pixels, leaving useful drawing space |
| image `--font-size` | 15 | 6..72 points (1/72 inch) |
| `--dpi` | 96 | 36..300 pixels/inch |
| image `--line-height` | 1.35 | 1..3 times measured font height |
| `--page-height` | 1400 | 128..8192 device pixels |
| `--max-height` | 32768 | 128..32768 pixels; applies to single-image height too |
| `--max-pixels` | 67108864 | total pixel count across all planned pages |
| `--quality` | 90 | JPEG quality 1..100; PNG remains lossless |

Page geometry remains in device pixels when DPI changes; a 15-point font at
96 DPI measures 20 pixels before its font metrics/line-height ratio. HTML's
existing font-size option remains in CSS pixels. Layout rejects impossible lines
and nonfinite/overflowing geometry. Extreme indentation is clamped with a warning
so content stays visible. No image is silently scaled to bypass a budget.

API defaults also cap plans at 256 pages, 250000 measured glyphs and 64 MiB of
total encoded output. Each budget may be reduced but cannot exceed its hard
ceiling. The total pixel ceiling is 64 Mi pixels (up to 256 MiB of RGBA for a
single maximum-size page); font/layout/encoding buffers add memory separately.
Callbacks retaining images consume additional memory outside the stream's one-page
allocation pattern. The CLI buffers bounded encoded pages before file replacement.

## API and error behavior

```go
// Imports: "github.com/arran4/golang-gtroff/renderimage"
plan, err := renderimage.Layout(document, renderimage.Options{Width: 800})
if err != nil {
    return err
}
result, err := plan.Encode(func(page renderimage.Page, encoded []byte) error {
    // Save each owned PNG byte slice using an explicit caller-provided sink.
    return savePage(page.Number, encoded)
})
// Handle err and result.Diagnostics.
```

`LayoutSelection` accepts an existing semantic selection. `Plan.Pages()` returns
caller-owned textual layout snapshots: node IDs, block kinds, lines, baselines,
painted/source text, faces and glyph bounds. `Plan.Image(index)` returns an
`image.Image`; `Stream` calls a sink once per image. `Encode` streams encoded
pages and enforces the total output budget. `WritePage` buffers one page before
writing. `Render(writer, document, options)` requires exactly one page. Plans own
font bytes and use separate face caches for each painting call, allowing concurrent
painting. Source/model objects are validated and never mutated.

Zero option fields request defaults. `NoMargin` explicitly requests a zero margin
in the API; CLI `--margin 0` sets it. `Format` accepts png, jpg or jpeg. `Fonts`
accepts explicit font bytes, never paths. The API opens no files. Layout diagnostics
include input semantic diagnostics even for a selection, so strict mode cannot
conceal unsupported content elsewhere in the supplied source.

`ErrLimit`, `ErrStrict` and `ErrMultiplePages` identify budget, strict and single-
writer failures. Strict failures return an inspectable plan but block painting.
Layout/budget/strict failures occur before writing. Writer failures can leave a
partial stream. A streaming sink may have committed earlier pages when a later
encoding/sink error occurs; it owns transactionality. `Result.Bytes` counts bytes
encoded/passed to the sink, not an assurance that the sink persisted them.

Named CLI output rejects source aliases, symlinks and special files. Multi-page
output preflights every destination, then writes/syncs/closes all temporary files
before replacing any destination. Each rename is atomic subject to the host
filesystem. The collection is **not** a global transaction: an OS rename failure
can leave earlier pages replaced and returns an error. No delete/copy fallback or
automatic directory creation occurs. Stale numbered pages from an older, longer
export are not deleted automatically. Exit codes follow `render`: 0 accepted with
possible warnings, 1 syntax/strict failure, 2 usage/selection, 3 I/O/resource error.

## Fixtures and explicit golden refresh

`testdata/image/*.txtar` stores readable source, options, measured layout, output
names and diagnostics. Binary PNG/JPEG goldens live separately under
`testdata/golden/image`. Tests compare encoded bytes with those goldens, decode
both formats, compare lossless PNG pixels, verify content-bearing pages have ink,
check bounds and compare every selected node's visible source contribution.
Normal tests never update artifacts.

Approve a refresh by running `go test . -run '^TestImageScenarios$' -update-images`.
Review the textual layout/diagnostic diff and visually inspect changed binary
images before committing. Pinning Go/font dependencies makes the baseline
reproducible; a deliberate rasterizer/font change can require an approved refresh.
The example files are changed only by the explicit CLI commands above and must
still match the byte-checking example tests. Platform smoke CI runs native CGO-free
unit/image/executable tests on Linux, Windows and macOS; local foreign-target
cross-builds alone do not prove native execution.
