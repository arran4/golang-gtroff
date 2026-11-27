package lint_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/lint"
	"github.com/arran4/golang-gtroff/source"
)

const good = ".TH TOOL 1\n.SH NAME\ntool \\- example tool\n.SH SYNOPSIS\n.B tool\n.SH DESCRIPTION\nDescription.\n"

func document(t *testing.T, text string) lint.Document {
	t.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(text), "input.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return lint.Document{Name: "input.1", Sources: set}
}
func run(t *testing.T, text string, o lint.Options) *lint.Report {
	t.Helper()
	r, err := lint.Run(context.Background(), []lint.Document{document(t, text)}, o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func findings(r *lint.Report, id string) []lint.Finding {
	var out []lint.Finding
	for _, f := range r.Findings {
		if f.Rule == id {
			out = append(out, f)
		}
	}
	return out
}
func TestGoodAndThresholds(t *testing.T) {
	for _, strict := range []bool{false, true} {
		r := run(t, good, lint.Options{Strict: strict})
		if !r.Complete || r.Failed() || len(r.Findings) != 0 {
			t.Fatalf("good page %+v", r)
		}
	}
	text := ".TH TOOL 1\n.SH DESCRIPTION\nText.\n"
	r := run(t, text, lint.Options{})
	fs := findings(r, "man.required-section")
	if len(fs) != 2 || fs[0].Message != "required section \"NAME\" was not found" || !r.Failed() {
		t.Fatalf("missing: %+v", fs)
	}
	r = run(t, text, lint.Options{FailOn: lint.Never})
	if r.Failed() {
		t.Fatal("never threshold failed policy")
	}
	r = run(t, good+".SH NAME\ntool \\- again\n", lint.Options{})
	if len(findings(r, "man.duplicate-section")) != 1 || r.Failed() {
		t.Fatal("warning threshold", r)
	}
	r = run(t, good+".SH NAME\ntool \\- again\n", lint.Options{Strict: true})
	if !r.Failed() {
		t.Fatal("strict warning threshold")
	}
}
func TestPolicyAndDefinitions(t *testing.T) {
	text := good + ".SH OPTIONS\n.TP\n.B --same\nFirst.\n.TP\n.B --same\nSecond.\n.SS OTHER\n.TP\n.B --same\nThird.\n.RS\n.TP\n.B --same\nFourth.\n.RE\n.IP\nDescription without tag.\n"
	r := run(t, text, lint.Options{})
	if fs := findings(r, "man.duplicate-definition"); len(fs) != 1 || fs[0].Span.Start.Line != 12 || len(fs[0].Related) != 1 {
		t.Fatalf("duplicate scopes %+v", fs)
	}
	if fs := findings(r, "man.empty-definition"); len(fs) != 0 {
		t.Fatalf("false positive %+v", fs)
	}
	r = run(t, ".TH TOOL 1\n", lint.Options{Policy: lint.Policy{RequiredSections: []string{}}})
	if len(r.Findings) != 0 {
		t.Fatalf("explicit empty policy %+v", r.Findings)
	}
	r = run(t, good+".SH OPTIONS\n.TP\n.B EMPTY\n", lint.Options{})
	if len(findings(r, "man.empty-definition")) != 1 {
		t.Fatal("empty definition absent", r.Findings)
	}
	r = run(t, good+".SH EMPTY\n", lint.Options{})
	if len(findings(r, "man.empty-section")) != 1 {
		t.Fatal("empty section absent", r.Findings)
	}
	r = run(t, strings.Replace(good, "tool \\- example tool", "tool only", 1), lint.Options{})
	if len(findings(r, "man.name-pattern")) != 1 {
		t.Fatal("bad name not found")
	}
}
func TestProfilesAndUncertainty(t *testing.T) {
	for _, text := range []string{".TL\nTitle\n.AU\nAuthor\n", ".Dd Today\n.Dt TOOL 1\n.Sh NAME\n.Nm tool\n"} {
		r := run(t, text, lint.Options{})
		if len(findings(r, "man.required-section")) != 0 || len(findings(r, "profile.man-policy")) != 1 {
			t.Fatalf("foreign false positive %+v", r)
		}
	}
	for _, text := range []string{".TH TOOL 1\n.so missing\n", ".TH TOOL 1\n.if 1 .SH NAME\n", ".TH TOOL 1\n.UNKNOWN data\n"} {
		r := run(t, text, lint.Options{})
		for _, f := range findings(r, "man.required-section") {
			if f.Severity != lint.Warning || f.Category != lint.Heuristic {
				t.Fatal("uncertainty claimed definitive", f)
			}
		}
		if r.Failed() {
			t.Fatal("permissive partial source failed", r)
		}
	}
	r := run(t, good+".SH MORE\n.UNKNOWN literal\n", lint.Options{})
	if len(findings(r, "man.empty-section")) != 0 || len(findings(r, "capability.unsupported")) != 0 {
		t.Fatal("unsupported false empty/default warning")
	}
	r = run(t, good+".UNKNOWN literal\n", lint.Options{Strict: true})
	if len(findings(r, "capability.unsupported")) == 0 {
		t.Fatal("strict capability warning absent")
	}
}
func TestSuppressionFilteringAndOpaqueComments(t *testing.T) {
	text := ".\\\" gtroff-lint-disable man.required-section\n.TH TOOL 1\n.SH DESCRIPTION\nText.\n"
	r := run(t, text, lint.Options{})
	if len(findings(r, "man.required-section")) != 0 || len(r.Suppressed) != 2 || r.Failed() {
		t.Fatalf("suppression %+v", r)
	}
	text = good + ".\\\" gtroff-lint-disable-next-line man.empty-section\n.SH EMPTY\n"
	r = run(t, text, lint.Options{})
	if len(r.Suppressed) != 1 {
		t.Fatal("next line", r)
	}
	text = good + ".\\\" gtroff-lint-disable man.empty-section\n.SH FIRST\n.\\\" gtroff-lint-enable man.empty-section\n.SH SECOND\n"
	r = run(t, text, lint.Options{})
	if len(r.Suppressed) != 1 || len(findings(r, "man.empty-section")) != 1 {
		t.Fatal("enable scopes", r)
	}
	text = ".de SAVED\n.\\\" gtroff-lint-disable all\n..\n.TH TOOL 1\n"
	r = run(t, text, lint.Options{})
	if len(r.Suppressed) != 0 || len(findings(r, "man.required-section")) != 3 {
		t.Fatal("stored suppression active", r)
	}
	r = run(t, good+".\\\" gtroff-lint-disable typo.rule\n", lint.Options{FailOn: lint.Never})
	if len(r.Diagnostics) != 1 || !r.Failed() {
		t.Fatal("malformed suppression silently discarded")
	}
	r = run(t, ".TH TOOL 1\n", lint.Options{Policy: lint.Policy{Disable: []string{"man.required-section"}}})
	if len(r.Findings) != 0 {
		t.Fatal("disabled rule emitted")
	}
}
func TestIncludesAndCollectionReferences(t *testing.T) {
	fsys := fstest.MapFS{"root.1": {Data: []byte(good + ".SH OPTIONS\n.so one.roff\n.so two.roff\n")}, "one.roff": {Data: []byte(".TP\n.B --same\nOne.\n")}, "two.roff": {Data: []byte(".TP\n.B --same\nTwo.\n")}}
	set, err := source.Load(context.Background(), fsys, "root.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := lint.Run(context.Background(), []lint.Document{{Name: "root.1", Sources: set}}, lint.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fs := findings(r, "man.duplicate-definition")
	if len(fs) != 1 || fs[0].Source != 3 || fs[0].Related[0].Source != 2 {
		t.Fatalf("include scope/provenance %+v", fs)
	}
	a := document(t, good+".MR other 1\n.MR external 3\n")
	b := document(t, strings.Replace(good, "TOOL", "OTHER", 1))
	r, err = lint.Run(context.Background(), []lint.Document{a, b}, lint.Options{Policy: lint.Policy{ClosedWorld: true}})
	if err != nil {
		t.Fatal(err)
	}
	if r.ReferenceCheck != "checked" {
		t.Fatal("missing reference-check status", r.ReferenceCheck)
	}
	fs = findings(r, "reference.local-target")
	if len(fs) != 1 || !strings.Contains(fs[0].Message, "external(3)") {
		t.Fatalf("closed references %+v", fs)
	}
	r, err = lint.Run(context.Background(), []lint.Document{a, b}, lint.Options{})
	if err != nil || len(findings(r, "reference.local-target")) != 0 {
		t.Fatal("external references failed open collection")
	}
	b.InputError = errors.New("incomplete collection")
	r, err = lint.Run(context.Background(), []lint.Document{a, b}, lint.Options{Policy: lint.Policy{ClosedWorld: true}})
	if err != nil || len(findings(r, "reference.local-target")) != 0 || !r.Failed() {
		t.Fatal("incomplete collection claimed missing target")
	}
	if r.ReferenceCheck != "skipped-incomplete" {
		t.Fatal("skip reason omitted", r.ReferenceCheck)
	}
}
func TestLimitsCancellationAndInputErrors(t *testing.T) {
	d := document(t, good)
	for _, o := range []lint.Options{{MaxNodes: 1}, {MaxWork: 1}, {MaxFindings: 1, Policy: lint.Policy{RequiredSections: []string{"A", "B"}}}, {MaxDocuments: 1}} {
		docs := []lint.Document{d}
		if o.MaxDocuments == 1 {
			docs = append(docs, d)
		}
		r, err := lint.Run(context.Background(), docs, o)
		if !errors.Is(err, lint.ErrLimit) || r.Complete || !r.Failed() {
			t.Fatalf("limit %+v %v", r, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := lint.Run(ctx, []lint.Document{d}, lint.Options{})
	if !errors.Is(err, context.Canceled) || r.Complete {
		t.Fatal("cancellation")
	}
	r, err = lint.Run(context.Background(), []lint.Document{{Name: "missing", InputError: errors.New("missing input")}, d}, lint.Options{FailOn: lint.Never})
	if err != nil || !r.Complete || !r.Failed() || len(r.Findings) != 0 || len(r.Documents) != 2 {
		t.Fatal("unreadable root generated phantom findings", r, err)
	}
	r = run(t, ".TH\n", lint.Options{FailOn: lint.Never, Policy: lint.Policy{Disable: []string{"man.required-section"}}})
	if !r.Failed() || len(r.Documents[0].Diagnostics) == 0 {
		t.Fatal("parser error hidden", r)
	}
}
func TestConfigRegistryAndDeterminism(t *testing.T) {
	for _, text := range []string{`{}`, `{"schemaVersion":2}`, `{"schemaVersion":1,"extra":1}`, `{"schemaVersion":1} {}`, strings.Repeat(" ", 65537)} {
		if _, err := lint.ParsePolicy(strings.NewReader(text)); !errors.Is(err, lint.ErrConfig) {
			t.Fatal("bad config accepted", err)
		}
	}
	policy, err := lint.ParsePolicy(strings.NewReader(`{"schemaVersion":1,"policy":{"requiredSections":[],"severity":{"man.required-section":"warning"}}}`))
	if err != nil || policy.RequiredSections == nil {
		t.Fatal("empty list lost", err)
	}
	if err = lint.ValidatePolicy(lint.Policy{Disable: []string{"typo"}}, nil); !errors.Is(err, lint.ErrConfig) {
		t.Fatal("unknown rule accepted")
	}
	registry, err := lint.NewRegistry(lint.Rule{Info: lint.RuleInfo{ID: "project.notice", Severity: lint.Info, Category: lint.Heuristic, Description: "Notice"}, Check: func(ctx context.Context, c *lint.Context, r lint.Reporter) error {
		sections := c.Sections()
		sections[0].Heading = "MUTATED"
		sections[0].ASTIDs[0] = 999
		return r.Emit(lint.Finding{Source: 1, Span: c.Sources().File(1).AST().Root().Span(), Message: c.Name()})
	}})
	if err != nil {
		t.Fatal(err)
	}
	d := document(t, good)
	o := lint.Options{Registry: registry}
	r, err := lint.Run(context.Background(), []lint.Document{d}, o)
	if err != nil || len(r.Findings) != 1 || r.Findings[0].Rule != "project.notice" {
		t.Fatal("custom", r, err)
	}
	again, err := lint.Run(context.Background(), []lint.Document{d}, o)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Fatal("not deterministic")
	}
}

func TestIndentConventionAndFunctionSynopsis(t *testing.T) {
	r := run(t, good+strings.Repeat(".RS\n", 7)+"Text.\n"+strings.Repeat(".RE\n", 7), lint.Options{})
	if fs := findings(r, "man.indent-depth"); len(fs) != 1 {
		t.Fatalf("indent convention %+v", fs)
	}
	text := strings.Replace(good, "TOOL 1", "TOOL 3", 1)
	text = strings.Replace(text, ".SH SYNOPSIS\n.B tool\n", "", 1)
	r = run(t, text, lint.Options{})
	if len(findings(r, "man.required-section")) != 0 {
		t.Fatal("function synopsis forced in permissive mode")
	}
	r = run(t, text, lint.Options{Strict: true})
	if len(findings(r, "man.required-section")) != 1 {
		t.Fatal("strict function synopsis absent")
	}
	r = run(t, ".TH TOOL 1\n.so missing\n", lint.Options{Policy: lint.Policy{Severity: map[string]lint.Severity{"man.required-section": lint.Error}}})
	for _, f := range findings(r, "man.required-section") {
		if f.Severity != lint.Error {
			t.Fatal("explicit severity override lost")
		}
	}
}
