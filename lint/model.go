// Package lint applies structural documentation policy to explicit roff source
// snapshots. Parser diagnostics and feature classification remain separate data.
// Built-in checks are bounded and non-executing; registered Go callbacks are
// trusted host code and must honor context and Reporter budgets.
package lint

import (
	"context"
	"errors"
	"fmt"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/source"
)

type Severity string

const (
	Info    Severity = "info"
	Warning Severity = "warning"
	Error   Severity = "error"
	Never   Severity = "never"
)

type Category string

const (
	Structural Category = "structural"
	Semantic   Category = "semantic"
	Heuristic  Category = "heuristic"
)

// Document labels a source set without granting I/O capabilities. InputError can
// retain the source loader's classified error alongside its captured snapshot.
// Name/Root are provenance labels only; Run never opens either path.
type Document struct {
	Name, Root string
	Sources    *source.Set
	InputError error
}

type Location struct {
	Document int        `json:"document"`
	Source   source.ID  `json:"source"`
	Span     groff.Span `json:"span"`
	Message  string     `json:"message,omitempty"`
}
type Finding struct {
	SchemaVersion int          `json:"schemaVersion"`
	Index         int          `json:"index"`
	Document      int          `json:"document"`
	Rule          string       `json:"rule"`
	Severity      Severity     `json:"severity"`
	Category      Category     `json:"category"`
	Source        source.ID    `json:"source"`
	Node          groff.NodeID `json:"node,omitempty"`
	Span          groff.Span   `json:"span"`
	Message       string       `json:"message"`
	Suggestion    string       `json:"suggestion,omitempty"`
	Related       []Location   `json:"related,omitempty"`
	Suppressed    bool         `json:"suppressed,omitempty"`
	Suppression   string       `json:"suppression,omitempty"`
}

type DocumentInfo struct {
	Index        int                  `json:"index"`
	Name         string               `json:"name"`
	Root         string               `json:"root,omitempty"`
	Profile      analyze.Profile      `json:"profile"`
	Sources      []analyze.SourceInfo `json:"sources"`
	Diagnostics  []source.Diagnostic  `json:"diagnostics,omitempty"`
	InputError   string               `json:"inputError,omitempty"`
	InputInvalid bool                 `json:"inputInvalid"`
}

// Report.Complete describes policy enumeration, not source EOF or rendering.
// Suppressed findings are retained separately so suppression never erases input
// evidence or changes parser diagnostics. Ordinals are one-based after sorting.
type Report struct {
	Policy Policy `json:"policy"`
	// ReferenceCheck is not-requested, pending, checked or skipped-incomplete.
	ReferenceCheck string             `json:"referenceCheck"`
	SchemaVersion  int                `json:"schemaVersion"`
	Command        string             `json:"command"`
	Complete       bool               `json:"complete"`
	Strict         bool               `json:"strict"`
	Threshold      Severity           `json:"threshold"`
	Error          string             `json:"error,omitempty"`
	Documents      []DocumentInfo     `json:"documents"`
	Findings       []Finding          `json:"findings"`
	Suppressed     []Finding          `json:"suppressed"`
	Diagnostics    []ConfigDiagnostic `json:"diagnostics,omitempty"`
}
type ConfigDiagnostic struct {
	Document int        `json:"document"`
	Source   source.ID  `json:"source"`
	Span     groff.Span `json:"span"`
	Message  string     `json:"message"`
}

func (r *Report) Failed() bool {
	if r == nil {
		return true
	}
	if !r.Complete || r.Error != "" || len(r.Diagnostics) > 0 {
		return true
	}
	for _, doc := range r.Documents {
		if doc.InputInvalid {
			return true
		}
	}
	if r.Threshold == Never {
		return false
	}
	for _, finding := range r.Findings {
		if severityRank(finding.Severity) >= severityRank(r.Threshold) {
			return true
		}
	}
	return false
}
func severityRank(severity Severity) int {
	switch severity {
	case Info:
		return 1
	case Warning:
		return 2
	case Error:
		return 3
	}
	return 0
}

type Section struct {
	Kind          man.Kind
	Source        source.ID
	ID            man.ID
	Span          groff.Span
	Heading, Text string
	Unsupported   bool
	ASTIDs        []groff.NodeID
}
type Definition struct {
	Source                                source.ID
	ID                                    man.ID
	Span                                  groff.Span
	Term, Text, Scope                     string
	ExpectedTerm, SimpleTerm, Unsupported bool
	ASTIDs                                []groff.NodeID
}

// Context exposes immutable source and caller-owned snapshots of useful man
// structure. Callback rules cannot edit the original source through these APIs.
type Context struct {
	document                Document
	index                   int
	analysis                *analyze.Report
	models                  map[source.ID]*man.Document
	sections                []Section
	definitions             []Definition
	scopes                  map[source.ID][]scope
	edges                   map[source.EdgeID]source.Edge
	includeStarts           map[source.ID][]int
	manScope, holes, strict bool
	title, manualSection    string
	closedWorldComplete     bool
	targets                 map[string]bool
	policy                  Policy
}

func (c *Context) Name() string         { return c.document.Name }
func (c *Context) Sources() *source.Set { return c.document.Sources }
func (c *Context) Profile() analyze.Profile {
	profile := c.analysis.Profile
	profile.Evidence = append([]analyze.Evidence(nil), profile.Evidence...)
	return profile
}
func (c *Context) Sections() []Section {
	out := append([]Section(nil), c.sections...)
	for i := range out {
		out[i].ASTIDs = append([]groff.NodeID(nil), out[i].ASTIDs...)
	}
	return out
}
func (c *Context) Definitions() []Definition {
	out := append([]Definition(nil), c.definitions...)
	for i := range out {
		out[i].ASTIDs = append([]groff.NodeID(nil), out[i].ASTIDs...)
	}
	return out
}
func (c *Context) IsMan() bool { return c.manScope }

// Reporter validates findings and charges explicit custom-rule work. Trusted Go
// callbacks must periodically call Spend or honor ctx; no script sandbox is implied.
type Reporter interface {
	Emit(Finding) error
	Spend(int) error
}
type RuleInfo struct {
	ID          string   `json:"id"`
	Severity    Severity `json:"severity"`
	Category    Category `json:"category"`
	Description string   `json:"description"`
	StrictOnly  bool     `json:"strictOnly,omitempty"`
}
type Rule struct {
	Info  RuleInfo
	Check func(context.Context, *Context, Reporter) error
}
type Registry struct{ rules []Rule }

var (
	ErrConfig = errors.New("invalid lint configuration")
	ErrLimit  = errors.New("lint resource limit exceeded")
)

func NewRegistry(extra ...Rule) (*Registry, error) {
	rules := append(builtinRules(), extra...)
	if len(rules) > 128 {
		return nil, fmt.Errorf("%w: at most 128 rules", ErrConfig)
	}
	seen := map[string]bool{}
	for _, rule := range rules {
		if !validID(rule.Info.ID) || seen[rule.Info.ID] || rule.Check == nil || severityRank(rule.Info.Severity) == 0 || !validCategory(rule.Info.Category) || rule.Info.Description == "" {
			return nil, fmt.Errorf("%w: invalid or duplicate rule %q", ErrConfig, rule.Info.ID)
		}
		seen[rule.Info.ID] = true
	}
	return &Registry{rules: append([]Rule(nil), rules...)}, nil
}
func (r *Registry) Rules() []RuleInfo {
	if r == nil {
		return nil
	}
	out := make([]RuleInfo, len(r.rules))
	for i, rule := range r.rules {
		out[i] = rule.Info
	}
	return out
}
func validCategory(category Category) bool {
	return category == Structural || category == Semantic || category == Heuristic
}
func validID(id string) bool {
	if len(id) == 0 || len(id) > 128 || id[0] < 'a' || id[0] > 'z' {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

// Policy configures structural checks only. RequiredSections=nil uses NAME and
// DESCRIPTION plus command/function SYNOPSIS conventions. An explicit empty list
// disables required-section questions. Inline suppressions do not hide input errors.
type Policy struct {
	RequiredSections []string            `json:"requiredSections"`
	Enable           []string            `json:"enable,omitempty"`
	Disable          []string            `json:"disable,omitempty"`
	Severity         map[string]Severity `json:"severity,omitempty"`
	MaxIndentDepth   int                 `json:"maxIndentDepth,omitempty"`
	ClosedWorld      bool                `json:"closedWorld,omitempty"`
}
type Options struct {
	Registry                                     *Registry
	Policy                                       Policy
	Strict                                       bool
	FailOn                                       Severity
	MaxDocuments, MaxNodes, MaxFindings, MaxWork int
}
type budget struct {
	ctx       context.Context
	remaining int
}

func (b *budget) spend(n int) error {
	if err := b.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > b.remaining {
		return ErrLimit
	}
	b.remaining -= n
	return nil
}
