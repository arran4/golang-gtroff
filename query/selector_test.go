package query

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParsedAndBuilderSelectorsAgree(t *testing.T) {
	cases := []struct {
		input   string
		builder Selector
	}{
		{`section[name="ENVIRONMENT"] definition`, Select(Section).Where(Name, Equal, "ENVIRONMENT").Descendant(Select(Definition))},
		{`source[source-id>=2]>document section > paragraph[normalized-text*='HOME directory']`, Select(Source).Where(SourceID, GreaterEqual, "2").Child(Select(Document)).Descendant(Select(Section)).Child(Select(Paragraph).Where(NormalizedText, Contains, "HOME directory"))},
		{`macro[macro='.B'][arg.0="TOOL_HOME"]`, Select(Macro).Where(MacroName, Equal, ".B").Where(Arg(0), Equal, "TOOL_HOME")},
		{`item[term^="雪"][line>=02]`, Select(Definition).Where(TermText, Prefix, "雪").Where(StartLine, GreaterEqual, "02")},
		{`line[opaque]`, Select(Line).Where(Opaque, Exists, "")},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			parsed, err := ParseSelector(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.builder.Validate(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(parsed, tc.builder) {
				t.Fatalf("different query representations: %#v %#v", parsed, tc.builder)
			}
		})
	}
	for _, tc := range []struct{ input, value string }{
		{`line[raw="literal\\path \"quoted\""]`, "literal\\path \"quoted\""},
		{`line[raw="invalid \xff"]`, "invalid \xff"},
		{`line[text='it\'s here']`, "it's here"},
	} {
		parsed, err := ParseSelector(tc.input)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.steps[0].predicates[0].value != tc.value {
			t.Fatal("quoted predicate value changed")
		}
	}
	base := Select(Section)
	_ = base.Where(Name, Equal, "first")
	_ = base.Child(Select(Definition))
	if len(base.steps) != 1 || len(base.steps[0].predicates) != 0 {
		t.Fatal("builder mutation leaked between queries")
	}
}
func TestSelectorErrorsAndBounds(t *testing.T) {
	for _, input := range []string{"", " ", "section >", "section + paragraph", "section,paragraph", "unknown-kind", "section[name]garbage", "section[name=]", "section[name==\"x\"]", "section[[name=x]]", "section[name=\"broken]", "section[name='bad\\q']", "section[line=2.5]", "section[line*=2]", "section[name>2]", "line[opaque='yes']", "line[absent=x]", "line[arg.-1=x]", strings.Repeat("x", (16<<10)+1), "line[text='\xff']", strings.Repeat("section ", 65)} {
		if _, err := ParseSelector(input); !errors.Is(err, ErrSelector) {
			t.Errorf("selector %q did not fail correctly: %v", input, err)
		}
	}
	for _, selector := range []Selector{{}, Select("made-up"), Select(Line).Where(Arg(-1), Equal, "x"), Select(Line).Where(Text, Operator("?="), "x"), Select(Line).Where(Text, Exists, "x")} {
		if !errors.Is(selector.Validate(), ErrSelector) {
			t.Fatal("invalid builder accepted")
		}
	}
}
func FuzzParseSelector(f *testing.F) {
	for _, input := range []string{`section[name="ENVIRONMENT"] definition`, `macro[arg.0='雪']`, ">", `line[text="\\"]`} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		s, err := ParseSelector(input)
		if err == nil {
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		}
	})
}
