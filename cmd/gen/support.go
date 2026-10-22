package main

import (
	"encoding/json"
	"fmt"
	"go/token"
	"strings"

	"github.com/arran4/golang-gtroff/gen"
)

// Support states describe syntax routes, independently of the catalogue's
// historical declaration status and of semantic/evaluator support.
const (
	supportGenerated = "supported_generated"
	supportCustom    = "supported_custom"
	supportPreserved = "syntactically_preserved"
	supportMissing   = "unimplemented"
	supportAmbiguous = "ambiguous"
)

type SupportEntry struct {
	BlockKind        string         `json:"block_kind,omitempty"`
	BlockEnds        []string       `json:"block_ends,omitempty"`
	ArgumentMode     string         `json:"argument_mode,omitempty"`
	MinimumArguments int            `json:"minimum_arguments"`
	MaximumArguments int            `json:"maximum_arguments"`
	Index            int            `json:"index"`
	Kind             DefinitionKind `json:"kind"`
	Package          PackageName    `json:"package"`
	Token            string         `json:"token"`
	DeclaredStatus   Status         `json:"declared_status"`
	ParseStatus      string         `json:"parse_status"`
	API              string         `json:"api"`
	Parser           string         `json:"parser,omitempty"`
	Example          string         `json:"example"`
	DocumentType     string         `json:"document_type,omitempty"`
	DocumentExample  string         `json:"document_example,omitempty"`
	EvaluationStatus string         `json:"evaluation_status"`
	Reason           string         `json:"reason,omitempty"`
}

type SupportReport struct {
	SchemaVersion int             `json:"schema_version"`
	Scope         string          `json:"scope"`
	Summary       CoverageSummary `json:"summary"`
	Entries       []SupportEntry  `json:"entries"`
}

type documentSupport struct{ tokenType, example string }

// These examples describe handwritten implementations. Routing itself comes
// from the same block plan emitted by the dispatcher template.
var documentBlockExamples = map[string]documentSupport{
	"definition": {"TokenMacroDefinition", ".de LOCAL\nbody\n..\n"},
	"append":     {"TokenMacroAppend", ".am LOCAL\nbody\n..\n"},
	"ignore":     {"TokenIgnoredBlock", ".ig\nbody\n..\n"},
	"table":      {"TokenTableBlock", ".TS\nbody\n.TE\n"},
	"equation":   {"TokenEquationBlock", ".EQ\nbody\n.EN\n"},
	"picture":    {"TokenPictureBlock", ".PS\nbody\n.PE\n"},
	"reference":  {"TokenReferenceBlock", ".[\nbody\n.]\n"},
}

// buildSupportReport consumes the same concrete generation plan as the Go
// templates. Runtime conformance tests independently execute each claimed route.
func buildSupportReport(cat *Catalog, macros MacrosTemplateData, lines ParselineTemplateData) SupportReport {
	report := SupportReport{SchemaVersion: 1, Scope: "Syntax routes only; examples are dispatch probes, not complete grammar or semantic/evaluator conformance.", Summary: cat.declarationSummary()}
	report.Summary.EntriesByParseStatus = map[string]int{}
	definitions := map[string]*MacroDef{}
	for _, def := range macros.Definitions {
		definitions[def.Kind+"/"+def.Package+"/"+def.Token] = def
	}
	dispatches := map[string][]DispatchEntry{}
	for _, dispatch := range lines.Dispatches {
		dispatches[dispatch.Command] = append(dispatches[dispatch.Command], dispatch)
	}
	blockRoutes := map[string][]BlockDefinition{}
	for _, block := range lines.Blocks {
		blockRoutes[block.Command] = append(blockRoutes[block.Command], block)
		for _, end := range block.Ends {
			blockRoutes[end] = append(blockRoutes[end], block)
		}
	}

	for i, entry := range cat.Entries {
		support := SupportEntry{Index: i, ParseStatus: supportMissing, EvaluationStatus: "not_assessed"}
		if entry == nil {
			support.Reason = "nil catalogue entry"
		} else {
			support.Kind, support.Package, support.Token, support.DeclaredStatus = entry.Kind, entry.Package, entry.Token, entry.Status
			support.Index = entry.Index
			classifySupport(entry, &support, definitions, dispatches, blockRoutes)
		}
		if support.ParseStatus == supportMissing || support.ParseStatus == supportAmbiguous {
			report.Summary.UnexplainedGaps++
		}
		report.Summary.EntriesByParseStatus[support.ParseStatus]++
		report.Entries = append(report.Entries, support)
	}
	return report
}

func classifySupport(entry *NormalizedEntry, support *SupportEntry, definitions map[string]*MacroDef, dispatches map[string][]DispatchEntry, blockRoutes map[string][]BlockDefinition) {
	if err := validateEntry(entry); err != nil {
		support.Reason = err.Error()
		return
	}
	if entry.Kind == KindString || entry.Kind == KindRegister {
		support.ParseStatus, support.API, support.Example = supportPreserved, "ParseInline", entry.Token
		return
	}
	def := definitions[string(entry.Kind)+"/"+string(entry.Package)+"/"+entry.Token]
	if def == nil {
		support.Reason = "missing generated token definition"
		return
	}
	choices := dispatches[entry.Token]
	if len(choices) == 0 {
		support.Reason = "missing generated dispatcher entry"
		return
	}
	if len(choices) != 1 {
		support.ParseStatus, support.Reason = supportAmbiguous, "duplicate generated dispatcher entries"
		return
	}
	dispatch := choices[0]
	parser := dispatch.DefaultParser
	if entry.Package == PackageMs && dispatch.HasVariants {
		parser = dispatch.MsParser
	}
	if parser != def.StructName {
		support.ParseStatus = supportAmbiguous
		support.Reason = fmt.Sprintf("dispatcher selects %q, expected %q", parser, def.StructName)
		return
	}
	support.ParseStatus, support.API, support.Parser = supportGenerated, "ParseLineWithPackage", "Parse"+parser
	support.ArgumentMode = def.ArgumentPlan.Mode
	support.MinimumArguments = def.ArgumentPlan.Minimum
	support.MaximumArguments = def.ArgumentPlan.Maximum
	support.Example = entry.Token
	for _, arg := range def.ArgWords {
		if !arg.Optional {
			support.Example += " value"
		}
	}
	if def.ArgumentPlan.Mode == "delimited" {
		support.Example = entry.Token + " 'left'center'right'"
	}
	if blocks := blockRoutes[entry.Token]; len(blocks) > 0 {
		if len(blocks) != 1 {
			support.ParseStatus, support.Reason = supportAmbiguous, "duplicate block registrations"
			return
		}
		block := blocks[0]
		route, ok := documentBlockExamples[block.Kind]
		if !ok {
			support.ParseStatus, support.Reason = supportMissing, "missing block implementation"
			return
		}
		support.BlockKind, support.BlockEnds = block.Kind, append([]string(nil), block.Ends...)
		support.ParseStatus, support.API = supportCustom, "ParseDocument"
		support.DocumentType, support.DocumentExample = route.tokenType, route.example
	} else if entry.IsCustom {
		switch entry.Token {
		case ".so", ".mso", ".if", ".ie", ".el":
			// Syntax is generated; execution is an independent opt-in contract.
		default:
			support.ParseStatus, support.Reason = supportMissing, "missing custom syntax route"
		}
	}
}

func (report SupportReport) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func validateEntry(entry *NormalizedEntry) error {
	if entry == nil {
		return fmt.Errorf("nil catalogue entry")
	}
	switch entry.Kind {
	case KindMacro, KindRequest, KindString, KindRegister:
	default:
		return fmt.Errorf("entry %d has unknown kind %q", entry.Index, entry.Kind)
	}
	switch entry.Package {
	case PackageMan, PackageMs, PackageRoff:
	default:
		return fmt.Errorf("entry %d has invalid package %q", entry.Index, entry.Package)
	}
	if entry.Token == "" || entry.Name == "" || entry.GoName == "" || entry.Provenance == "" {
		return fmt.Errorf("entry %d is missing token/name/Go name/provenance", entry.Index)
	}
	if !token.IsIdentifier(entry.GoName) {
		return fmt.Errorf("entry %d has invalid Go name %q", entry.Index, entry.GoName)
	}
	switch entry.Status {
	case StatusGenerated:
		if entry.Kind != KindMacro && entry.Kind != KindRequest || entry.IsCustom {
			return fmt.Errorf("entry %d has inconsistent generated status", entry.Index)
		}
	case StatusSyntaxPreserved:
		if entry.Kind != KindString && entry.Kind != KindRegister || entry.IsCustom {
			return fmt.Errorf("entry %d has inconsistent preserved status", entry.Index)
		}
	case StatusCustomHandler:
		if !entry.IsCustom || !customTokens[entry.Token] {
			return fmt.Errorf("entry %d has unregistered custom classification", entry.Index)
		}
	default:
		return fmt.Errorf("entry %d has unknown status %q", entry.Index, entry.Status)
	}
	if (entry.Kind == KindMacro || entry.Kind == KindRequest) && (!strings.HasPrefix(entry.Token, ".") || len(entry.Token) == 1) {
		return fmt.Errorf("entry %d has invalid control token %q", entry.Index, entry.Token)
	}
	if entry.Kind == KindMacro || entry.Kind == KindRequest {
		words, err := entryArgumentWords(entry)
		if err != nil {
			return fmt.Errorf("entry %d argument fields: %w", entry.Index, err)
		}
		if _, err := planArguments(entry, words); err != nil {
			return fmt.Errorf("entry %d argument grammar: %w", entry.Index, err)
		}
		if err := validateSignature(entry.ArgsRaw); err != nil {
			return fmt.Errorf("entry %d has invalid signature: %w", entry.Index, err)
		}
		if override, ok := consolidatedSignatures[entry.Token]; ok {
			if err := validateSignature(override); err != nil {
				return fmt.Errorf("entry %d has invalid signature override: %w", entry.Index, err)
			}
		}
	}
	return nil
}

func validateSignature(signature string) error {
	depth := 0
	for _, r := range signature {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				return fmt.Errorf("unmatched optional-argument delimiter")
			}
		}
	}
	if depth != 0 {
		return fmt.Errorf("unclosed optional-argument delimiter")
	}
	_, err := gen.ParseArgs(signature)
	return err
}
