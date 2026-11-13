// Package analyze inventories roff syntax, source requirements and capability
// evidence without executing macros or rendering. Classification is separate
// from parser diagnostics, structural lint policy and subjective quality scores.
package analyze

import (
	"context"
	"errors"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/source"
)

// Status labels one capability stage, not an assertion about all other stages.
type Status string

const (
	Recognized        Status = "recognized"
	Parsed            Status = "parsed"
	Modeled           Status = "modeled"
	PartiallyResolved Status = "partially-resolved"
	PartiallyModeled  Status = "partially-modeled"
	Unsupported       Status = "unsupported"
	Unknown           Status = "unknown"
	Opaque            Status = "opaque"
	NotAssessed       Status = "not-assessed"
)

// Capability explicitly separates syntax recognition, man interpretation, include
// resolution and rendering. Rendering is never attempted by this analyzer.
type Capability struct {
	Syntax     Status `json:"syntax"`
	Semantics  Status `json:"semantics"`
	Resolution string `json:"resolution,omitempty"`
	Rendering  Status `json:"rendering"`
}
type Finding struct {
	SchemaVersion int          `json:"schemaVersion"`
	Rule          string       `json:"rule"`
	Feature       string       `json:"feature"`
	Name          string       `json:"name,omitempty"`
	Source        source.ID    `json:"source"`
	Node          groff.NodeID `json:"node"`
	Span          groff.Span   `json:"span"`
	Package       string       `json:"package,omitempty"`
	KnownPackages []string     `json:"knownPackages,omitempty"`
	Active        bool         `json:"active"`
	Capability    Capability   `json:"capability"`
	Reason        string       `json:"reason"`
	Requires      []string     `json:"requires,omitempty"`
	Target        string       `json:"target,omitempty"`
	Parent        groff.NodeID `json:"parent,omitempty"`
}

type Evidence struct {
	Source  source.ID    `json:"source"`
	Node    groff.NodeID `json:"node"`
	Span    groff.Span   `json:"span"`
	Profile string       `json:"profile"`
	Reason  string       `json:"reason"`
}

// Profile is a lexical document-profile hypothesis, with explicit uncertainty.
// Macro definitions can change runtime meaning; evidence is not execution proof.
type Profile struct {
	Name       string     `json:"name"`
	Confidence string     `json:"confidence"`
	Reason     string     `json:"reason"`
	Evidence   []Evidence `json:"evidence,omitempty"`
}

// Flags describe observed syntax. RequiresEvaluation excludes opaque/comment
// material; presence flags can include stored syntax, as shown by Finding.Active.
type Flags struct {
	HasConditional      bool `json:"hasConditional"`
	HasInclude          bool `json:"hasInclude"`
	RequiresEvaluation  bool `json:"requiresEvaluation"`
	UsesPreprocessor    bool `json:"usesPreprocessor"`
	HasDefinitions      bool `json:"hasDefinitions"`
	HasReferences       bool `json:"hasReferences"`
	HasSideEffectSyntax bool `json:"hasSideEffectSyntax"`
	HasUnknown          bool `json:"hasUnknown"`
	HasOpaque           bool `json:"hasOpaque"`
	HasPartialSource    bool `json:"hasPartialSource"`
}
type Counts struct {
	Features map[string]int `json:"features"`
	Controls map[string]int `json:"controls"`
	Escapes  map[string]int `json:"escapes"`
	Packages map[string]int `json:"packages"`
	Rules    map[string]int `json:"rules"`
}
type SourceInfo struct {
	Profile       Profile       `json:"profile"`
	ID            source.ID     `json:"id"`
	Name          string        `json:"name"`
	Depth         int           `json:"depth"`
	ParentEdge    source.EdgeID `json:"parentEdge,omitempty"`
	Bytes         int           `json:"bytes"`
	Nodes         int           `json:"nodes"`
	Lines         int           `json:"lines"`
	MaxBlockDepth int           `json:"maxBlockDepth"`
	Complete      bool          `json:"complete"`
}

// Report is caller-owned analysis data. Complete describes enumeration over the
// supplied source set, independently of source EOF, resolution and support.
type Report struct {
	SemanticProfile   string              `json:"semanticProfile"`
	Includes          []source.Edge       `json:"includes,omitempty"`
	SchemaVersion     int                 `json:"schemaVersion"`
	CapabilityVersion int                 `json:"capabilityVersion"`
	Command           string              `json:"command"`
	Complete          bool                `json:"complete"`
	Error             string              `json:"error,omitempty"`
	InputError        string              `json:"inputError,omitempty"`
	Profile           Profile             `json:"profile"`
	Flags             Flags               `json:"flags"`
	Counts            Counts              `json:"counts"`
	Requirements      []string            `json:"requirements,omitempty"`
	Sources           []SourceInfo        `json:"sources"`
	Findings          []Finding           `json:"findings"`
	Diagnostics       []source.Diagnostic `json:"diagnostics,omitempty"`

	// WorkUnits counts charged analyzer work, allowing callers to share budgets.
	// It is execution accounting and is omitted from serialized analysis reports.
	WorkUnits int `json:"-"`
}

func (r *Report) HasRule(id string) bool { return r != nil && r.Counts.Rules[id] > 0 }

var ErrLimit = errors.New("analysis resource limit exceeded")

type Options struct {
	Registry                       *Registry
	MaxNodes, MaxFindings, MaxWork int
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
