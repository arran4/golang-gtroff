package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/arran4/golang-gtroff/gen"
)

var (
	//go:embed "in.json"
	inJson []byte
	//go:embed "*.template"
	goTemplates embed.FS
)

type MacroDef struct {
	ArgumentPlan         argumentPlan
	StructName           string
	Aliases              []string
	DescriptionCommented string
	Token                string
	Kind                 string
	Package              string
	ArgWords             []*gen.Word
	KeepArgumentSlots    bool
	descriptions         map[string]bool
}

type MacrosTemplateData struct {
	Definitions []*MacroDef
}

type DispatchEntry struct {
	Command       string
	HasVariants   bool
	DefaultParser string
	MsParser      string
}

type ParselineTemplateData struct {
	Blocks     []BlockDefinition
	Dispatches []DispatchEntry
}

type CatalogueEntryData struct {
	Index             int
	KindQuoted        string
	PackageQuoted     string
	TokenQuoted       string
	NameQuoted        string
	GoNameQuoted      string
	ArgsRawQuoted     string
	DescriptionQuoted string
	ProvenanceQuoted  string
	IsCustom          bool
	StatusQuoted      string
	HasCollision      bool
}

type CatalogueTemplateData struct {
	Entries []CatalogueEntryData
}

var consolidatedSignatures = map[string]string{
	".po":  "[offset]",
	".in":  "[indent]",
	".ll":  "[length]",
	".pl":  "[length]",
	".lt":  "[length]",
	".pn":  "page",
	".bp":  "[page]",
	".bd":  "font [font2 [offset]]",
	".ps":  "[size]",
	".vs":  "[space]",
	".pvs": "[space]",
	".ft":  "[font]",
	".ti":  "offset",
	".nr":  "ident value [incr]",
	".DS":  "[placement]",
	".NH":  "[level]",
	".CW":  "[txt [post [pre]]]",
	".if":  "cond [anything]",
	".ie":  "cond [anything]",
	".el":  "[anything]",
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepareMacrosData(cat *Catalog) (MacrosTemplateData, error) {
	defsByStruct := map[string]*MacroDef{}
	var structNames []string

	for _, e := range cat.Entries {
		if e == nil {
			continue
		}
		if e.Kind != KindMacro && e.Kind != KindRequest {
			continue
		}

		var structName string
		var aliases []string
		cleanName := sanitizeIdentifier(e.Name)
		parserPackage, parserCollision := e.parserNamespace()

		if e.Kind == KindMacro {
			if parserCollision {
				if parserPackage == PackageMan {
					structName = "TokenMan" + cleanName
					aliases = append(aliases, "TokenMacro"+cleanName)
				} else {
					structName = "TokenMs" + cleanName
				}
			} else {
				structName = "TokenMacro" + cleanName
				if parserPackage == PackageMan {
					aliases = append(aliases, "TokenMan"+cleanName)
				} else if parserPackage == PackageMs {
					aliases = append(aliases, "TokenMs"+cleanName)
				}
			}
		} else if e.Kind == KindRequest {
			structName = "TokenRequest" + cleanName
		}

		// Determine signature
		sig := e.ArgsRaw
		if override, ok := consolidatedSignatures[e.Token]; ok {
			sig = override
		}

		var words []*gen.Word
		if len(strings.TrimSpace(sig)) > 0 {
			a, err := gen.ParseArgs(sig)
			if err != nil {
				return MacrosTemplateData{}, fmt.Errorf("signature %q for %s: %w", sig, e.Token, err)
			} else {
				words = a.Words
			}
		}

		descCommented := ""
		if len(e.Description) > 0 {
			descCommented = "// " + strings.Join(strings.Split(e.Description, "\n"), "\n// ")
		}

		plan, err := planArguments(e, words)
		if err != nil {
			return MacrosTemplateData{}, fmt.Errorf("%s argument grammar: %w", e.Token, err)
		}
		if existing, ok := defsByStruct[structName]; ok {
			if existing.ArgumentPlan.Literal != plan.Literal {
				return MacrosTemplateData{}, fmt.Errorf("conflicting argument grammars for %s", e.Token)
			}
			if !existing.descriptions[e.Description] && e.Description != "" {
				existing.DescriptionCommented += "\n//\n// " + strings.Join(strings.Split(e.Description, "\n"), "\n// ")
				existing.descriptions[e.Description] = true
			}
			continue
		}
		def := &MacroDef{
			descriptions:         map[string]bool{e.Description: true},
			ArgumentPlan:         plan,
			StructName:           structName,
			Aliases:              aliases,
			DescriptionCommented: descCommented,
			Token:                e.Token,
			Kind:                 string(e.Kind),
			Package:              string(parserPackage),
			ArgWords:             words,
		}
		optionalSeen := false
		for _, word := range words {
			if word.Optional {
				optionalSeen = true
			} else if optionalSeen {
				// A serializer must retain positional empty slots around required
				// arguments (e.g. .Pn [punct] path [punct]).
				def.KeepArgumentSlots = true
			}
		}

		defsByStruct[structName] = def
		structNames = append(structNames, structName)
	}

	sort.Strings(structNames)
	var orderedDefs []*MacroDef
	for _, name := range structNames {
		orderedDefs = append(orderedDefs, defsByStruct[name])
	}

	return MacrosTemplateData{Definitions: orderedDefs}, nil
}

func prepareParselineData(cat *Catalog, macrosData MacrosTemplateData) ParselineTemplateData {
	dispatches := map[string]DispatchEntry{}

	// Map structs by (token, package)
	for _, def := range macrosData.Definitions {
		tok := def.Token
		if !strings.HasPrefix(tok, ".") {
			continue
		}

		dispatch, exists := dispatches[tok]
		if !exists {
			dispatch = DispatchEntry{Command: tok}
		}

		if def.Package == string(PackageMs) {
			dispatch.MsParser = def.StructName
		} else {
			dispatch.DefaultParser = def.StructName
		}

		dispatches[tok] = dispatch
	}

	// For colliding tokens, ensure both DefaultParser and MsParser are set
	for _, e := range cat.Entries {
		if e == nil {
			continue
		}
		_, parserCollision := e.parserNamespace()
		if parserCollision && strings.HasPrefix(e.Token, ".") {
			d := dispatches[e.Token]
			d.HasVariants = true
			cleanName := sanitizeIdentifier(e.Name)
			if d.DefaultParser == "" {
				d.DefaultParser = "TokenMan" + cleanName
			}
			if d.MsParser == "" {
				d.MsParser = "TokenMs" + cleanName
			}
			dispatches[e.Token] = d
		}
	}

	// For any entries that only had MsParser, make DefaultParser fall back to MsParser
	for tok, d := range dispatches {
		if d.DefaultParser == "" && d.MsParser != "" {
			d.DefaultParser = d.MsParser
			dispatches[tok] = d
		}
	}

	var commands []string
	for cmd := range dispatches {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)

	var orderedDispatches []DispatchEntry
	for _, cmd := range commands {
		d := dispatches[cmd]
		orderedDispatches = append(orderedDispatches, DispatchEntry{
			Command:       cmd,
			HasVariants:   d.HasVariants,
			DefaultParser: d.DefaultParser,
			MsParser:      d.MsParser,
		})
	}

	return ParselineTemplateData{Dispatches: orderedDispatches, Blocks: prepareBlockDefinitions(cat)}
}

func prepareCatalogueData(cat *Catalog) CatalogueTemplateData {
	var entries []CatalogueEntryData
	for _, e := range cat.Entries {
		entries = append(entries, CatalogueEntryData{
			Index:             e.Index,
			KindQuoted:        fmt.Sprintf("%q", e.Kind),
			PackageQuoted:     fmt.Sprintf("%q", e.Package),
			TokenQuoted:       fmt.Sprintf("%q", e.Token),
			NameQuoted:        fmt.Sprintf("%q", e.Name),
			GoNameQuoted:      fmt.Sprintf("%q", e.GoName),
			ArgsRawQuoted:     fmt.Sprintf("%q", e.ArgsRaw),
			DescriptionQuoted: fmt.Sprintf("%q", e.Description),
			ProvenanceQuoted:  fmt.Sprintf("%q", e.Provenance),
			IsCustom:          e.IsCustom,
			StatusQuoted:      fmt.Sprintf("%q", e.Status),
			HasCollision:      e.HasCollision,
		})
	}
	return CatalogueTemplateData{Entries: entries}
}
