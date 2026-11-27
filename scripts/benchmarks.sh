#!/bin/sh
set -eu

usage() {
    cat <<'EOF'
Usage: scripts/benchmarks.sh [list|smoke|parser|generator|profile DIR|compare BEFORE AFTER]
  list        List available benchmarks.
  smoke       Run one iteration per root benchmark.
  parser      Repeat parser/AST/model/catalogue benchmarks (default).
  generator   Repeat generator signature and process-level check benchmarks.
  profile DIR Capture CPU and allocation profiles in an existing directory.
  compare     Compare two saved reports using benchstat.

BENCHTIME defaults to 500ms (2s for profile); COUNT defaults to 10.
Compare identical fixtures, Go versions, machines and flags.
Save reports outside the repository, e.g. scripts/benchmarks.sh > /tmp/before.txt.
GeneratorCheck B/op and allocs/op measure parent-process allocations only.
EOF
}

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mode=${1:-parser}
case "$mode" in
    -h|--help|help) usage; exit 0 ;;
    compare)
        [ "$#" -eq 3 ] || { usage >&2; exit 2; }
        command -v benchstat >/dev/null || { echo 'Install benchstat separately (golang.org/x/perf/cmd/benchstat).' >&2; exit 1; }
        exec benchstat "$2" "$3"
        ;;
    profile)
        [ "$#" -eq 2 ] || { usage >&2; exit 2; }
        dir=$(CDPATH= cd -- "$2" && pwd)
        ;;
    list|smoke|parser|generator)
        [ "$#" -le 1 ] || { usage >&2; exit 2; }
        ;;
    *) usage >&2; exit 2 ;;
esac
cd "$root"
export CGO_ENABLED=0
case "$mode" in
    list) go test ./... -run '^$' -list '^Benchmark' ;;
    smoke) go test . -run '^$' -bench . -benchmem -benchtime=1x -count=1 ;;
    parser|generator)
        pattern='Benchmark(Parse(Line|Document|AST|Arguments)|LexArguments|ManBuild|CatalogueLookup)'
        if [ "$mode" = generator ]; then pattern='^BenchmarkGenerator'; fi
        git rev-parse HEAD >&2
        go version >&2
        go test . -run '^$' -bench "$pattern" -benchmem -benchtime="${BENCHTIME:-500ms}" -count="${COUNT:-10}" -cpu=1
        ;;
    profile)
        go test . -run '^$' -bench '^BenchmarkParseAST/man-medium$' -benchtime="${BENCHTIME:-2s}" -cpu=1 -cpuprofile="$dir/gtroff-ast.cpu" -memprofile="$dir/gtroff-ast.mem" -o "$dir/gtroff-bench.test"
        go tool pprof -top "$dir/gtroff-bench.test" "$dir/gtroff-ast.cpu"
        go tool pprof -alloc_space -top "$dir/gtroff-bench.test" "$dir/gtroff-ast.mem"
        ;;
esac
