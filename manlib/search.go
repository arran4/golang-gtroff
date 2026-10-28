package manlib

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Hit struct {
	Name, Section, Description, Root, URL string
	External                              bool
}
type SearchProvider interface {
	Search(context.Context, string) ([]Hit, error)
}

func validQuery(query string) bool {
	if !utf8.ValidString(query) || len(query) > 1024 || utf8.RuneCountInString(query) > 256 {
		return false
	}
	for _, r := range query {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Search performs bounded case-insensitive AND-substring matching across names,
// sections, NAME descriptions and complete semantic text, with stable ranking.
func (i *Index) Search(ctx context.Context, query string, limit int) ([]Hit, error) {
	if i == nil || ctx == nil {
		return nil, fmt.Errorf("nil index/context")
	}
	if !validQuery(query) || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("invalid search query or limit")
	}
	terms := strings.Fields(strings.ToLower(query))
	type ranked struct {
		hit  Hit
		rank int
	}
	var matches []ranked
	for _, p := range i.pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ok := true
		for _, term := range terms {
			if !strings.Contains(p.folded, term) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		name := strings.ToLower(p.entry.Name)
		q := strings.ToLower(strings.TrimSpace(query))
		rank := 4
		switch {
		case name == q:
			rank = 0
		case strings.HasPrefix(name, q):
			rank = 1
		case strings.Contains(name, q):
			rank = 2
		case strings.Contains(strings.ToLower(p.entry.Description), q):
			rank = 3
		}
		e := p.entry
		matches = append(matches, ranked{Hit{Name: e.Name, Section: e.Section, Description: e.Description, Root: e.Root, URL: e.URL}, rank})
	}
	sort.SliceStable(matches, func(a, b int) bool { return matches[a].rank < matches[b].rank })
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]Hit, len(matches))
	for n, m := range matches {
		out[n] = m.hit
	}
	return out, nil
}
func (i *Index) find(name, section, root string) *indexedPage {
	for _, n := range i.lookup[pageKey(name, section)] {
		p := &i.pages[n]
		if root == "" || root == p.entry.Root {
			return p
		}
	}
	return nil
}
