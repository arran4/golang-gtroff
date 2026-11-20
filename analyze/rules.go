package analyze

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/internal/sourceio"
)

var ErrRules = errors.New("invalid analysis rules")

// RuleMatch is a declarative conjunction over one original syntax node. Empty
// fields impose no condition. A macro without explicit Kind matches physical
// control lines; an explicit block Kind selects captures. At least one condition
// is required. Package means
// the parsed token package, not a guessed document profile. No code is executed.
type RuleMatch struct {
	Kind       groff.ASTKind    `json:"kind,omitempty"`
	Macro      string           `json:"macro,omitempty"`
	InlineKind groff.InlineKind `json:"inlineKind,omitempty"`
	Package    string           `json:"package,omitempty"`
	Opaque     *bool            `json:"opaque,omitempty"`
}

// Rule adds a named question/annotation without overriding syntax, semantic or
// rendering capability facts. Requires are declared processing tags, not acquired
// capabilities. Keep IDs stable when sharing a rule set with other applications.
type Rule struct {
	ID       string    `json:"id"`
	Match    RuleMatch `json:"match"`
	Feature  string    `json:"feature"`
	Reason   string    `json:"reason"`
	Requires []string  `json:"requires,omitempty"`
}

// Registry is an immutable set of additional declarative rules. Built-in feature
// detectors always run; custom rules supplement them. NewRegistry snapshots data.
type Registry struct{ rules []Rule }

func NewRegistry(rules ...Rule) (*Registry, error) {
	if len(rules) > 128 {
		return nil, fmt.Errorf("%w: at most 128 custom rules", ErrRules)
	}
	r := &Registry{}
	seen := map[string]bool{}
	for _, rule := range rules {
		if !validID(rule.ID) || strings.HasPrefix(rule.ID, "builtin.") || seen[rule.ID] {
			return nil, fmt.Errorf("%w: invalid, reserved or duplicate ID %q", ErrRules, rule.ID)
		}
		if !validID(rule.Feature) || len(rule.Reason) == 0 || len(rule.Reason) > 4096 || len(rule.Requires) > 32 {
			return nil, fmt.Errorf("%w: rule %s needs feature/reason and bounded requirements", ErrRules, rule.ID)
		}
		match := rule.Match
		if match.Kind == "" && match.Macro == "" && match.InlineKind == "" && match.Package == "" && match.Opaque == nil {
			return nil, fmt.Errorf("%w: rule %s has no match conditions", ErrRules, rule.ID)
		}
		if match.Kind != "" && match.Kind != groff.ASTLine && match.Kind != groff.ASTInline && match.Kind != groff.ASTBlock && match.Kind != groff.ASTSection && match.Kind != groff.ASTSubsection && match.Kind != groff.ASTDocument {
			return nil, fmt.Errorf("%w: unknown syntax kind", ErrRules)
		}
		if match.Macro != "" && (!strings.HasPrefix(match.Macro, ".") || len(match.Macro) > 128 || strings.ContainsAny(match.Macro, " \t\r\n\x00")) {
			return nil, fmt.Errorf("%w: macro must use dot-normalized literal spelling", ErrRules)
		}
		if match.Package != "" && match.Package != "man" && match.Package != "ms" && match.Package != "roff" {
			return nil, fmt.Errorf("%w: unsupported parsed package", ErrRules)
		}
		if match.InlineKind != "" {
			switch match.InlineKind {
			case groff.InlineText, groff.InlineStringRef, groff.InlineRegisterRef, groff.InlineSpecialChar, groff.InlineFontChange, groff.InlineSizeChange, groff.InlineEscape:
			default:
				return nil, fmt.Errorf("%w: unknown inline kind", ErrRules)
			}
			if match.Macro != "" || match.Package != "" {
				return nil, fmt.Errorf("%w: inline nodes have no macro/package attributes", ErrRules)
			}
		}
		for _, requirement := range rule.Requires {
			if !validID(requirement) {
				return nil, fmt.Errorf("%w: invalid requirement tag", ErrRules)
			}
		}
		seen[rule.ID] = true
		r.rules = append(r.rules, cloneRule(rule))
	}
	if len(r.rules) > 128 {
		return nil, fmt.Errorf("%w: at most 128 custom rules", ErrRules)
	}
	return r, nil
}
func (r *Registry) Rules() []Rule {
	if r == nil {
		return nil
	}
	out := make([]Rule, len(r.rules))
	for i, rule := range r.rules {
		out[i] = cloneRule(rule)
	}
	return out
}
func cloneRule(rule Rule) Rule {
	rule.Requires = append([]string(nil), rule.Requires...)
	if rule.Match.Opaque != nil {
		value := *rule.Match.Opaque
		rule.Match.Opaque = &value
	}
	return rule
}
func validID(value string) bool {
	if len(value) == 0 || len(value) > 128 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == ':') {
			return false
		}
	}
	return true
}

// ParseRules reads a schema-version-1 JSON rules file, limited to 64 KiB. Unknown
// fields, trailing documents and malformed rules are rejected. The reader remains
// borrowed; no imports, file includes, scripts or host capabilities are supported.
func ParseRules(reader io.Reader) (*Registry, error) {
	if sourceio.IsNil(reader) {
		return nil, fmt.Errorf("%w: nil rules reader", ErrRules)
	}
	checked := sourceio.NewReader(reader, func(count, size int, err error) error {
		return &groff.ReaderCountError{Count: count, Size: size, Err: err}
	})
	data, err := io.ReadAll(io.LimitReader(checked, (64<<10)+1))
	if err != nil {
		return nil, fmt.Errorf("read analysis rules: %w", err)
	}
	if len(data) > 64<<10 {
		return nil, fmt.Errorf("%w: rules exceed 64 KiB", ErrRules)
	}
	var config struct {
		SchemaVersion int    `json:"schemaVersion"`
		Rules         []Rule `json:"rules"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRules, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("%w: trailing data", ErrRules)
	}
	if config.SchemaVersion != 1 {
		return nil, fmt.Errorf("%w: unsupported schemaVersion", ErrRules)
	}
	return NewRegistry(config.Rules...)
}
