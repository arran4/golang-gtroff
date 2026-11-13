package groff_test

import (
	"os"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/gen"
	"github.com/arran4/golang-gtroff/man"
)

type parserWorkload struct{ name, path, pkg string }

var parserWorkloads = []parserWorkload{
	{"man-small", "testdata/simple.1", "man"},
	{"man-medium", "testdata/bench/environment.1", "man"},
	{"ms-small", "testdata/bench/report.ms", "ms"},
	{"roff-small", "testdata/bench/requests.roff", "roff"},
}

func workloadSource(t testing.TB, w parserWorkload) string {
	t.Helper()
	data, e := os.ReadFile(w.path)
	if e != nil {
		t.Fatal(e)
	}
	if len(data) == 0 {
		t.Fatal("empty workload", w.path)
	}
	return string(data)
}
func workloadOptions(w parserWorkload) []groff.Option {
	return []groff.Option{groff.WithPackage(w.pkg), groff.WithRegistry(groff.NewRegistry()), groff.WithStrict(true), groff.WithSourceName(w.path)}
}

// Keep fixture correctness independent of timing. Exact captured source and
// syntax recognition do not establish byte-for-byte canonical token roundtrips
// or full macro evaluation/rendering support (especially for ms/roff).
func TestBenchmarkWorkloadContracts(t *testing.T) {
	for _, w := range parserWorkloads {
		t.Run(w.name, func(t *testing.T) {
			input := workloadSource(t, w)
			options := workloadOptions(w)
			document, e := groff.ParseDocument(strings.NewReader(input), options...)
			if e != nil || document == nil {
				t.Fatal(e)
			}
			if !document.SourceComplete || document.Source() != input || len(document.Diagnostics) > 0 || len(document.Tokens) < 5 {
				t.Fatal("invalid document fixture", document.Diagnostics)
			}
			tree, e := groff.ParseAST(strings.NewReader(input), options...)
			if e != nil || tree == nil {
				t.Fatal(e)
			}
			if !tree.SourceComplete() || tree.Source() != input || len(tree.Diagnostics()) > 0 {
				t.Fatal("invalid AST fixture", tree.Diagnostics())
			}
			if w.pkg == "man" {
				model, e := man.Build(tree, man.Options{Strict: true})
				if e != nil || model == nil || len(model.Diagnostics) > 0 {
					t.Fatal(e, model)
				}
				if e = model.Validate(); e != nil {
					t.Fatal(e)
				}
				if w.name == "man-medium" && len(model.FindTerm("BENCH_VALUE_099", false)) != 1 {
					t.Fatal("missing final definition")
				}
			}
		})
	}
}

func BenchmarkParseDocument(b *testing.B) {
	for _, w := range parserWorkloads {
		b.Run(w.name, func(b *testing.B) {
			input := workloadSource(b, w)
			options := workloadOptions(w)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, e := groff.ParseDocument(strings.NewReader(input), options...); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
func BenchmarkParseAST(b *testing.B) {
	for _, w := range parserWorkloads {
		b.Run(w.name, func(b *testing.B) {
			input := workloadSource(b, w)
			options := workloadOptions(w)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, e := groff.ParseAST(strings.NewReader(input), options...); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}

// Isolate semantic construction over an existing immutable AST. Parsing is a
// separate benchmark; no ms/roff semantic support is inferred from this model.
func BenchmarkManBuild(b *testing.B) {
	for _, w := range parserWorkloads {
		if w.pkg != "man" {
			continue
		}
		b.Run(w.name, func(b *testing.B) {
			input := workloadSource(b, w)
			tree, e := groff.ParseAST(strings.NewReader(input), workloadOptions(w)...)
			if e != nil {
				b.Fatal(e)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, e := man.Build(tree, man.Options{Strict: true}); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
func BenchmarkParseArguments(b *testing.B) {
	for _, tc := range []struct{ name, input string }{
		{"ascii", `title 1 date source manual`},
		{"quoted-escaped", `"two words" escaped\ space "quoted\"value" \" trailing comment`},
		{"unicode", `"café 東京" naïve\ space \[em]`},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.input)))
			for b.Loop() {
				if _, e := groff.ParseArguments(tc.input); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
func BenchmarkLexArguments(b *testing.B) {
	input := `"café 東京" escaped\ space \[em] \" comment`
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	for b.Loop() {
		if _, _, e := groff.LexArguments(input); e != nil {
			b.Fatal(e)
		}
	}
}
func BenchmarkGeneratorArgumentGrammar(b *testing.B) {
	for _, signature := range []struct{ name, input string }{
		{"required", `title section`},
		{"nested-optional", `title section [extra1 [extra2 [extra3]]]`},
		{"repeating", `[word1 [word2 ...]]`},
	} {
		b.Run(signature.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(signature.input)))
			for b.Loop() {
				if _, e := gen.ParseArgs(signature.input); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}

var benchmarkDefinition *groff.CatalogueDefinition

func BenchmarkCatalogueLookup(b *testing.B) {
	for _, tc := range []struct {
		name, token, pkg string
		found            bool
	}{
		{"man", ".TH", "man", true}, {"ms-collision", ".SH", "ms", true},
		{"request", ".sp", "roff", true}, {"fallback", ".SH", "unknown", true},
		{"missing", ".NOT_A_CATALOGUE_TOKEN", "man", false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			got := groff.LookupCatalogue(tc.token, tc.pkg)
			if (got != nil) != tc.found {
				b.Fatal("wrong catalogue setup", got)
			}
			wantPackage := tc.pkg
			if tc.name == "fallback" {
				wantPackage = "man"
			}
			if got != nil && (got.Package != wantPackage || got.Token != tc.token) {
				b.Fatal("wrong catalogue package/token", got.Package, got.Token)
			}
			b.ReportAllocs()
			for b.Loop() {
				benchmarkDefinition = groff.LookupCatalogue(tc.token, tc.pkg)
			}
		})
	}
}
