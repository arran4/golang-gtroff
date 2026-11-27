// Package cataloguedoc resolves shared documentation in generator inputs.
// It is used by generation and provenance verification, not runtime parsing.
package cataloguedoc

import (
	"fmt"
	"regexp"
	"strings"
)

// Text either owns a literal or refers directly to a named literal. Empty,
// unnamed literals are allowed for existing synthetic generator inputs.
type Text struct {
	ID, Literal, Ref string
}

type Resolver struct {
	texts  []Text
	owners map[string]int
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// New validates the whole set before returning a resolver. References cannot
// own IDs, so chains and cycles are rejected rather than silently followed.
func New(texts []Text) (*Resolver, error) {
	r := &Resolver{append([]Text(nil), texts...), make(map[string]int)}
	for i, t := range r.texts {
		if t.Ref != "" && (t.Literal != "" || t.ID != "") {
			return nil, fmt.Errorf("documentation %d: reference cannot also own text or an ID", i)
		}
		if t.ID != "" {
			if !identifier.MatchString(t.ID) || strings.TrimSpace(t.Literal) == "" {
				return nil, fmt.Errorf("documentation %d: ID requires a valid name and nonempty literal", i)
			}
			if _, duplicate := r.owners[t.ID]; duplicate {
				return nil, fmt.Errorf("documentation %d: duplicate ID %q", i, t.ID)
			}
			r.owners[t.ID] = i
		}
	}
	for i, t := range r.texts {
		if t.Ref != "" {
			if _, ok := r.owners[t.Ref]; !ok {
				return nil, fmt.Errorf("documentation %d: unknown reference %q", i, t.Ref)
			}
		}
	}
	return r, nil
}

// Resolve returns the original literal and its owner's index. It does not
// normalize whitespace or infer an association from input order.
func (r *Resolver) Resolve(index int) (string, int, error) {
	if r == nil || index < 0 || index >= len(r.texts) {
		return "", 0, fmt.Errorf("documentation index out of range")
	}
	owner := index
	if ref := r.texts[index].Ref; ref != "" {
		owner = r.owners[ref]
	}
	return r.texts[owner].Literal, owner, nil
}
