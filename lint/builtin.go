package lint

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

func builtinRules() []Rule {
	return []Rule{
		{RuleInfo{"man.required-section", Error, Structural, "Required sections are present in inspected man source", false}, requiredSections},
		{RuleInfo{"man.duplicate-section", Warning, Structural, "Required section headings are not repeated", false}, duplicateSections},
		{RuleInfo{"man.empty-section", Warning, Heuristic, "Section bodies contain modeled content", false}, emptySections},
		{RuleInfo{"man.name-pattern", Warning, Heuristic, "NAME contains a name, separator and short description", false}, namePattern},
		{RuleInfo{"man.empty-definition", Warning, Heuristic, "Tagged definitions contain a term and description", false}, emptyDefinitions},
		{RuleInfo{"man.duplicate-definition", Warning, Heuristic, "Simple definition terms are unique within a structural scope", false}, duplicateDefinitions},
		{RuleInfo{"man.indent-depth", Warning, Heuristic, "Indent nesting stays within the configured convention", false}, indentDepth},
		{RuleInfo{"source.unresolved-include", Warning, Structural, "Requested local source was available to inspection", false}, unresolvedIncludes},
		{RuleInfo{"reference.local-target", Warning, Semantic, "Explicit manual references resolve in a declared closed collection", false}, localTargets},
		{RuleInfo{"capability.unsupported", Warning, Heuristic, "Active syntax has unsupported optional man semantics", true}, unsupportedCapabilities},
		{RuleInfo{"profile.man-policy", Info, Heuristic, "Man conventions are applied only to compatible profile evidence", false}, profilePolicy},
	}
}
func required(c *Context) []string {
	if c.policy.RequiredSections != nil {
		return c.policy.RequiredSections
	}
	out := []string{"NAME", "DESCRIPTION"}
	if c.manualSection == "1" || c.manualSection == "8" || c.strict && (c.manualSection == "2" || c.manualSection == "3") {
		out = append(out, "SYNOPSIS")
	}
	return out
}
func sectionFinding(s Section, message string) Finding {
	f := Finding{Source: s.Source, Span: s.Span, Message: message}
	if len(s.ASTIDs) > 0 {
		f.Node = s.ASTIDs[0]
	}
	return f
}
func definitionFinding(d Definition, message string) Finding {
	f := Finding{Source: d.Source, Span: d.Span, Message: message}
	if len(d.ASTIDs) > 0 {
		f.Node = d.ASTIDs[0]
	}
	return f
}
func requiredSections(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	seen := map[string]bool{}
	for _, s := range c.sections {
		if err := r.Spend(len(s.Heading) + 1); err != nil {
			return err
		}
		if s.Kind == man.Section {
			seen[normalizeHeading(s.Heading)] = true
		}
	}
	for _, name := range required(c) {
		if err := r.Spend(len(name) + 1); err != nil {
			return err
		}
		if seen[normalizeHeading(name)] {
			continue
		}
		f := rootFinding(c, fmt.Sprintf("required section %q was not found", name))
		f.Suggestion = "Add the section or configure requiredSections for this document policy."
		if name == "SYNOPSIS" && c.policy.RequiredSections == nil {
			f.Severity = Warning
		}
		if c.holes {
			f.Severity = Warning
			f.Category = Heuristic
			f.Message += "; incomplete sources or unevaluated syntax may supply it"
		}
		if err := r.Emit(f); err != nil {
			return err
		}
	}
	return nil
}
func duplicateSections(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	wanted := map[string]bool{}
	for _, s := range required(c) {
		wanted[normalizeHeading(s)] = true
	}
	seen := map[string]Section{}
	for _, s := range c.sections {
		if err := r.Spend(len(s.Heading) + 1); err != nil {
			return err
		}
		key := normalizeHeading(s.Heading)
		if s.Kind != man.Section || !wanted[key] {
			continue
		}
		if first, ok := seen[key]; ok {
			f := sectionFinding(s, fmt.Sprintf("required section %q repeats", s.Heading))
			f.Related = []Location{{c.index, first.Source, first.Span, "first heading"}}
			if err := r.Emit(f); err != nil {
				return err
			}
		} else {
			seen[key] = s
		}
	}
	return nil
}
func emptySections(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	for _, s := range c.sections {
		if err := r.Spend(len(s.Text) + 1); err != nil {
			return err
		}
		if !s.Unsupported && strings.TrimSpace(s.Text) == "" && !c.includeWithin(s.Source, s.Span.Start.Offset, s.Span.End.Offset) {
			f := sectionFinding(s, "section has no modeled body content")
			f.Suggestion = "Add content or remove the empty heading."
			if err := r.Emit(f); err != nil {
				return err
			}
		}
	}
	return nil
}
func namePattern(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	for _, s := range c.sections {
		if s.Kind != man.Section || normalizeHeading(s.Heading) != "NAME" || s.Unsupported || c.includeWithin(s.Source, s.Span.Start.Offset, s.Span.End.Offset) {
			continue
		}
		if err := r.Spend(len(s.Text) + 1); err != nil {
			return err
		}
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		valid := false
		for i, ch := range text {
			if (ch == '-' || ch == '−' || ch == '–') && i > 0 {
				before, _ := utf8.DecodeLastRuneInString(text[:i])
				after := text[i+len(string(ch)):]
				if unicode.IsSpace(before) && len(after) > 0 {
					next, _ := utf8.DecodeRuneInString(after)
					if unicode.IsSpace(next) && strings.TrimSpace(after) != "" {
						valid = true
						break
					}
				}
			}
		}
		if !valid {
			f := sectionFinding(s, "NAME does not match the conventional name - description pattern")
			f.Suggestion = `Use a name followed by \- and a short description, or suppress this heuristic for an intentional convention.`
			if err := r.Emit(f); err != nil {
				return err
			}
		}
	}
	return nil
}
func emptyDefinitions(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	for _, d := range c.definitions {
		if err := r.Spend(len(d.Term) + len(d.Text) + 1); err != nil {
			return err
		}
		if d.Unsupported || c.includeWithin(d.Source, d.Span.Start.Offset, d.Span.End.Offset) {
			continue
		}
		message := ""
		if d.ExpectedTerm && strings.TrimSpace(d.Term) == "" {
			message = "tagged paragraph has no modeled term"
		} else if strings.TrimSpace(d.Term) != "" && strings.TrimSpace(d.Text) == "" {
			message = "definition has no modeled description"
		}
		if message != "" {
			f := definitionFinding(d, message)
			f.Suggestion = "Add the missing term or description, or suppress an intentional label-only entry."
			if err := r.Emit(f); err != nil {
				return err
			}
		}
	}
	return nil
}
func duplicateDefinitions(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	seen := map[string]Definition{}
	for _, d := range c.definitions {
		if err := r.Spend(len(d.Term) + len(d.Scope) + 1); err != nil {
			return err
		}
		term := normalizeTerm(d.Term)
		if d.Unsupported || !d.SimpleTerm || term == "" {
			continue
		}
		key := d.Scope + "\x00" + term
		if first, ok := seen[key]; ok {
			f := definitionFinding(d, fmt.Sprintf("definition term %q repeats in this scope", term))
			f.Related = []Location{{c.index, first.Source, first.Span, "first definition"}}
			f.Suggestion = "Combine duplicate descriptions, rename the term, or suppress an intentional repetition."
			if err := r.Emit(f); err != nil {
				return err
			}
		} else {
			seen[key] = d
		}
	}
	return nil
}
func indentDepth(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	limit := c.policy.MaxIndentDepth
	if limit == 0 {
		limit = 6
	}
	for _, file := range c.document.Sources.Files() {
		model := c.models[file.ID()]
		if model == nil {
			continue
		}
		depth := map[man.ID]int{}
		for _, node := range model.Nodes {
			if err := r.Spend(1); err != nil {
				return err
			}
			depth[node.ID] = depth[node.Parent]
			if node.Kind != man.Indent {
				continue
			}
			depth[node.ID]++
			if depth[node.ID] == limit+1 {
				f := Finding{Source: file.ID(), Span: node.Origin.Span, Message: fmt.Sprintf("indent nesting exceeds policy depth %d", limit)}
				if len(node.Origin.ASTIDs) > 0 {
					f.Node = node.Origin.ASTIDs[0]
				}
				if err := r.Emit(f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func unresolvedIncludes(_ context.Context, c *Context, r Reporter) error {
	for _, edge := range c.document.Sources.Edges() {
		if err := r.Spend(len(edge.Target) + 1); err != nil {
			return err
		}
		if edge.State == "resolved" || edge.State == "disabled" {
			continue
		}
		f := Finding{Source: edge.From, Node: edge.Node, Span: edge.Span, Message: fmt.Sprintf("include %q could not be inspected (%s)", edge.Target, edge.State), Suggestion: "Provide the allowed local source or explicitly disable includes for a single-file policy."}
		if err := r.Emit(f); err != nil {
			return err
		}
	}
	return nil
}
func localTargets(_ context.Context, c *Context, r Reporter) error {
	if !c.policy.ClosedWorld || !c.closedWorldComplete {
		return nil
	}
	for _, feature := range c.analysis.Findings {
		if err := r.Spend(len(feature.Target) + 1); err != nil {
			return err
		}
		if feature.Feature != "reference" || feature.Name != "manual" || !feature.Active || feature.Target == "" {
			continue
		}
		target := strings.ToLower(strings.TrimSpace(feature.Target))
		if !strings.Contains(target, "(") || !strings.HasSuffix(target, ")") || strings.ContainsAny(target, `\`) {
			continue
		}
		if !c.targets[target] {
			f := Finding{Source: feature.Source, Node: feature.Node, Span: feature.Span, Message: fmt.Sprintf("manual reference %q has no title/section target in the declared closed collection", feature.Target), Suggestion: "Include the target document, correct the reference, or use an open collection policy."}
			if err := r.Emit(f); err != nil {
				return err
			}
		}
	}
	return nil
}
func unsupportedCapabilities(_ context.Context, c *Context, r Reporter) error {
	if !c.manScope {
		return nil
	}
	seen := map[source.ID]map[groff.NodeID]bool{}
	for _, feature := range c.analysis.Findings {
		if err := r.Spend(1); err != nil {
			return err
		}
		if !feature.Active || feature.Capability.Semantics != analyze.Unsupported || feature.Name == ".so" || feature.Name == ".mso" {
			continue
		}
		if seen[feature.Source] == nil {
			seen[feature.Source] = map[groff.NodeID]bool{}
		}
		if seen[feature.Source][feature.Node] {
			continue
		}
		seen[feature.Source][feature.Node] = true
		f := Finding{Source: feature.Source, Node: feature.Node, Span: feature.Span, Message: fmt.Sprintf("%s %q has unsupported optional man semantics: %s", feature.Feature, feature.Name, feature.Reason), Suggestion: "Inspect the preserved source and choose an interpreter appropriate to this feature."}
		if err := r.Emit(f); err != nil {
			return err
		}
	}
	return nil
}
func profilePolicy(_ context.Context, c *Context, r Reporter) error {
	if c.manScope {
		return nil
	}
	f := rootFinding(c, "man section/definition conventions were skipped for profile "+c.analysis.Profile.Name)
	return r.Emit(f)
}
func (c *Context) includeWithin(id source.ID, start, end int) bool {
	starts := c.includeStarts[id]
	at := sort.SearchInts(starts, start)
	return at < len(starts) && starts[at] < end
}
