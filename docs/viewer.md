# Optional Shiny desktop viewer

The desktop program lives in a separate Go module. The root parser, CLI, HTML,
images and HTTP server do not import Shiny, X11, GLFW or native GUI libraries.
Ordinary root `CGO_ENABLED=0 go test ./...` and `go build ./...` remain headless.

Build from a complete checkout:

```sh
cd desktop
CGO_ENABLED=0 go build -o gtroff-view ./cmd/gtroff-view
./gtroff-view ../examples/man-tool.1
./gtroff-view --section OPTIONS ../examples/man-tool.1
./gtroff-view --width 800 --page-height 700 ../examples/man-tool.1
```

On Windows, build with `go build -o gtroff-view.exe ./cmd/gtroff-view` from
`desktop`, then pass a local source file. On macOS, the actual GUI backend needs a
native Apple toolchain with `CGO_ENABLED=1`; a Linux cross-build cannot supply
Apple frameworks. The nested module's local `replace ..` deliberately uses the
same checkout's renderer. It is a source-checkout build, not a claimed
`go install ...@latest` distribution contract.

No system `man` utility, network service, browser, groff, fontconfig or source
include is needed. `FILE '-'` reads stdin explicitly; omitting FILE prints usage
rather than waiting on an interactive terminal. Help and version never open a
window. Optional `--font`, `--font-bold`, `--font-italic` and `--font-mono` load
only explicit TTF/OTF paths, capped at 2 MiB per face. Defaults remain bundled Go
faces. The input closes before the window opens, so rendering never rereads a
file or follows a source reference.

## Platforms

| Platform | Requirements |
| --- | --- |
| Linux | X11 or XWayland, valid DISPLAY/Xauthority; CGO-free |
| Windows | Shiny Win32 backend; experimental |
| macOS | Native Apple SDK/compiler and CGO; experimental |

There is no native Wayland backend. The packaged viewer configuration targets Linux amd64/X11. Unsupported backend combinations report an error without opening a window.

## Reading and controls

The viewer uses the existing `renderimage.Plan`, with its page breaks, font
coverage policy, diagnostics and resource limits. It caches only the current
page. Zoom scales that raster page; it does not reflow text or create a second
layout engine. Large zoom may soften text. Choose document width/font size/DPI at
startup for another base resolution.

| Input | Action |
| --- | --- |
| Wheel, arrows, H/J/K/L | Scroll/pan, clamped to current page |
| Left mouse drag | Grab and pan page |
| Shift-wheel | Horizontal scroll |
| + / - or Ctrl/Meta-wheel | Zoom 25%..400%; wheel zoom keeps the pointer's source point when bounds allow |
| F / Fit button | Fit page width; follow subsequent viewport resizing |
| 0 | Reset to 100% and top-left scroll position |
| N/P, brackets, Prev/Next buttons | Next/previous rendered page, retaining zoom |
| PageUp/PageDown/Space | Scroll by a screen; Shift-Space goes back |
| Ctrl-PageUp/Down | Previous/next rendered page |
| Home/End | Top/bottom of current page |
| D / Notes | Toggle conversion diagnostics |
| ? / F1 / Help | Show controls |
| Q / Escape / window close | Close cleanly |

The toolbar shows document title, page/count, zoom and diagnostic count. Source
warnings appear in the notes panel and are printed in full to stderr. Long UI
labels are clipped to the viewport; further diagnostics remain available on
stderr. Recoverable backend/gesture errors appear in a visible error banner.
Fatal input/font/backend/allocation failures return a nonzero exit and a terminal
message. `--strict` rejects conversion diagnostics before opening a window.
Exit codes are 0 normal close/help, 1 strict conversion, 2 usage/selection and
3 I/O/backend/resource failure.

A resize replaces the actual Shiny buffer and controller viewport. Zero-sized
minimized views release buffers until restored. The initial requested viewport
is allocated/painted explicitly, including when no window manager emits an initial
resize event. Paint requests coalesce while pending. Mouse dragging ends on focus
loss. UI drawing and controller use are serial in the event loop.

## Bounds and API

`viewer.Controller` is a deterministic native-independent state machine. It
validates page/viewport sizes, clamps zoom and offsets, handles anchored zoom,
page selection, fit/resize, dragging, toolbar hits and terminal close state.
`viewer.Load(reader, sourceName, renderOptions, viewport)` uses the existing
source-aware AST, semantic model and raster plan. `viewer.NewSession` also accepts
an existing image plan. `Session.Draw` paints directly into a matching viewport
RGBA buffer; `Frame` returns a viewport image for explicit consumers/tests.
Neither API opens files, displays, network connections or installed fonts.

Viewport allocation is capped at 4096×4096 / 16 Mi pixels. A scaled full-page
bitmap is never allocated; drawing clips/scales into the viewport. Renderer page,
pixel, glyph and source budgets still apply independently. The native driver may
need its own display/buffer storage, and callers retaining frames consume extra
memory. Oversized viewport events fail before allocating a buffer. `Load` sessions retain original bytes through `Session.Source()` independently of
the raster plan; `NewSession` accepts a plan without claiming a raw snapshot.
Viewing never writes source or canonical text.
