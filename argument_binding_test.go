package groff

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/binding/*.txtar
var bindingCases embed.FS

func TestArgumentBindingScenarios(t *testing.T) {
	paths, err := fs.Glob(bindingCases, "testdata/binding/*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := bindingCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for _, file := range txtar.Parse(data).Files {
				if _, exists := files[file.Name]; exists {
					t.Fatalf("duplicate %s", file.Name)
				}
				switch file.Name {
				case "input.roff", "package.txt", "fields.json":
				default:
					t.Fatalf("unknown %s", file.Name)
				}
				files[file.Name] = file.Data
			}
			input, ok := files["input.roff"]
			if !ok {
				t.Fatal("missing input.roff")
			}
			expected, ok := files["fields.json"]
			if !ok {
				t.Fatal("missing fields.json")
			}
			pkg := strings.TrimSpace(string(files["package.txt"]))
			if pkg == "" {
				pkg = "man"
			}
			line := strings.TrimSuffix(string(input), "\n")
			token, err := NewRegistry().ParseLine(line, pkg)
			if err != nil {
				t.Fatal(err)
			}
			// Decode expected fields into the concrete type, so field equality is checked
			// independently of canonical output or legacy String formatting.
			want := reflect.New(reflect.TypeOf(token).Elem()).Interface()
			decoder := json.NewDecoder(strings.NewReader(string(expected)))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(token, want) {
				t.Fatalf("fields: %#v, want %#v", token, want)
			}
			assertCanonicalToken(t, token, pkg)
			doc, err := ParseDocument(strings.NewReader(string(input)), WithRegistry(NewRegistry()), WithPackage(pkg), WithStrict(true))
			if err != nil || doc.Source() != string(input) || len(doc.Tokens) != 1 || doc.Tokens[0].RawSource != string(input) {
				t.Fatalf("raw document changed: %#v %v", doc, err)
			}
		})
	}
}
func assertCanonicalToken(t *testing.T, token Token, pkg string) {
	t.Helper()
	source, err := CanonicalToken(token)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := NewRegistry().ParseLine(source, pkg)
	if err != nil || !reflect.DeepEqual(parsed, token) {
		t.Fatalf("canonical %q: %#v %v, want %#v", source, parsed, err, token)
	}
}
func TestArgumentBindingErrors(t *testing.T) {
	for _, tt := range []struct {
		line  string
		cause error
	}{
		{`.TH T`, ErrMissingArguments}, {`.TH T 1 a b c extra`, ErrUnexpectedArguments},
		{`.IP tag 4 extra`, ErrUnexpectedArguments}, {`.sp (3 + 5) extra`, ErrUnexpectedArguments},
		{`.sp (3 + 5`, ErrInvalidArguments}, {`.nr X`, ErrMissingArguments},
		{`.nr X 1 2 3`, ErrUnexpectedArguments}, {`.hcode a 1 b`, ErrInvalidArguments},
		{`.VS 5`, ErrInvalidArguments}, {`.tl 'a'b'`, ErrInvalidArguments},
		{`.tl 'a'b'c' extra`, ErrUnexpectedArguments}, {`.if 'incomplete`, ErrInvalidArguments},
		{`.NT C title extra`, ErrUnexpectedArguments}, {`.write`, ErrMissingArguments}, {`.while 1`, ErrMissingArguments},
	} {
		t.Run(tt.line, func(t *testing.T) {
			tok, err := NewRegistry().ParseLine(tt.line, "man")
			if tok != nil || !errors.Is(err, tt.cause) {
				t.Fatalf("got %T %v; want %v", tok, err, tt.cause)
			}
			for _, strict := range []bool{false, true} {
				input := tt.line + "\nstill here\n"
				doc, err := ParseDocument(strings.NewReader(input), WithRegistry(NewRegistry()), WithStrict(strict))
				if doc == nil || doc.Source() != input || len(doc.Tokens) != 2 || len(doc.Diagnostics) != 1 {
					t.Fatalf("lost invalid source: %#v %v", doc, err)
				}
				if _, ok := doc.Tokens[0].Token.(*TokenUnknownControl); !ok {
					t.Fatalf("invalid token %T", doc.Tokens[0].Token)
				}
				if strict != (err != nil) {
					t.Fatalf("strict %v: %v", strict, err)
				}
			}
		})
	}
}
func TestGeneratedArgumentGrammars(t *testing.T) {
	var report struct{ Entries []catalogueSupportEntry }
	if err := json.Unmarshal(catalogueSupportJSON, &report); err != nil {
		t.Fatal(err)
	}
	fixed := 0
	for _, entry := range report.Entries {
		if entry.Parser == "" {
			continue
		}
		t.Run(entry.Package+"/"+entry.Token, func(t *testing.T) {
			token, err := NewRegistry().ParseLine(entry.Example, entry.Package)
			if err != nil {
				t.Fatal(err)
			}
			declarer, ok := token.(interface{ ArgumentGrammar() ArgumentGrammar })
			if !ok {
				t.Fatalf("no grammar for %T", token)
			}
			g := declarer.ArgumentGrammar()
			if err := g.validate(); err != nil {
				t.Fatal(err)
			}
			assertCanonicalToken(t, token, entry.Package)
			if g.Mode == ArgumentsFixed {
				fixed++
				line := entry.Token + strings.Repeat(" value", len(g.Optional)+1)
				if _, err := NewRegistry().ParseLine(line, entry.Package); !errors.Is(err, ErrUnexpectedArguments) {
					t.Fatalf("extra field accepted: %s: %v", line, err)
				}
			}
			// Returned metadata must be independent of later calls and parser behavior.
			if len(g.Optional) > 0 {
				g.Optional[0] = !g.Optional[0]
			}
			for i := range g.Flags {
				g.Flags[i] = "changed"
			}
			if _, err := NewRegistry().ParseLine(entry.Example, entry.Package); err != nil {
				t.Fatal(err)
			}
		})
	}
	if fixed < 100 {
		t.Fatalf("only %d fixed routes checked", fixed)
	}
	t.Logf("checked %d fixed routes for extra arguments", fixed)
}
func TestConstructedCanonicalTokens(t *testing.T) {
	for _, token := range []Token{
		&TokenMacroTH{Title: "", Section: "1", Extra3: "manual"},
		&TokenMacroIP{Designator: "two words", Nnn: "4"},
		&TokenMacroPn{Path: "/some path", Punct2: "."},
		&TokenMacroNT{Title: "Title without C"},
		&TokenRequestDs{Name: "S", StringVal: "  leading and trailing  "},
		&TokenRequestIf{Cond: "1", Anything: `.IP "two words" 4`},
	} {
		assertCanonicalToken(t, token, "man")
	}
	if _, err := CanonicalToken(&TokenMacroVS{Arg4: "5"}); !errors.Is(err, ErrArgumentEncoding) {
		t.Fatal(err)
	}
	if _, err := CanonicalToken(&TokenRequestDs{Name: "S", StringVal: `lost\"comment`}); !errors.Is(err, ErrArgumentEncoding) {
		t.Fatal(err)
	}
	if _, err := CanonicalToken((*TokenMacroTH)(nil)); !errors.Is(err, ErrArgumentEncoding) {
		t.Fatal(err)
	}
}
func FuzzArgumentBinding(f *testing.F) {
	for i, s := range []string{`T 1 "" x y`, `X (3 + 5) 2`, `1 .IP "two words" 4`, `S "unfinished string`, `'left'center'right'`, `a 1 b 2 c 3`, `! 'one two'one two' body`} {
		f.Add(s, uint8(i))
	}
	modes := []ArgumentGrammar{
		(&TokenMacroTH{}).ArgumentGrammar(), (&TokenRequestNr{}).ArgumentGrammar(),
		(&TokenRequestIf{}).ArgumentGrammar(), (&TokenRequestDs{}).ArgumentGrammar(),
		(&TokenRequestTl{}).ArgumentGrammar(), (&TokenRequestHcode{}).ArgumentGrammar(),
		(&TokenRequestSs{}).ArgumentGrammar(), (&TokenMacroPn{}).ArgumentGrammar(),
	}
	f.Fuzz(func(t *testing.T, source string, kind uint8) {
		if len(source) > 16384 {
			t.Skip()
		}
		g := modes[int(kind)%len(modes)]
		fields, err := BindArguments(source, g)
		if err != nil {
			return
		}
		if len(fields) != len(g.Optional) {
			t.Fatalf("field count %d", len(fields))
		}
		canonical, err := FormatArguments(fields, g)
		if err != nil {
			return
		}
		again, err := BindArguments(canonical, g)
		if err != nil || !reflect.DeepEqual(fields, again) {
			t.Fatalf("canonical fields changed: %q %#v %v", canonical, again, err)
		}
	})
}

func TestInvalidArgumentGrammars(t *testing.T) {
	for _, g := range []ArgumentGrammar{
		{Mode: "guess"}, {Mode: ArgumentsFixed, Minimum: -1},
		{Mode: ArgumentsJoined, Optional: []bool{true}, Rest: 2},
		{Mode: ArgumentsRaw, Optional: []bool{true, true}, Rest: 1},
		{Mode: ArgumentsRegister}, {Mode: ArgumentsConditional}, {Mode: ArgumentsDelimited},
		{Mode: ArgumentsFixed, Flags: map[int]string{10: "C"}},
		{Mode: ArgumentsFixed, Optional: []bool{true}, Numeric: []bool{true, true}},
		{Mode: ArgumentsFixed, Optional: []bool{true, false}, Numeric: []bool{true, true}, Minimum: 1},
		{Mode: ArgumentsVariadic, Optional: []bool{true}, Rest: 0, RepeatFrom: 1},
	} {
		if _, err := BindArguments("data", g); !errors.Is(err, ErrArgumentGrammar) {
			t.Fatalf("bind %#v: %v", g, err)
		}
		if _, err := FormatArguments(nil, g); !errors.Is(err, ErrArgumentGrammar) {
			t.Fatalf("format %#v: %v", g, err)
		}
	}
	for _, line := range []string{`.nr X 1 2 3 4`, `.ss 1 2 3 4 5`} {
		_, err := NewRegistry().ParseLine(line, "man")
		var extra *UnexpectedArgumentsError
		if !errors.As(err, &extra) || extra.Actual != 5 {
			t.Fatalf("extra count: %s: %v", line, err)
		}
	}
}
