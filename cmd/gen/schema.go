package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type DefinitionKind string

const (
	KindMacro    DefinitionKind = "Macro"
	KindRequest  DefinitionKind = "Request"
	KindString   DefinitionKind = "String"
	KindRegister DefinitionKind = "Register"
)

type PackageName string

const (
	PackageMan  PackageName = "man"
	PackageMs   PackageName = "ms"
	PackageRoff PackageName = "roff"
)

type Status string

const (
	StatusGenerated       Status = "generated"
	StatusCustomHandler   Status = "custom_handler"
	StatusSyntaxPreserved Status = "syntax_preserved"
)

// RawContainer mirrors the input structure in in.json
type RawContainer struct {
	Description string       `json:"Description"`
	Contents    []RawContent `json:"Contents"`
}

type RawContent struct {
	Type  string `json:"Type"`
	Token string `json:"Token"`
	Args  string `json:"Args"`
}

// NormalizedEntry is the canonical definition for each item in the catalogue
type NormalizedEntry struct {
	Index        int            `json:"index"`
	Kind         DefinitionKind `json:"kind"`
	Package      PackageName    `json:"package"`
	Token        string         `json:"token"`
	Name         string         `json:"name"`
	GoName       string         `json:"go_name"`
	ArgsRaw      string         `json:"args_raw"`
	Description  string         `json:"description"`
	Provenance   string         `json:"provenance"`
	IsCustom     bool           `json:"is_custom"`
	Status       Status         `json:"status"`
	HasCollision bool           `json:"has_collision"`
}

type Catalog struct {
	Entries []*NormalizedEntry `json:"entries"`
}

var customTokens = map[string]bool{
	".de":  true,
	".am":  true,
	".ig":  true,
	".so":  true,
	".mso": true,
	".if":  true,
	".ie":  true,
	".el":  true,
	".TS":  true,
	".TE":  true,
	".EQ":  true,
	".EN":  true,
	".PS":  true,
	".PE":  true,
	".[":   true,
	".]":   true,
}

// NormalizeCatalog parses raw in.json and creates a fully normalized catalogue of 518 entries.
func NormalizeCatalog(inJson []byte) (*Catalog, error) {
	var rawContainers []RawContainer
	if err := json.Unmarshal(inJson, &rawContainers); err != nil {
		return nil, fmt.Errorf("unmarshal in.json: %w", err)
	}

	var entries []*NormalizedEntry
	idx := 0

	for cIdx, c := range rawContainers {
		desc := strings.TrimSpace(c.Description)

		for _, m := range c.Contents {
			kind := DefinitionKind(m.Type)
			token := m.Token
			argsRaw := m.Args

			// Clean name
			cleanName := token
			if strings.HasPrefix(cleanName, ".") {
				cleanName = cleanName[1:]
			} else if strings.HasPrefix(cleanName, `\*[`) && strings.HasSuffix(cleanName, "]") {
				cleanName = cleanName[3 : len(cleanName)-1]
			} else if strings.HasPrefix(cleanName, `\*(`) {
				cleanName = cleanName[3:]
			} else if strings.HasPrefix(cleanName, `\*`) {
				cleanName = cleanName[2:]
			} else if strings.HasPrefix(cleanName, `\n[`) && strings.HasSuffix(cleanName, "]") {
				cleanName = cleanName[3 : len(cleanName)-1]
			} else if strings.HasPrefix(cleanName, `\n(`) {
				cleanName = cleanName[3:]
			} else if strings.HasPrefix(cleanName, `\n`) {
				cleanName = cleanName[2:]
			}

			// Package & Provenance
			var pkg PackageName
			var prov string

			if kind == KindRequest {
				pkg = PackageRoff
				prov = "groff(7)"
			} else if cIdx < 50 || cIdx >= 357 {
				pkg = PackageMan
				prov = "groff_man(7)"
			} else if cIdx < 169 {
				pkg = PackageMs
				prov = "groff_ms(7)"
			} else if cIdx == 356 {
				if kind == KindMacro || kind == KindString || kind == KindRegister {
					pkg = PackageMs
					prov = "groff_ms(7)"
				} else {
					pkg = PackageRoff
					prov = "groff_man(7)"
				}
			} else {
				pkg = PackageRoff
				prov = "groff(7)"
			}

			isCustom := customTokens[token]
			status := StatusGenerated
			if isCustom {
				status = StatusCustomHandler
			} else if kind == KindString || kind == KindRegister {
				status = StatusSyntaxPreserved
			}

			// Generate Go struct/type name
			goName := buildGoName(kind, pkg, token, cleanName)

			entry := &NormalizedEntry{
				Index:       idx,
				Kind:        kind,
				Package:     pkg,
				Token:       token,
				Name:        cleanName,
				GoName:      goName,
				ArgsRaw:     argsRaw,
				Description: desc,
				Provenance:  prov,
				IsCustom:    isCustom,
				Status:      status,
			}
			entries = append(entries, entry)
			idx++
		}
	}

	// Detect collisions (e.g. same token in both man and ms)
	tokenPackages := map[string]map[PackageName]bool{}
	for _, e := range entries {
		if e.Kind == KindMacro {
			if tokenPackages[e.Token] == nil {
				tokenPackages[e.Token] = map[PackageName]bool{}
			}
			tokenPackages[e.Token][e.Package] = true
		}
	}

	for _, e := range entries {
		if e.Kind == KindMacro && len(tokenPackages[e.Token]) > 1 {
			e.HasCollision = true
		}
	}

	cat := &Catalog{Entries: entries}
	if err := cat.Validate(); err != nil {
		return nil, fmt.Errorf("catalog validation failed: %w", err)
	}

	return cat, nil
}

func buildGoName(kind DefinitionKind, pkg PackageName, token, cleanName string) string {
	safeName := sanitizeIdentifier(cleanName)
	switch kind {
	case KindMacro:
		if pkg == PackageMan {
			return "TokenMan" + safeName
		}
		if pkg == PackageMs {
			return "TokenMs" + safeName
		}
		return "TokenMacro" + safeName
	case KindRequest:
		return "TokenRequest" + safeName
	case KindString:
		return "String" + safeName
	case KindRegister:
		return "Register" + safeName
	default:
		return "Token" + safeName
	}
}

func sanitizeIdentifier(s string) string {
	// Handle special tokens like [ and ]
	switch s {
	case "[":
		return "SquareBracketOpen"
	case "]":
		return "SquareBracketClose"
	}

	var sb strings.Builder
	for i, r := range s {
		if i == 0 && r >= '0' && r <= '9' {
			// Prepend Num if starts with digit
			sb.WriteString(string(r))
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if i == 0 && r >= 'a' && r <= 'z' {
				sb.WriteRune(r - 'a' + 'A')
			} else {
				sb.WriteRune(r)
			}
		} else if r == '-' || r == '_' || r == '.' {
			// skip or next char capitalized
		} else {
			sb.WriteString(fmt.Sprintf("%d", r))
		}
	}
	res := sb.String()
	if len(res) == 0 {
		return "Empty"
	}
	return res
}

func (c *Catalog) Validate() error {
	if len(c.Entries) != 518 {
		return fmt.Errorf("expected 518 catalogue entries, got %d", len(c.Entries))
	}

	kindCounts := map[DefinitionKind]int{}
	distinctNames := map[DefinitionKind]map[string]bool{
		KindMacro:    {},
		KindRequest:  {},
		KindString:   {},
		KindRegister: {},
	}

	for _, e := range c.Entries {
		kindCounts[e.Kind]++
		distinctNames[e.Kind][e.Token] = true

		if e.Token == "" {
			return fmt.Errorf("entry %d has empty token", e.Index)
		}
		if e.Package == "" {
			return fmt.Errorf("entry %s has empty package", e.Token)
		}
		if e.Provenance == "" {
			return fmt.Errorf("entry %s has empty provenance", e.Token)
		}
		if e.Status == "" {
			return fmt.Errorf("entry %s has empty status", e.Token)
		}
	}

	if kindCounts[KindMacro] != 141 {
		return fmt.Errorf("expected 141 Macro entries, got %d", kindCounts[KindMacro])
	}
	if kindCounts[KindRequest] != 206 {
		return fmt.Errorf("expected 206 Request entries, got %d", kindCounts[KindRequest])
	}
	if kindCounts[KindString] != 59 {
		return fmt.Errorf("expected 59 String entries, got %d", kindCounts[KindString])
	}
	if kindCounts[KindRegister] != 112 {
		return fmt.Errorf("expected 112 Register entries, got %d", kindCounts[KindRegister])
	}

	if len(distinctNames[KindMacro]) != 111 {
		return fmt.Errorf("expected 111 distinct Macro names, got %d", len(distinctNames[KindMacro]))
	}
	if len(distinctNames[KindRequest]) != 179 {
		return fmt.Errorf("expected 179 distinct Request names, got %d", len(distinctNames[KindRequest]))
	}
	if len(distinctNames[KindString]) != 59 {
		return fmt.Errorf("expected 59 distinct String names, got %d", len(distinctNames[KindString]))
	}
	if len(distinctNames[KindRegister]) != 111 {
		return fmt.Errorf("expected 111 distinct Register names, got %d", len(distinctNames[KindRegister]))
	}

	return nil
}

type CoverageSummary struct {
	TotalEntries         int                    `json:"total_entries"`
	TotalDistinctTokens  int                    `json:"total_distinct_tokens"`
	EntriesByKind        map[string]int         `json:"entries_by_kind"`
	DistinctTokensByKind map[string]int         `json:"distinct_tokens_by_kind"`
	EntriesByStatus      map[string]int         `json:"entries_by_status"`
	EntriesByPackage     map[string]int         `json:"entries_by_package"`
	CollidingTokens      []string               `json:"colliding_tokens"`
	UnexplainedGaps      int                    `json:"unexplained_gaps"`
}

func (c *Catalog) GenerateCoverageSummary() CoverageSummary {
	entriesByKind := map[string]int{}
	distinctByKind := map[string]map[string]bool{}
	entriesByStatus := map[string]int{}
	entriesByPackage := map[string]int{}
	allDistinctTokens := map[string]bool{}
	collidingMap := map[string]bool{}

	for _, e := range c.Entries {
		k := string(e.Kind)
		entriesByKind[k]++
		if distinctByKind[k] == nil {
			distinctByKind[k] = map[string]bool{}
		}
		distinctByKind[k][e.Token] = true
		allDistinctTokens[e.Token] = true
		entriesByStatus[string(e.Status)]++
		entriesByPackage[string(e.Package)]++
		if e.HasCollision {
			collidingMap[e.Token] = true
		}
	}

	distinctTokensByKind := map[string]int{}
	for k, v := range distinctByKind {
		distinctTokensByKind[k] = len(v)
	}

	var colliding []string
	for tok := range collidingMap {
		colliding = append(colliding, tok)
	}
	sort.Strings(colliding)

	return CoverageSummary{
		TotalEntries:         len(c.Entries),
		TotalDistinctTokens:  len(allDistinctTokens),
		EntriesByKind:        entriesByKind,
		DistinctTokensByKind: distinctTokensByKind,
		EntriesByStatus:      entriesByStatus,
		EntriesByPackage:     entriesByPackage,
		CollidingTokens:      colliding,
		UnexplainedGaps:      0,
	}
}
