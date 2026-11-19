package script_test

import (
	"context"
	"github.com/arran4/golang-gtroff/source"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/script"
)

func BenchmarkScriptEnvironment(b *testing.B) {
	set := inputForBenchmark(b)
	request := script.Request{Name: "bench.tengo", Sources: set, Code: []byte(`for m in doc.query("definition") {if m.term=="TOOL_HOME"{doc.emit_json({term:m.term,file:m.source})}}`)}
	b.ReportAllocs()
	var compileTime, runTime int64
	var steps int
	for i := 0; i < b.N; i++ {
		result, e := script.Run(context.Background(), request, script.Options{})
		if e != nil || len(result.Output) == 0 {
			b.Fatal(result, e)
		}
		compileTime += int64(result.Stats.CompileDuration)
		runTime += int64(result.Stats.RunDuration)
		steps += result.Stats.Steps
	}
	b.ReportMetric(float64(compileTime)/float64(b.N), "compile-ns/op")
	b.ReportMetric(float64(runTime)/float64(b.N), "run-ns/op")
	b.ReportMetric(float64(steps)/float64(b.N), "checkpoints/op")
}

func inputForBenchmark(b *testing.B) *source.Set {
	b.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(".SH ENVIRONMENT\n.TP\nTOOL_HOME\nDirectory.\n"), "bench.1", source.Options{})
	if err != nil {
		b.Fatal(err)
	}
	return set
}
