# Read-only man-page HTTP library

```sh
go build -o gtroff ./cmd/gtroff
./gtroff serve --root examples --listen 127.0.0.1:8080
# gtroff: serving 2 pages at http://127.0.0.1:8080 (read-only snapshot)
```

Open `http://127.0.0.1:8080`, search for `man-tool`, and follow its section links.
The same command works with provided collections on Linux, Windows and macOS.
No installed `man` command or database is required. On Linux, explicitly select
an installed collection, for example `--root /usr/share/man/man1`. Repeat `--root`
for other collections. Nothing scans host man directories implicitly. Port `0`
chooses an available port; the actual address is printed to stderr. Ctrl-C stops
the server. Stdout is unused.

The index includes plain `.1`, `.3`, `.3p`, `.3ssl`, `.n`, `.l` and corresponding
`.gz` files. Directories may contain `man1` etc, locale subdirectories or flat
page files. Names can contain letters, combining marks, digits, dot, underscore, plus, minus and
`@`; unsupported filenames are ignored. Hidden directories are skipped. Symlinks,
special files and unreadable/corrupt compressed pages produce collection notices.
Every included source is parsed without evaluating macros or following includes;
malformed/unsupported text remains visible with conversion notes.

Entries sort by name, section, configured root order and relative path. Two names
in different sections have distinct routes; duplicate identities in different
roots retain variants. The first configured root wins an unqualified lookup.
Links include the root label to retain the selected variant. Plain/compressed
variants within a root use relative-path ordering. Anonymous CLI labels such as
`root-1` avoid publishing absolute host root paths.

## Routes and search

| Route | Behavior |
| --- | --- |
| `/` | Catalogue, 50 entries per result page |
| `/?page=2` | Next catalogue page |
| `/search?q=term` | Built-in search; AND-substring terms, case-insensitive |
| `/page/1/man-tool` | Page by name and section; first configured variant |
| `/page/1/man-tool?root=root-1` | Specific configured variant |
| `/page/1/man-tool?heading=OPTIONS` | All matching semantic sections |

Search covers names, section suffixes, extracted NAME descriptions and full
semantic text, including visible unsupported source. Ranking prefers exact names,
prefixes, name substrings, descriptions and then full text. Ties keep catalogue
order. Search returns at most 1000 matches with an explicit limit notice; refine
the query to find more. Catalogue pagination remains available for every indexed
entry. Queries are bounded to 256 runes/1024 decoded bytes and 2048 encoded query
bytes; complete request URIs are capped at 4096 bytes. Unknown/repeated parameters
are rejected. Page section navigation targets the renderer's actual stable node
anchors, rather than a plain text dump. A filtered page's navigation links return
to the complete page for sections that were not selected.

Only GET and HEAD are accepted. Other methods return 405 with `Allow: GET, HEAD`.
Invalid identities/traversal/queries return 400, unknown pages/headings 404,
unrecognized Host values 403, rendering rejection 422 and resource/concurrency
failures 503. Cancellation returns 408 before response construction completes.
HEAD reports the same constructed content length while sending no body.

The listener defaults to numeric IPv4 loopback. `localhost` is normalized to that
address; numeric IPv6 loopback is also supported. Non-loopback binding requires
`--allow-remote`, which also allows arbitrary HTTP Host values. That mode has no
authentication: every reachable client can read the configured collection. The
library handler otherwise accepts only localhost/loopback Host values, or an
explicit `AllowedHosts` list, to reject DNS-rebinding-style host names. A reverse
proxy must configure those allowed names deliberately.

## Optional system search

`--system-search` explicitly probes `apropos`, then `whatis`, then `man -k` through
an injected provider. It supplements built-in results and never broadens the
configured roots. A system result links to a page only if it is already indexed;
other results are escaped plain labels/descriptions. Missing tools, failures,
timeouts or invalid output fall back to built-in results with a notice. Nothing
probes or executes system search while this flag is absent.

Programs are executed directly with argument arrays and `--` before the query,
without a shell. Defaults cap each command at one second and 64 KiB stdout, with
up to 256 parsed hits. Provider limits may not exceed five seconds/256 KiB.
The real process adapter has an independent five-second ceiling and a bounded
pipe-drain delay. Unit tests inject lookup/runner capabilities; no installed
man-db binary is required. Process integration tests use an owned Go test helper.
The explicitly enabled utility still uses its own system database/configuration;
its results are descriptive, not authority to open another file.

## Filesystem and snapshot contract

`manlib.Build(context, []Root, Limits)` receives explicit `fs.FS` capabilities.
`OpenDirectory(path)` wraps `os.OpenRoot`; its methods prevent symlink escape on
the supported headless platforms. Scanning skips symlinks entirely. Arbitrary
filesystem implementations must provide confinement themselves and directories
must implement `fs.ReadDirFile` for bounded incremental listing. Tests use
`fstest.MapFS`, not host man directories. Directory entries and file types are
validated; there is no request-driven file opening or HTTP resource fetching.

Build reads/decompresses and validates sources once. It retains immutable source
and searchable-text snapshots, not every parsed tree. Requests parse only the
selected cached source with bounded AST/model budgets. Closing root handles,
editing/removing files or mutating returned models cannot change the served
snapshot. Restart or call Build again to refresh it; file watching, incremental
invalidation and on-disk indexes are deferred. `.so` aliases remain literal
unsupported input. No include resolver/evaluator is exposed; any future opt-in
resolver must retain the root sandbox.

All opened files are closed, including read/decode/error paths. Caller-owned FS
operations can block according to their implementation; cancellation is checked
between operations and throughout bounded reads/search. Synchronous parser/render
work checks context at stage boundaries rather than promising immediate CPU
preemption. Root directories should be stable and trusted; OS roots do not block
mounts/devices placed inside them. Nonregular entries are skipped/rechecked, but
this is not a sandbox for an adversary actively replacing regular files with
blocking special files during a scan.

## Bounds and API

Default index limits are 10000 pages, 100000 scanned directory entries, 12 directory
levels, 2 MiB raw/decoded per page, 64 MiB total source/decode bytes and 20000 AST/
semantic nodes per page. Hard ceilings are respectively 50000, 500000, 64, 8 MiB,
256 MiB and 100000. Compressed pages count both raw and decoded bytes against the
total. The same budget is enforced during raw reads and decompression, shared
across every configured root and including bytes returned by failed reads,
failed closes and invalid gzip attempts (including output before a checksum
error). One extra byte may be consumed to detect a limit violation; Build then
stops without opening later pages. Decompression cannot spend a full per-page
budget after the aggregate budget runs out. Exact-limit inputs are accepted if
their own reads/decode succeed. Files within the limit that cannot be read or
decoded still produce collection notices. AST physical-line limits
also apply. Zero API limits request defaults; positive limits reduce or raise them
within the ceilings. CLI `--max-pages`, `--max-file-bytes` and `--max-total-bytes`
allow explicit budget changes. A budget failure aborts CLI startup before listening.
Build returns a partial inspectable index alongside an error; serving it would be
an explicit caller decision, not the CLI default. See the
[executable failed-attempt budget example](../manlib/read_budget_example_test.go).

The handler defaults to four simultaneous requests, a 4 MiB constructed-response
budget and 1000 search hits. Hard ceilings are 32 concurrent requests, 16 MiB and
1000 hits. It buffers a response before writing, applies CSP/no-sniff/no-referrer/
no-store headers and escapes every filename, query, description and provider result
with `html/template`. Only HTML produced by the validated source-escaping renderer
is inserted as trusted markup. Default page fonts are the four embedded Go faces;
`--assets omit-fonts` explicitly omits those assets. No source HTML executes.

```go
// Imports: "github.com/arran4/golang-gtroff/manlib"
//          "github.com/arran4/golang-gtroff/renderhtml"
directory, err := manlib.OpenDirectory(configuredPath)
if err != nil {
    return err
}
index, buildErr := manlib.Build(ctx, []manlib.Root{directory.Root("manuals")}, manlib.Limits{})
closeErr := directory.Close()
if buildErr != nil {
    return buildErr
}
if closeErr != nil {
    return closeErr
}
handler, err := manlib.Handler(index, manlib.HandlerOptions{Render: renderhtml.Options{}})
// Mount handler on an owned server; or use manlib.Serve(ctx, listener, handler).
```

`Index.Entries`, `Notices`, `Search` and `Document` expose caller-owned descriptions,
results and freshly parsed semantic models. `Handler` has no global mux or process
state. `Serve` owns its listener and sets header/read/write/idle deadlines and a
16 KiB header limit; context cancellation triggers bounded graceful shutdown.
The convenience server discards transport logs instead of using process-global
stderr; embedding applications can supply their own configured `http.Server`.
CLI exit codes are 0 normal shutdown, 2 usage and 3 filesystem/listener/resource
failure. Signal handling is installed only for `serve`; existing batch commands
keep their ordinary process behavior.

`testdata/serve/*.txtar` contains readable virtual trees, explicit gzip synthesis,
requests, status/header/body snapshots and positive/negative substring assertions.
Budget cases include invalid gzip headers and generated streams with explicitly
corrupted checksum trailers; binary gzip data is synthesized from readable text.
Normal tests never update them. Explicit refresh:
`go test . -run '^TestServeScenarios$' -update-serve`. Unit/httptest, real-directory
symlink, cancellation, bounded-process, HTTP fuzz and built-executable tests verify
the public contracts without external network or system man dependencies.
