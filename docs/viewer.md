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

## Verified platform matrix

| Target | Backend / CGO | Evidence and status |
| --- | --- | --- |
| Linux amd64 | Shiny X11, CGO=0 | Built and executed under Xvfb; native input, pixels, resize, full/selected pages and close verified. This is the verified desktop target. |
| Windows amd64 | Shiny Win32, CGO=0 | Cross-build passed on Linux. Runtime/interactive desktop unverified; available as a preview build. |
| macOS amd64, CGO=0 | Explicit unsupported-backend stub | Cross-build passed; it reports the missing native backend. This is not a GUI support claim. |
| macOS with native CGO | Shiny default OpenGL/GLFW backend | Backend source is enabled and a native CI build lane is configured. Local cross-build probe failed at the Linux compiler's unsupported Apple `-arch` option; no local native build/runtime is claimed. |

Linux requires an X server or XWayland and a valid DISPLAY/Xauthority connection.
It does not require native Xlib/GLFW linkage: the tested binary uses the pure-Go
X11 driver. There is no native Wayland backend in this implementation. Windows
uses the Shiny Windows backend. macOS requires its native compiler/SDK/frameworks;
unsupported OS/build combinations return a clear backend error while help/version
still work. Experimental Shiny/mobile dependencies are pinned in `desktop/go.mod`.
Ordinary unit tests never open a display. Hosted CI results are reported separately
from local cross-build and Xvfb execution evidence.

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

## Tests and native smoke

Root `testdata/viewer/*.txtar` scenarios capture readable gesture/state/rectangle
expectations, including invalid geometry and closed state. Approve refreshes
explicitly with `go test . -run '^TestViewerScenarios$' -update-viewer`.
Controller/session/adapter/CLI tests need no display. Root tests also continue to
exercise the semantic conversion loops and raster fixtures unchanged.

On a Linux X11 test machine, install Xvfb, xauth and xdotool only as test tools:

```sh
cd desktop
CGO_ENABLED=0 xvfb-run -a -s '-screen 0 1600x1200x24 -nolisten tcp' \
  go test ./cmd/gtroff-view -run '^TestX11Executable$' -count=1 -gui-smoke
```

The smoke test launches the actual binary with the synthetic man fixture, finds
its new X11 window by `_NET_WM_NAME`, sends native wheel/key/drag/resize input and
reads pixels back from that window. Each observed frame must match the expected
controller/session frame before advancing. It verifies full and named-section
views, next/previous pages, fit, notes, help and process close. The test owns its
process/window and does not manipulate existing viewer windows. It waits for
observable state rather than assuming startup sleeps succeeded.

The [captured Linux screenshot](../examples/gtroff-view-linux.png) is a genuine
X11 client-window capture after resize, fit, page switch and notes. It was visually
inspected after capture from the running viewer.
To approve a new screenshot artifact, pass
`-screenshot /absolute/path/to/examples/gtroff-view-linux.png` to that explicit
smoke command, then inspect the image before committing. Normal tests do not
refresh screenshots. On failure, that explicit path may also receive
`.actual.png`/`.expected.png` debug captures. Native CI has a separate X11 smoke
lane and optional module build/unit lanes; headless jobs stay independent.

For a manual smoke on each additional desktop before claiming runtime support:
open `examples/man-tool.1` with `--width 800 --page-height 700`; read all pages,
scroll/drag, zoom at a pointer, fit, resize the real window, switch pages, open
notes/help and close. Repeat with `--section OPTIONS` and an unknown macro input
with warnings. Record OS/architecture/backend/CGO and capture an actual window.
A successful cross-build alone does not satisfy this runtime procedure.

## Optional release candidate

The separate root `.goreleaser-view.yaml` produces only the locally executed
Linux amd64/X11/CGO-free distribution. It never participates in headless linking.
Windows and macOS viewer artifacts are not advertised or packaged. Full CI lanes
may attempt their native source builds as optional checks.

After extracting a candidate archive, test that exact executable rather than
rebuilding it:

```sh
CGO_ENABLED=0 xvfb-run -a -s '-screen 0 1600x1200x24 -nolisten tcp' go test ./cmd/gtroff-view -run '^TestX11Executable$' -count=1 -gui-smoke -gui-binary /absolute/path/to/gtroff-view
```

Run from `desktop`. The explicit `-gui-binary` path selects the existing candidate;
without it the test retains its usual build-and-smoke behavior. See the
[release graph, gates and verification scope](releasing.md).
