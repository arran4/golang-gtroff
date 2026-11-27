package analyze_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/analyze"
)

func TestDeclarativeRegistryAndIsolation(t *testing.T) {
	config := `{"schemaVersion":1,"rules":[{"id":"project.has-heading","match":{"macro":".SH"},"feature":"project-heading","reason":"A heading deserves project-specific review","requires":["manual-review"]}]}`
	registry, err := analyze.ParseRules(strings.NewReader(config))
	if err != nil {
		t.Fatal(err)
	}
	report, err := analyze.Run(context.Background(), readSet(t, ".TH TOOL 1\n.SH SYNOPSIS\ntext\n"), analyze.Options{Registry: registry})
	if err != nil {
		t.Fatal(err)
	}
	if !report.HasRule("project.has-heading") || report.Counts.Features["project-heading"] != 1 || !report.HasRule("builtin.control") {
		t.Fatal("custom rule replaced built-ins")
	}
	rules := registry.Rules()
	rules[0].Requires[0] = "changed"
	rules[0].Match.Macro = ".UNKNOWN"
	if registry.Rules()[0].Match.Macro != ".SH" || registry.Rules()[0].Requires[0] != "manual-review" {
		t.Fatal("registry accessor aliases mutable rules")
	}
	value := true
	rule := analyze.Rule{ID: "project.opaque", Match: analyze.RuleMatch{Kind: groff.ASTLine, Opaque: &value}, Feature: "opaque-data", Reason: "Opaque input remains visible"}
	registry, err = analyze.NewRegistry(rule)
	if err != nil {
		t.Fatal(err)
	}
	value = false
	if !*registry.Rules()[0].Match.Opaque {
		t.Fatal("registry did not snapshot pointers")
	}
}
func TestRuleFileFailures(t *testing.T) {
	for _, config := range []string{
		`{}`, `{"schemaVersion":2}`, `{"schemaVersion":1,"unknown":true}`, `{"schemaVersion":1} {}`,
		`{"schemaVersion":1,"rules":[{"id":"builtin.control","match":{"macro":".SH"},"feature":"x","reason":"override"}]}`,
		`{"schemaVersion":1,"rules":[{"id":"project.x","match":{},"feature":"x","reason":"no conditions"}]}`,
		`{"schemaVersion":1,"rules":[{"id":"project.x","match":{"macro":"SH"},"feature":"x","reason":"bad spelling"}]}`,
		`{"schemaVersion":1,"rules":[{"id":"project.x","match":{"inlineKind":"made-up"},"feature":"x","reason":"bad kind"}]}`,
		strings.Repeat(" ", (64<<10)+1),
	} {
		if _, err := analyze.ParseRules(strings.NewReader(config)); !errors.Is(err, analyze.ErrRules) {
			t.Fatalf("invalid rules accepted: %q: %v", config, err)
		}
	}
	if _, err := analyze.ParseRules(nil); !errors.Is(err, analyze.ErrRules) {
		t.Fatal("nil rules reader accepted")
	}
}
func TestBuiltInRuleIDsCoverFindings(t *testing.T) {
	known := map[string]bool{}
	for _, rule := range analyze.BuiltinRules() {
		if known[rule.ID] || rule.Description == "" {
			t.Fatal("invalid builtin registry")
		}
		known[rule.ID] = true
	}
	report, err := analyze.Run(context.Background(), readSet(t, ".TH TOOL 1\n.ds X y\n.if 1 .B hi\n.sy no\n.so missing\n.TS\nl.\n.TE\n.MR printf 3\n.RS\n.RS\ntext \\fBbold\\fP\n.RE\n.RE\n"), analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range report.Findings {
		if !known[finding.Rule] {
			t.Fatal("unregistered built-in finding", finding.Rule)
		}
	}
}

func TestCustomQuestionsDoNotChangeBuiltInFacts(t *testing.T) {
	registry, err := analyze.NewRegistry(analyze.Rule{ID: "project.review", Match: analyze.RuleMatch{Macro: ".TH"}, Feature: "review", Reason: "A project requirement", Requires: []string{"evaluation"}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := analyze.Run(context.Background(), readSet(t, ".TH TOOL 1\n"), analyze.Options{Registry: registry})
	if err != nil {
		t.Fatal(err)
	}
	if report.Flags.RequiresEvaluation || !report.HasRule("project.review") {
		t.Fatal("declarative annotation changed a builtin syntax fact")
	}
}

func TestMacroQuestionsMatchOnePhysicalHeader(t *testing.T) {
	registry, err := analyze.NewRegistry(analyze.Rule{ID: "project.table", Match: analyze.RuleMatch{Macro: ".TS"}, Feature: "table-review", Reason: "Review table"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := analyze.Run(context.Background(), readSet(t, ".TS\nl.\n.TE\n"), analyze.Options{Registry: registry})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Rules["project.table"] != 1 {
		t.Fatal("macro question matched both group and header")
	}
}
