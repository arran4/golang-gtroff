package analyze

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/source"
)

type nodeContext struct {
	file       *source.File
	node       *groff.ASTNode
	attrs      groff.ASTAttributes
	inline     groff.InlineToken
	isInline   bool
	active     bool
	capability Capability
	known      []string
}
type runner struct {
	diagnosticsSeen map[source.ID]map[string]bool
	report          *Report
	options         Options
	budget          *budget
	catalogue       map[string][]groff.CatalogueDefinition
	edges           map[source.ID]map[groff.NodeID]source.Edge
	requirements    map[string]bool
}

// Run inventories a supplied immutable source set. It never loads sources,
// executes macros or renders. Nil registries use only built-in detectors; custom
// declarative rules cannot override capability facts. Defaults are 200,000 AST
// nodes, 100,000 findings and 64 Mi work units. Limits/cancellation return a useful
// partial report plus an error; Complete describes enumeration, not input success.
func Run(ctx context.Context, set *source.Set, o Options) (report *Report, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if o.MaxNodes <= 0 {
		o.MaxNodes = 200000
	}
	if o.MaxFindings <= 0 {
		o.MaxFindings = 100000
	}
	if o.MaxWork <= 0 {
		o.MaxWork = 64 << 20
	}
	report = &Report{SchemaVersion: 1, CapabilityVersion: 1, Command: "analyze", SemanticProfile: "man", Findings: []Finding{}, Sources: []SourceInfo{}, Counts: Counts{Features: map[string]int{}, Controls: map[string]int{}, Escapes: map[string]int{}, Packages: map[string]int{}, Rules: map[string]int{}}}
	r := &runner{report: report, options: o, budget: &budget{ctx, o.MaxWork}, catalogue: map[string][]groff.CatalogueDefinition{}, edges: map[source.ID]map[groff.NodeID]source.Edge{}, requirements: map[string]bool{}}
	defer func() {
		report.WorkUnits = o.MaxWork - r.budget.remaining
		r.finish()
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
			report.Complete = false
		}
		if err != nil {
			report.Error = err.Error()
		}
	}()
	if set == nil {
		return report, fmt.Errorf("nil source set")
	}
	if err := r.budget.spend(1); err != nil {
		return report, err
	}
	for _, definition := range groff.CatalogueEntries() {
		r.catalogue[definition.Token] = append(r.catalogue[definition.Token], definition)
	}
	report.Diagnostics = set.Diagnostics()
	r.diagnosticsSeen = map[source.ID]map[string]bool{}
	for _, diagnostic := range report.Diagnostics {
		if r.diagnosticsSeen[diagnostic.Source] == nil {
			r.diagnosticsSeen[diagnostic.Source] = map[string]bool{}
		}
		r.diagnosticsSeen[diagnostic.Source][diagnosticKey(diagnostic.Diagnostic)] = true
	}
	report.Includes = set.Edges()
	for _, edge := range report.Includes {
		if r.edges[edge.From] == nil {
			r.edges[edge.From] = map[groff.NodeID]source.Edge{}
		}
		r.edges[edge.From][edge.Node] = edge
	}
	rules := o.Registry.Rules()
	nodes := 0
	for _, file := range set.Files() {
		if err := r.budget.spend(len(file.Name())); err != nil {
			return report, err
		}
		if err := r.budget.spend(len(file.AST().Source())); err != nil {
			return report, err
		}
		info := SourceInfo{ID: file.ID(), Name: file.Name(), Depth: file.Depth(), ParentEdge: file.ParentEdge(), Bytes: len(file.AST().Source()), Lines: physicalLines(file.AST().Source()), Complete: file.AST().SourceComplete()}
		if !info.Complete {
			report.Flags.HasPartialSource = true
		}
		var walkErr error
		file.AST().Root().Walk(func(node *groff.ASTNode) groff.WalkAction {
			if nodes >= o.MaxNodes {
				walkErr = ErrLimit
				return groff.WalkStop
			}
			nodes++
			info.Nodes++
			if err := r.budget.spend(1); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			if node.Kind() == groff.ASTLine {
				if activeNode(node) {
					addProfileEvidence(&info.Profile, file, node, r.catalogue[node.Attributes().Name])
				}
			}
			if node.Kind() == groff.ASTBlock {
				depth := 1
				for p := node.Parent(); p != nil; p = p.Parent() {
					if p.Kind() == groff.ASTBlock {
						depth++
					}
				}
				if depth > info.MaxBlockDepth {
					info.MaxBlockDepth = depth
				}
			}
			return groff.WalkContinue
		})
		info.Profile = resolveProfile(info.Profile.Evidence)
		report.Sources = append(report.Sources, info)
		if walkErr != nil {
			return report, walkErr
		}
		semantics, semanticDiags, err := indexSemantics(file, info.Profile, o.MaxNodes)
		r.appendDiagnostics(file.ID(), semanticDiags)
		if err != nil {
			return report, err
		}
		var commentLine groff.NodeID
		file.AST().Root().Walk(func(node *groff.ASTNode) groff.WalkAction {
			if err := r.budget.spend(1); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			attrs := node.Attributes()
			inline, isInline := node.Inline()
			active := activeNode(node)
			if node.Kind() == groff.ASTLine {
				commentLine = 0
			}
			if isInline && node.Parent() != nil && node.Parent().ID() == commentLine {
				active = false
			}
			c := nodeContext{file: file, node: node, attrs: attrs, inline: inline, isInline: isInline, active: active, capability: Capability{Syntax: Parsed, Semantics: NotAssessed, Rendering: NotAssessed}}
			if opaqueNode(node) {
				c.capability.Syntax = Opaque
			}
			if node.Kind() == groff.ASTLine && attrs.Name != "" && attrs.TokenKind != groff.KindComment {
				for _, entry := range r.catalogue[attrs.Name] {
					if !contains(c.known, entry.Package) {
						c.known = append(c.known, entry.Package)
					}
				}
				sort.Strings(c.known)
				if c.capability.Syntax != Opaque && (attrs.TokenKind == groff.KindUnknown || hasError(node.Diagnostics())) {
					c.capability.Syntax = Unknown
					if len(c.known) > 0 {
						c.capability.Syntax = Recognized
					}
				}
			}
			c.capability.Semantics = semantics[node.ID()]
			if c.capability.Semantics == "" {
				c.capability.Semantics = NotAssessed
			}
			if !active {
				c.capability.Semantics = NotAssessed
			}
			if isInline {
				c.capability.Semantics = inlineSemantics(c, semantics)
			}
			if isInline && node.Parent() != nil && inline.Kind == groff.InlineEscape && (inline.Name == "\"" || inline.Name == "#") {
				commentLine = node.Parent().ID()
			}
			if err := r.builtins(c); err != nil {
				walkErr = err
				return groff.WalkStop
			}
			for _, rule := range rules {
				if err := r.budget.spend(1); err != nil {
					walkErr = err
					return groff.WalkStop
				}
				if matchRule(rule.Match, c) {
					if err := r.emit(c, rule.ID, rule.Feature, attrs.Name, rule.Reason, rule.Requires, ""); err != nil {
						walkErr = err
						return groff.WalkStop
					}
				}
			}
			return groff.WalkContinue
		})
		if walkErr != nil {
			return report, walkErr
		}
	}
	report.Complete = true
	return report, nil
}
func (r *runner) emit(c nodeContext, rule, feature, name, reason string, requires []string, target string) error {
	if err := r.budget.spend(len(reason) + len(name) + len(target) + 1); err != nil {
		return err
	}
	if len(r.report.Findings) >= r.options.MaxFindings {
		return ErrLimit
	}
	finding := Finding{SchemaVersion: 1, Rule: rule, Feature: feature, Name: name, Source: c.file.ID(), Node: c.node.ID(), Span: c.node.Span(), Package: c.attrs.Package, KnownPackages: append([]string(nil), c.known...), Active: c.active, Capability: c.capability, Reason: reason, Requires: append([]string(nil), requires...), Target: target}
	if c.node.Parent() != nil {
		finding.Parent = c.node.Parent().ID()
	}
	r.report.Findings = append(r.report.Findings, finding)
	r.report.Counts.Features[feature]++
	r.report.Counts.Rules[rule]++
	if strings.HasPrefix(rule, "builtin.") {
		flags := &r.report.Flags
		if c.capability.Syntax == Opaque {
			flags.HasOpaque = true
		}
		switch feature {
		case "control":
			r.report.Counts.Controls[name]++
			pkg := c.attrs.Package
			if pkg == "" {
				pkg = "unknown"
			}
			r.report.Counts.Packages[pkg]++
			if c.capability.Syntax == Unknown {
				flags.HasUnknown = true
			}
		case "escape":
			r.report.Counts.Escapes[name]++
			if c.inline.Kind == groff.InlineStringRef || c.inline.Kind == groff.InlineRegisterRef {
				flags.HasReferences = true
			}
		case "include":
			flags.HasInclude = true
		case "macro-definition", "string-definition", "register-definition", "definition-update":
			flags.HasDefinitions = true
		case "conditional":
			flags.HasConditional = true
		case "side-effect-syntax":
			flags.HasSideEffectSyntax = true
		case "preprocessor":
			flags.UsesPreprocessor = true
		case "reference":
			flags.HasReferences = true
		}
	}

	for _, requirement := range requires {
		if c.active {
			r.requirements[requirement] = true
			if requirement == "evaluation" && strings.HasPrefix(rule, "builtin.") {
				r.report.Flags.RequiresEvaluation = true
			}
		}
	}
	return nil
}
func (r *runner) finish() {
	var evidence []Evidence
	for _, file := range r.report.Sources {
		evidence = append(evidence, file.Profile.Evidence...)
	}
	r.report.Profile = resolveProfile(evidence)
	for requirement := range r.requirements {
		r.report.Requirements = append(r.report.Requirements, requirement)
	}
	sort.Strings(r.report.Requirements)
	sort.SliceStable(r.report.Findings, func(i, j int) bool {
		a, b := r.report.Findings[i], r.report.Findings[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Span.Start.Offset != b.Span.Start.Offset {
			return a.Span.Start.Offset < b.Span.Start.Offset
		}
		return a.Rule < b.Rule
	})
	sort.SliceStable(r.report.Diagnostics, func(i, j int) bool {
		a, b := r.report.Diagnostics[i], r.report.Diagnostics[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Position.Offset < b.Position.Offset
	})
}
func activeNode(node *groff.ASTNode) bool {
	line := node
	for line.Parent() != nil && line.Kind() == groff.ASTInline {
		line = line.Parent()
	}
	if line.Attributes().Opaque || line.Attributes().TokenKind == groff.KindComment {
		return false
	}
	for p := line.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == groff.ASTBlock && p.Attributes().Opaque && line.Span().Start.Offset > p.Span().Start.Offset {
			return false
		}
	}
	return true
}
func hasError(diagnostics []groff.Diagnostic) bool {
	for _, d := range diagnostics {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func matchRule(match RuleMatch, c nodeContext) bool {
	if match.Kind != "" && match.Kind != c.node.Kind() {
		return false
	}
	if match.Macro != "" && match.Kind == "" && c.node.Kind() != groff.ASTLine {
		return false
	}
	if match.Macro != "" && match.Macro != c.attrs.Name {
		return false
	}
	if match.Package != "" && match.Package != c.attrs.Package {
		return false
	}
	if match.InlineKind != "" && (!c.isInline || c.inline.Kind != match.InlineKind) {
		return false
	}
	if match.Opaque != nil && *match.Opaque != (c.capability.Syntax == Opaque) {
		return false
	}
	return true
}

func diagnosticKey(d groff.Diagnostic) string {
	return strconv.Itoa(d.Position.Offset) + ":" + d.Severity + ":" + d.Message
}
func (r *runner) appendDiagnostics(id source.ID, diagnostics []groff.Diagnostic) {
	if r.diagnosticsSeen[id] == nil {
		r.diagnosticsSeen[id] = map[string]bool{}
	}
	seen := r.diagnosticsSeen[id]
	for _, d := range diagnostics {
		key := diagnosticKey(d)
		if !seen[key] {
			r.report.Diagnostics = append(r.report.Diagnostics, source.Diagnostic{Source: id, Code: "semantic", Diagnostic: d})
			seen[key] = true
		}
	}
}

func physicalLines(source string) int {
	lines := strings.Count(source, "\n")
	if len(source) > 0 && !strings.HasSuffix(source, "\n") {
		lines++
	}
	return lines
}
