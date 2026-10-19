package main

import (
	"bytes"
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/arran4/golang-gtroff/gen"
)

var (
	//go:embed "in.json"
	inJson []byte
	//go:embed "*.template"
	goTemplates embed.FS

	outMacrosLoc    = flag.String("macrosloc", "macros.go", "Output file for macros")
	outParselineLoc = flag.String("parselineloc", "parseline.go", "Output file for parse line")
	outCatalogueLoc = flag.String("catalogueloc", "catalogue.go", "Output file for catalogue")
)

type MacroDef struct {
	StructName           string
	Aliases              []string
	DescriptionCommented string
	Token                string
	Kind                 string
	Package              string
	ArgWords             []*gen.Word
	Forwards             bool
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
	".po": "[offset]",
	".in": "[indent]",
	".ll": "[length]",
	".pl": "[length]",
	".lt": "[length]",
	".pn": "page",
	".bp": "[page]",
	".bd": "font [font2 [offset]]",
	".ps": "[size]",
	".vs": "[space]",
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
	flag.Parse()
	log.SetFlags(log.Flags() | log.Lshortfile)

	cat, err := NormalizeCatalog(inJson)
	if err != nil {
		log.Fatalf("Normalize catalog error: %v", err)
	}

	templates := template.Must(template.ParseFS(goTemplates, "*.template"))

	// 1. Prepare Macros data
	macrosData := prepareMacrosData(cat)
	GenerateFile(templates, macrosData, "macros.go.template", *outMacrosLoc)

	// 2. Prepare Parseline data
	parselineData := prepareParselineData(cat, macrosData)
	GenerateFile(templates, parselineData, "parseline.go.template", *outParselineLoc)

	// 3. Prepare Catalogue data
	catalogueData := prepareCatalogueData(cat)
	GenerateFile(templates, catalogueData, "catalogue.go.template", *outCatalogueLoc)

	log.Println("Code generation complete.")
}

func prepareMacrosData(cat *Catalog) MacrosTemplateData {
	defsByStruct := map[string]*MacroDef{}
	var structNames []string

	for _, e := range cat.Entries {
		if e.Kind != KindMacro && e.Kind != KindRequest {
			continue
		}

		var structName string
		var aliases []string
		cleanName := sanitizeIdentifier(e.Name)

		if e.Kind == KindMacro {
			if e.HasCollision {
				if e.Package == PackageMan {
					structName = "TokenMan" + cleanName
					aliases = append(aliases, "TokenMacro"+cleanName)
				} else {
					structName = "TokenMs" + cleanName
				}
			} else {
				structName = "TokenMacro" + cleanName
				if e.Package == PackageMan {
					aliases = append(aliases, "TokenMan"+cleanName)
				} else if e.Package == PackageMs {
					aliases = append(aliases, "TokenMs"+cleanName)
				}
			}
		} else if e.Kind == KindRequest {
			structName = "TokenRequest" + cleanName
		}

		if existing, ok := defsByStruct[structName]; ok {
			// Append description if not duplicate
			if !strings.Contains(existing.DescriptionCommented, e.Description) && len(e.Description) > 0 {
				existing.DescriptionCommented += "\n//\n// " + strings.Join(strings.Split(e.Description, "\n"), "\n// ")
			}
			continue
		}

		// Determine signature
		sig := e.ArgsRaw
		if override, ok := consolidatedSignatures[e.Token]; ok {
			sig = override
		}

		var words []*gen.Word
		forwards := false
		if len(strings.TrimSpace(sig)) > 0 {
			a, err := gen.ParseArgs(sig)
			if err != nil {
				log.Printf("Warning: failed to parse signature %q for %s: %v", sig, e.Token, err)
			} else {
				words = a.Words
				forwards = a.Forwards()
			}
		}

		descCommented := ""
		if len(e.Description) > 0 {
			descCommented = "// " + strings.Join(strings.Split(e.Description, "\n"), "\n// ")
		}

		def := &MacroDef{
			StructName:           structName,
			Aliases:              aliases,
			DescriptionCommented: descCommented,
			Token:                e.Token,
			Kind:                 string(e.Kind),
			Package:              string(e.Package),
			ArgWords:             words,
			Forwards:             forwards,
		}

		defsByStruct[structName] = def
		structNames = append(structNames, structName)
	}

	sort.Strings(structNames)
	var orderedDefs []*MacroDef
	for _, name := range structNames {
		orderedDefs = append(orderedDefs, defsByStruct[name])
	}

	return MacrosTemplateData{Definitions: orderedDefs}
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
		if e.HasCollision && strings.HasPrefix(e.Token, ".") {
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

	return ParselineTemplateData{Dispatches: orderedDispatches}
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

func GenerateFile(templates *template.Template, data any, templateFilename, outputFilename string) {
	log.Printf("Generating: %s", outputFilename)

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, templateFilename, data); err != nil {
		log.Fatalf("Template execution error for %s: %v", templateFilename, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Print lines around error for easier debugging
		lines := strings.Split(buf.String(), "\n")
		for i, line := range lines {
			if i < 40 {
				log.Printf("%4d: %s", i+1, line)
			}
		}
		log.Fatalf("format.Source failed for %s: %v", outputFilename, err)
	}

	tmpFile := outputFilename + ".tmp"
	if err := os.WriteFile(tmpFile, formatted, 0644); err != nil {
		log.Fatalf("Failed to write temp file %s: %v", tmpFile, err)
	}

	if err := os.Rename(tmpFile, outputFilename); err != nil {
		log.Fatalf("Failed to rename %s to %s: %v", tmpFile, outputFilename, err)
	}
}
