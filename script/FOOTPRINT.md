# Local footprint checkpoint

Informational measurements on 2026-11-06, Go toolchain go1.26.10, Linux amd64,
CGO disabled, Intel i5-12600KF. These are observations of development builds,
not release-size, speed, platform-runtime or API-stability guarantees.

| Build | Bytes |
| --- | ---: |
| Default core CLI, `go build -trimpath ./cmd/gtroff` | 20,299,505 |
| Optional CLI, `go -C script build -trimpath ./cli` | 19,499,574 |
| Optional CLI, `GOOS=js GOARCH=wasm go -C script build -trimpath ./cli` | 26,183,414 |

The two native binaries use different entrypoints: the default includes the
complete generated command dispatcher; the optional variant uses the shared
flags-before-input CLI dispatcher. Their size difference is not an isolated
measurement of Tengo's incremental cost. Both include the existing render/export
libraries. The core dependency list contains no github.com/d5/tengo package.

`BenchmarkScriptEnvironment` runs an actual one-definition document query and
JSON emission, including fresh working capture, native compilation,
instrumentation and VM startup: approximately 73.1 microseconds, 144,592 B and
570 allocations per invocation. Reported internal phase averages were 26.0 us
compile/instrumentation and 39.4 us VM/host execution, with three checkpoints.
Timing overhead and machine noise are included; there is no benchmark gate.

Reproduce from the repository root:

```sh
CGO_ENABLED=0 go build -trimpath -o /tmp/gtroff-core ./cmd/gtroff
CGO_ENABLED=0 go -C script build -trimpath -o /tmp/gtroff-tengo ./cli
CGO_ENABLED=0 GOOS=js GOARCH=wasm go -C script build -trimpath -o /tmp/gtroff-tengo.wasm ./cli
go -C script test . -run '^$' -bench BenchmarkScriptEnvironment -benchmem -benchtime 200ms
go list -deps ./cmd/gtroff
```

The pinned Tengo v2.17.0 module itself declares Go 1.13 and no module dependencies.
The optional adapter depends on the local root module and its already pinned
libraries; x/tools/txtar is a test-only fixture dependency. Its MIT notice is
retained. No parser-module engine dependency or license change was introduced.

Actual Go/js execution through the Go toolchain's Node verification adapter
passed document queries/edits, source/error mapping, capability denial,
allocation/host/output limits, loop checkpoints and deadline cases. Node is a
verification host, not a runtime dependency of the native optional binary.
An uninstrumented empty loop first exhausted the external test timeout; the
checkpoint adapter fixed that concrete scheduling limitation. See the guide for
remaining native-operation/heap-quota limits. Platform compilation and native
runtime execution must be reported separately.
