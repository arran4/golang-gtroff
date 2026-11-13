package analyze_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleRun() {
	set, err := source.Read(context.Background(), strings.NewReader(".TH TOOL 1\n.if 1 .B conditional\n.so common.roff\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := analyze.Run(context.Background(), set, analyze.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println("profile:", report.Profile.Name)
	fmt.Println("requires evaluation:", report.Flags.RequiresEvaluation)
	for _, finding := range report.Findings {
		if finding.Feature == "include" {
			fmt.Printf("%s:%d: %s [%s]\n", finding.Span.Start.Filename, finding.Span.Start.Line, finding.Target, finding.Capability.Resolution)
		}
	}
	// Output:
	// profile: man
	// requires evaluation: true
	// tool.1:3: common.roff [unresolved]
}
func ExampleNewRegistry() {
	registry, err := analyze.NewRegistry(analyze.Rule{ID: "project.has-title", Match: analyze.RuleMatch{Macro: ".TH"}, Feature: "project-title", Reason: "Project-specific title review"})
	if err != nil {
		panic(err)
	}
	set, err := source.Read(context.Background(), strings.NewReader(".TH TOOL 1\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := analyze.Run(context.Background(), set, analyze.Options{Registry: registry})
	if err != nil {
		panic(err)
	}
	fmt.Println(report.HasRule("project.has-title"))
	// Output: true
}
func ExampleParseRules() {
	registry, err := analyze.ParseRules(strings.NewReader(`{"schemaVersion":1,"rules":[{"id":"project.has-table","match":{"macro":".TS"},"feature":"project-table","reason":"Inspect captured table source"}]}`))
	if err != nil {
		panic(err)
	}
	fmt.Println(registry.Rules()[0].ID)
	// Output: project.has-table
}
func ExampleReport_Write() {
	set, err := source.Read(context.Background(), strings.NewReader("plain text\n"), "text.roff", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := analyze.Run(context.Background(), set, analyze.Options{})
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	if err := report.Write(&out, analyze.JSON, false); err != nil {
		panic(err)
	}
	fmt.Println(strings.Contains(out.String(), `"capabilityVersion": 1`))
	// Output: true
}
