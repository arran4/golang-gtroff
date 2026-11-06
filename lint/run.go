package lint

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Run applies policy to one or more explicit source snapshots. It does no I/O.
// Limits are cumulative across documents: defaults 256 documents, 200,000 AST
// nodes, 100,000 emitted findings (including suppressed), and 64 Mi work units.
// Operational errors return a sorted partial report with Complete=false. Input
// diagnostics remain separate; Failed applies the selected lint threshold and
// input validity, independently of the returned execution error.
func Run(ctx context.Context, documents []Document, options Options) (report *Report, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if options.MaxDocuments <= 0 {
		options.MaxDocuments = 256
	}
	if options.MaxNodes <= 0 {
		options.MaxNodes = 200000
	}
	if options.MaxFindings <= 0 {
		options.MaxFindings = 100000
	}
	if options.MaxWork <= 0 {
		options.MaxWork = 64 << 20
	}
	threshold := options.FailOn
	if threshold == "" {
		threshold = Error
		if options.Strict {
			threshold = Warning
		}
	}
	report = &Report{SchemaVersion: 1, Command: "lint", ReferenceCheck: "not-requested", Strict: options.Strict, Threshold: threshold, Documents: []DocumentInfo{}, Findings: []Finding{}, Suppressed: []Finding{}}
	defer func() {
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			report.Error = err.Error()
			report.Complete = false
		}
		sortFindings(report.Findings)
		sortFindings(report.Suppressed)
	}()
	if severityRank(threshold) == 0 && threshold != Never {
		return report, fmt.Errorf("%w: invalid failure threshold %q", ErrConfig, threshold)
	}
	if options.MaxNodes > 1000000 || options.MaxDocuments > 4096 || options.MaxFindings > 1000000 || options.MaxWork > 1<<30 {
		return report, fmt.Errorf("%w: limits exceed supported ceilings", ErrConfig)
	}
	registry := options.Registry
	if registry == nil {
		registry, err = NewRegistry()
		if err != nil {
			return report, err
		}
	}
	if err = ValidatePolicy(options.Policy, registry); err != nil {
		return report, err
	}
	report.Policy = clonePolicy(options.Policy)
	if options.Policy.ClosedWorld {
		report.ReferenceCheck = "pending"
	}
	if len(documents) == 0 {
		return report, fmt.Errorf("%w: expected at least one document", ErrConfig)
	}
	meter := &budget{ctx, options.MaxWork}
	remainingNodes := options.MaxNodes
	contexts := []*Context{}
	targets := map[string]bool{}
	closedComplete := true
	for i, document := range documents {
		if i >= options.MaxDocuments {
			return report, ErrLimit
		}
		if err = meter.spend(len(document.Name) + len(document.Root) + 1); err != nil {
			return report, err
		}
		info := DocumentInfo{Index: i + 1, Name: document.Name, Root: document.Root}
		if document.InputError != nil {
			info.InputError = document.InputError.Error()
			info.InputInvalid = true
		}
		if document.Sources == nil || len(document.Sources.Files()) == 0 {
			info.InputInvalid = true
			if info.InputError == "" {
				info.InputError = "no source snapshot"
			}
			report.Documents = append(report.Documents, info)
			closedComplete = false
			continue
		}
		if remainingNodes <= 0 {
			return report, ErrLimit
		}
		var c *Context
		c, err = makeContext(ctx, document, i+1, options.Policy, meter, remainingNodes)
		if c.analysis != nil {
			info.Profile = c.Profile()
			info.Sources = c.analysis.Sources
			info.Diagnostics = c.analysis.Diagnostics
			for _, s := range info.Sources {
				remainingNodes -= s.Nodes
				if !s.Complete {
					info.InputInvalid = true
					if info.InputError == "" {
						info.InputError = "source capture incomplete"
					}
				}
			}
			for _, d := range info.Diagnostics {
				if d.Severity == "error" || options.Strict {
					info.InputInvalid = true
				}
			}
		}
		report.Documents = append(report.Documents, info)
		if err != nil {
			return report, err
		}
		c.strict = options.Strict
		if info.InputInvalid || c.holes || !c.manScope || c.title == "" || c.manualSection == "" {
			closedComplete = false
		}
		if c.title != "" && c.manualSection != "" {
			targets[strings.ToLower(strings.TrimSpace(c.title)+"("+strings.TrimSpace(c.manualSection)+")")] = true
		}
		contexts = append(contexts, c)
	}
	if options.Policy.ClosedWorld {
		report.ReferenceCheck = "checked"
		if !closedComplete {
			report.ReferenceCheck = "skipped-incomplete"
		}
	}
	emitted := 0
	for _, c := range contexts {
		c.targets = targets
		c.closedWorldComplete = closedComplete
		suppressions, diagnostics, suppressErr := parseSuppressions(c, registry, meter)
		report.Diagnostics = append(report.Diagnostics, diagnostics...)
		if suppressErr != nil {
			return report, suppressErr
		}
		for _, rule := range registry.rules {
			if err = meter.spend(1); err != nil {
				return report, err
			}
			if !enabled(rule.Info, options) {
				continue
			}
			sink := &reporter{report: report, c: c, rule: rule.Info, meter: meter, suppressions: suppressions, emitted: &emitted, max: options.MaxFindings}
			if err = rule.Check(ctx, c, sink); err != nil {
				return report, fmt.Errorf("rule %s: %w", rule.Info.ID, err)
			}
		}
	}
	report.Complete = true
	return report, nil
}
func enabled(rule RuleInfo, o Options) bool {
	enabled := !rule.StrictOnly || o.Strict
	for _, id := range o.Policy.Enable {
		if id == rule.ID {
			enabled = true
		}
	}
	for _, id := range o.Policy.Disable {
		if id == rule.ID {
			enabled = false
		}
	}
	return enabled
}
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Document != b.Document {
			return a.Document < b.Document
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Span.Start.Offset != b.Span.Start.Offset {
			return a.Span.Start.Offset < b.Span.Start.Offset
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Message < b.Message
	})
	for i := range findings {
		findings[i].Index = i + 1
	}
}

type reporter struct {
	report       *Report
	c            *Context
	rule         RuleInfo
	meter        *budget
	suppressions []suppression
	emitted      *int
	max          int
}

func (r *reporter) Spend(n int) error { return r.meter.spend(n) }
func (r *reporter) Emit(f Finding) error {
	if err := r.Spend(len(f.Message) + len(f.Suggestion) + 1); err != nil {
		return err
	}
	if *r.emitted >= r.max {
		return ErrLimit
	}
	if f.Rule != "" && f.Rule != r.rule.ID {
		return fmt.Errorf("%w: callback supplied another rule ID", ErrConfig)
	}
	file := r.c.document.Sources.File(f.Source)
	if file == nil || f.Span.Start.Offset < 0 || f.Span.End.Offset < f.Span.Start.Offset || f.Span.End.Offset > len(file.AST().Source()) || f.Message == "" || len(f.Related) > 128 {
		return fmt.Errorf("%w: invalid finding source/span/message", ErrConfig)
	}
	if f.Span.Start.Filename != file.Name() || f.Span.End.Filename != file.Name() || f.Span.Start.Line < 1 || f.Span.End.Line < f.Span.Start.Line || f.Span.Start.Column < 1 || f.Span.End.Column < 1 {
		return fmt.Errorf("%w: invalid finding source positions", ErrConfig)
	}
	if f.Node != 0 && file.AST().Node(f.Node) == nil {
		return fmt.Errorf("%w: absent finding node", ErrConfig)
	}
	if f.Severity == "" {
		f.Severity = r.rule.Severity
	}
	if severity, ok := r.c.policy.Severity[r.rule.ID]; ok {
		f.Severity = severity
	}
	if severityRank(f.Severity) == 0 {
		return fmt.Errorf("%w: invalid finding severity", ErrConfig)
	}
	if f.Category == "" {
		f.Category = r.rule.Category
	}
	if !validCategory(f.Category) {
		return fmt.Errorf("%w: invalid finding category", ErrConfig)
	}
	for _, location := range f.Related {
		if err := r.Spend(len(location.Message) + 1); err != nil {
			return err
		}
		if location.Document < 1 || location.Document > len(r.report.Documents) {
			return fmt.Errorf("%w: invalid related document", ErrConfig)
		}
	}
	f.Rule = r.rule.ID
	f.SchemaVersion = 1
	f.Document = r.c.index
	f.Related = append([]Location(nil), f.Related...)
	f.Suppressed = false
	f.Suppression = ""
	for _, s := range r.suppressions {
		if err := r.Spend(1); err != nil {
			return err
		}
		if s.source == f.Source && s.rule == f.Rule && (s.nextLine == f.Span.Start.Line || s.nextLine == 0 && f.Span.Start.Offset >= s.start && f.Span.Start.Offset < s.end) {
			f.Suppressed = true
			f.Suppression = s.reason
			break
		}
	}
	*r.emitted++
	if f.Suppressed {
		r.report.Suppressed = append(r.report.Suppressed, f)
	} else {
		r.report.Findings = append(r.report.Findings, f)
	}
	return nil
}

func clonePolicy(p Policy) Policy {
	if p.RequiredSections != nil {
		p.RequiredSections = append([]string{}, p.RequiredSections...)
	}
	p.Enable = append([]string(nil), p.Enable...)
	p.Disable = append([]string(nil), p.Disable...)
	if p.Severity != nil {
		severity := map[string]Severity{}
		for id, s := range p.Severity {
			severity[id] = s
		}
		p.Severity = severity
	}
	return p
}
