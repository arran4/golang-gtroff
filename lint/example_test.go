package lint_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arran4/golang-gtroff/lint"
	"github.com/arran4/golang-gtroff/source"
)

func ExampleRun() {
	set, err := source.Read(context.Background(), strings.NewReader(".TH TOOL 1\n.SH DESCRIPTION\nText.\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := lint.Run(context.Background(), []lint.Document{{Name: "tool.1", Sources: set}}, lint.Options{})
	if err != nil {
		panic(err)
	}
	for _, finding := range report.Findings {
		fmt.Printf("%s:%d: %s [%s]\n", finding.Span.Start.Filename, finding.Span.Start.Line, finding.Rule, finding.Severity)
	}
	fmt.Println("CI failed:", report.Failed())
	// Output:
	// tool.1:1: man.required-section [error]
	// tool.1:1: man.required-section [warning]
	// CI failed: true
}
func ExampleNewRegistry() {
	registry, err := lint.NewRegistry(lint.Rule{Info: lint.RuleInfo{ID: "project.notice", Severity: lint.Info, Category: lint.Heuristic, Description: "Request project review"}, Check: func(ctx context.Context, c *lint.Context, r lint.Reporter) error {
		return r.Emit(lint.Finding{Source: 1, Span: c.Sources().File(1).AST().Root().Span(), Message: "Review " + c.Name()})
	}})
	if err != nil {
		panic(err)
	}
	fmt.Println(registry.Rules()[len(registry.Rules())-1].ID)
	// Output: project.notice
}
func ExampleParsePolicy() {
	policy, err := lint.ParsePolicy(strings.NewReader(`{"schemaVersion":1,"policy":{"requiredSections":["NAME","ENVIRONMENT"],"disable":["man.name-pattern"]}}`))
	if err != nil {
		panic(err)
	}
	if err := lint.ValidatePolicy(policy, nil); err != nil {
		panic(err)
	}
	fmt.Println(policy.RequiredSections)
	// Output: [NAME ENVIRONMENT]
}
func ExampleReport_Write() {
	set, err := source.Read(context.Background(), strings.NewReader(".TH TOOL 1\n"), "tool.1", source.Options{})
	if err != nil {
		panic(err)
	}
	report, err := lint.Run(context.Background(), []lint.Document{{Name: "tool.1", Sources: set}}, lint.Options{Policy: lint.Policy{RequiredSections: []string{}}})
	if err != nil {
		panic(err)
	}
	if err := report.Write(os.Stdout, lint.Text); err != nil {
		panic(err)
	}
	// Output:
	// lint: documents=1 findings=0 suppressed=0 complete=true fail-on=error failed=false
	// document 1: tool.1 (profile=man input-invalid=false)
}
