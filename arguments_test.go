package groff

import (
	"bytes"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/arguments/*.txtar
var argumentCases embed.FS
var updateArguments = flag.Bool("update-arguments", false, "explicitly refresh raw/decoded argument expectations")

func TestArgumentSourceScenarios(t *testing.T) {
	paths, err := fs.Glob(argumentCases, "testdata/arguments/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range paths {
		t.Run(fixture, func(t *testing.T) {
			data, err := argumentCases.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			files := map[string][]byte{}
			for _, file := range archive.Files {
				if _, exists := files[file.Name]; exists {
					t.Fatalf("duplicate file %s", file.Name)
				}
				switch file.Name {
				case "input.args", "input.hex", "error.txt", "expected.txt":
				default:
					t.Fatalf("unknown file %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, plain := files["input.args"]
			encoded, binary := files["input.hex"]
			if plain == binary {
				t.Fatal("exactly one input file is required")
			}
			if plain {
				input = bytes.TrimSuffix(input, []byte("\n"))
			} else {
				input, err = hex.DecodeString(strings.TrimSpace(string(encoded)))
				if err != nil {
					t.Fatal(err)
				}
			}
			list, parseErr := ParseArguments(string(input))
			wantErr := strings.TrimSpace(string(files["error.txt"]))
			if wantErr == "" && parseErr != nil || wantErr != "" && (parseErr == nil || !strings.Contains(parseErr.Error(), wantErr)) {
				t.Fatalf("want error %q, got %v", wantErr, parseErr)
			}
			assertArgumentSource(t, list, string(input))
			assertArgumentWrappers(t, string(input), list, parseErr)
			var got strings.Builder
			byteStop, runeStop := list.Stop()
			fmt.Fprintf(&got, "source=%q stop=%d/%d\n", list.Source(), byteStop, runeStop)
			lexemes, spans := list.Lexemes(), list.ByteSpans()
			for i, lexeme := range lexemes {
				fmt.Fprintf(&got, "arg %d bytes=[%d,%d) runes=[%d,%d) quoted=%v raw=%q value=%q\n", i, spans[i].Start, spans[i].End, lexeme.Start, lexeme.End, lexeme.Quoted, lexeme.Raw, lexeme.Value)
			}
			if span, ok := list.Comment(); ok {
				fmt.Fprintf(&got, "comment [%d,%d) %q\n", span.Start, span.End, list.Source()[span.Start:span.End])
			}
			if span, ok := list.Unparsed(); ok {
				fmt.Fprintf(&got, "unparsed [%d,%d) %q\n", span.Start, span.End, list.Source()[span.Start:span.End])
			}
			canonical, err := list.Canonical()
			if err != nil {
				fmt.Fprintf(&got, "canonical error=%v\n", err)
			} else {
				fmt.Fprintf(&got, "canonical=%q\n", canonical)
			}
			if parseErr != nil {
				fmt.Fprintf(&got, "syntax error=%v\n", parseErr)
			}
			if *updateArguments {
				found := false
				for i, file := range archive.Files {
					if file.Name == "expected.txt" {
						archive.Files[i].Data = []byte(got.String())
						found = true
					}
				}
				if !found {
					archive.Files = append(archive.Files, txtar.File{Name: "expected.txt", Data: []byte(got.String())})
				}
				if err := os.WriteFile(fixture, txtar.Format(archive), 0644); err != nil {
					t.Fatal(err)
				}
			} else if got.String() != string(files["expected.txt"]) {
				t.Fatalf("got:\n%s\nwant:\n%s", got.String(), files["expected.txt"])
			}
		})
	}
}
func assertArgumentSource(t *testing.T, list *ArgumentList, input string) {
	t.Helper()
	if list == nil || list.Source() != input {
		t.Fatal("argument source lost bytes")
	}
	lexemes, spans := list.Lexemes(), list.ByteSpans()
	if len(lexemes) != len(spans) {
		t.Fatal("argument/span cardinality differs")
	}
	cursor := 0
	var reconstructed strings.Builder
	for i, span := range spans {
		if span.Start < cursor || span.End < span.Start || span.End > len(input) {
			t.Fatalf("invalid byte span: %+v", span)
		}
		lexeme := lexemes[i]
		if lexeme.Raw != input[span.Start:span.End] {
			t.Fatalf("raw lexeme changed source bytes: %q vs %q", lexeme.Raw, input[span.Start:span.End])
		}
		if lexeme.Start != utf8.RuneCountInString(input[:span.Start]) || lexeme.End != utf8.RuneCountInString(input[:span.End]) {
			t.Fatalf("rune offsets disagree with byte spans: %+v %+v", lexeme, span)
		}
		reconstructed.WriteString(input[cursor:span.Start])
		reconstructed.WriteString(lexeme.Raw)
		cursor = span.End
	}
	reconstructed.WriteString(input[cursor:])
	if reconstructed.String() != input {
		t.Fatal("raw/gap/tail reconstruction changed bytes")
	}
	byteStop, runeStop := list.Stop()
	if byteStop < 0 || byteStop > len(input) || runeStop != utf8.RuneCountInString(input[:byteStop]) {
		t.Fatal("invalid stop offsets")
	}
	if span, ok := list.Comment(); ok {
		if span.Start != byteStop || span.End != len(input) || !strings.HasPrefix(input[span.Start:span.End], `\"`) {
			t.Fatal("comment extent changed")
		}
	}
	if span, ok := list.Unparsed(); ok {
		if span.Start < cursor || span.End != len(input) {
			t.Fatal("malformed argument tail lost bytes")
		}
	}
}
func assertArgumentWrappers(t *testing.T, input string, list *ArgumentList, parseErr error) {
	t.Helper()
	values, index, err := ArgumentParser(input)
	lexemes, lexIndex, lexErr := LexArguments(input)
	_, runeStop := list.Stop()
	if index != runeStop || lexIndex != runeStop {
		t.Fatal("legacy rune-index contract changed")
	}
	if parseErr != nil {
		if values != nil || lexemes != nil || !errors.Is(err, ErrUnclosedQuote) || !errors.Is(lexErr, ErrUnclosedQuote) {
			t.Fatal("legacy error contract lost")
		}
		var syntax *ArgumentSyntaxError
		if !errors.As(err, &syntax) {
			t.Fatal("compatibility wrapper hid positional cause")
		}
		return
	}
	if err != nil || lexErr != nil || !reflect.DeepEqual(lexemes, list.Lexemes()) {
		t.Fatalf("scanner wrappers disagree: %v/%v", err, lexErr)
	}
	var want []string
	for _, lexeme := range lexemes {
		want = append(want, lexeme.Value)
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("typed values differ from raw lexer: %q vs %q", values, want)
	}
}
func TestArgumentAccessorsAndPartialError(t *testing.T) {
	input := `ok "unfinished`
	list, err := ParseArguments(input)
	var syntax *ArgumentSyntaxError
	if !errors.Is(err, ErrUnclosedQuote) || !errors.As(err, &syntax) || syntax.Span != (ArgumentSpan{3, len(input)}) {
		t.Fatalf("missing error identity/position: %v", err)
	}
	if len(list.Lexemes()) != 1 || list.Lexemes()[0].Value != "ok" {
		t.Fatal("successful prefix unavailable")
	}
	lexemes, spans := list.Lexemes(), list.ByteSpans()
	lexemes[0].Raw = "changed"
	spans[0].Start = 99
	if list.Source() != input || list.Lexemes()[0].Raw != "ok" || list.ByteSpans()[0].Start != 0 {
		t.Fatal("accessors exposed mutable snapshot storage")
	}
	if _, err := list.Canonical(); !errors.Is(err, ErrUnclosedQuote) {
		t.Fatal("canonical serialization silently dropped malformed tail")
	}
}
func TestCanonicalArguments(t *testing.T) {
	for _, values := range [][]string{nil, {""}, {"a b", "café", "\t", `\n[X]`}, {"tail with space\\"}, {`\`}, {string([]byte{0xff, 0}), "end"}} {
		source, err := CanonicalArguments(values)
		if err != nil {
			t.Fatalf("representable values %q rejected: %v", values, err)
		}
		decoded, _, err := ArgumentParser(source)
		if err != nil || !reflect.DeepEqual(decoded, values) {
			t.Fatalf("canonical round trip: %q -> %q -> %q, %v", values, source, decoded, err)
		}
	}
	for _, values := range [][]string{{`a\ b`}, {`a\"b`}, {`\`, "next"}, {"line\nbreak"}} {
		out, err := CanonicalArguments(values)
		var encoding *ArgumentEncodingError
		if out != "" || !errors.Is(err, ErrArgumentEncoding) || !errors.As(err, &encoding) || encoding.Index < 0 || encoding.Index >= len(values) {
			t.Fatalf("ambiguous values silently serialized: %q -> %q, %v", values, out, err)
		}
	}
}
func FuzzParseArguments(f *testing.F) {
	for _, input := range []string{"", `a "" café`, `"unterminated\`, `a\ b \"comment`, string([]byte{0xff, 0, ' ', 0xfe}), `"a"b`} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 8192 {
			t.Skip()
		}
		list, err := ParseArguments(input)
		assertArgumentSource(t, list, input)
		assertArgumentWrappers(t, input, list, err)
	})
}
func FuzzCanonicalArguments(f *testing.F) {
	f.Add("hello world", `tail\`)
	f.Add(`\`, "next")
	f.Add(`a\ b`, `a\"b`)
	f.Add("", "")
	f.Fuzz(func(t *testing.T, first, second string) {
		if len(first)+len(second) > 8192 {
			t.Skip()
		}
		want := []string{first, second}
		source, err := CanonicalArguments(want)
		if err != nil {
			if !errors.Is(err, ErrArgumentEncoding) {
				t.Fatal(err)
			}
			return
		}
		got, _, err := ArgumentParser(source)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("successful checked encoding changed values: %q -> %q -> %q", want, source, got)
		}
	})
}

func TestZeroArgumentRequestsRejectContentLoss(t *testing.T) {
	for _, input := range []string{`.br discarded`, `.nf ""`, `.PP text`} {
		_, err := ParseLineWithPackage(input, "man")
		var arity *UnexpectedArgumentsError
		if !errors.Is(err, ErrUnexpectedArguments) || !errors.As(err, &arity) || arity.Maximum != 0 {
			t.Fatalf("zero-argument request discarded input: %q, %v", input, err)
		}
		for _, strict := range []bool{false, true} {
			doc, err := ParseDocument(strings.NewReader(input+"\n"), WithPackage("man"), WithRegistry(NewRegistry()), WithStrict(strict))
			if (err != nil) != strict || doc == nil || doc.Source() != input+"\n" || len(doc.Tokens) != 1 || len(doc.Diagnostics) != 1 || doc.Tokens[0].RawSource != input+"\n" {
				t.Fatalf("arity recovery lost source/diagnostic: strict=%v, doc=%+v, err=%v", strict, doc, err)
			}
			if _, ok := doc.Tokens[0].Token.(*TokenUnknownControl); !ok {
				t.Fatalf("invalid request became a valid zero-argument token: %T", doc.Tokens[0].Token)
			}
		}
	}
	for _, input := range []string{`.br`, `.br \" comment`, `.nf`, `.PP`} {
		if _, err := ParseLineWithPackage(input, "man"); err != nil {
			t.Fatalf("valid zero-argument request rejected: %q %v", input, err)
		}
	}
	for _, match := range []func([]string, ...bool) ([]string, error){ForwardMatcher, BackwardMatcher} {
		if _, err := match([]string{"extra"}); !errors.Is(err, ErrUnexpectedArguments) {
			t.Fatal("compatibility matcher discarded all input")
		}
	}
}

func TestGeneratedZeroArgumentRoutes(t *testing.T) {
	var report struct {
		Entries []catalogueSupportEntry `json:"entries"`
	}
	if err := json.Unmarshal(catalogueSupportJSON, &report); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	checked := 0
	for _, entry := range report.Entries {
		if entry.ParseStatus != "supported_generated" {
			continue
		}
		token, err := registry.ParseLine(entry.Example, entry.Package)
		if err != nil {
			t.Fatal(err)
		}
		shape := reflect.TypeOf(token)
		if shape.Kind() != reflect.Pointer || shape.Elem().Kind() != reflect.Struct || shape.Elem().NumField() != 0 {
			continue
		}
		checked++
		_, err = registry.ParseLine(entry.Token+" extra", entry.Package)
		if !errors.Is(err, ErrUnexpectedArguments) {
			t.Fatalf("generated zero-argument route %s/%s discarded extra input: %v", entry.Package, entry.Token, err)
		}
	}
	if checked == 0 {
		t.Fatal("no zero-argument generated routes were exercised")
	}
	t.Logf("checked %d generated zero-argument routes", checked)
}
func TestConstructedTokenFieldRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   Token
	}{
		{".TH", []string{`T "quoted"`, "1", "2026", "Linux", "Manual"}, &TokenMacroTH{Title: `T "quoted"`, Section: "1", Extra1: "2026", Extra2: "Linux", Extra3: "Manual"}},
		{".IP", []string{"two words", "4"}, &TokenManIP{Designator: "two words", Nnn: "4"}},
		{".Pn", []string{"", "dir/世界 path", ""}, &TokenMacroPn{Path: "dir/世界 path"}},
	} {
		arguments, err := CanonicalArguments(tc.values)
		if err != nil {
			t.Fatal(err)
		}
		token, err := ParseLineWithPackage(tc.name+" "+arguments, "man")
		if err != nil || !reflect.DeepEqual(token, tc.want) {
			t.Fatalf("constructed fields %s: %v %v; want %v", tc.name, token, err, tc.want)
		}
		reparsed, err := ParseLineWithPackage(token.String(), "man")
		if err != nil || !reflect.DeepEqual(reparsed, tc.want) {
			t.Fatalf("representative legacy serialization %s: %v %v", tc.name, reparsed, err)
		}
	}
}
