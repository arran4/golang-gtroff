package groff

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

//go:embed docs/generated/catalogue-support.json
var catalogueSupportJSON []byte

type catalogueSupportEntry struct {
	Index            int              `json:"index"`
	Kind             TokenKind        `json:"kind"`
	Package          string           `json:"package"`
	Token            string           `json:"token"`
	DeclaredStatus   DefinitionStatus `json:"declared_status"`
	ParseStatus      string           `json:"parse_status"`
	API              string           `json:"api"`
	Parser           string           `json:"parser"`
	Example          string           `json:"example"`
	DocumentType     string           `json:"document_type"`
	DocumentExample  string           `json:"document_example"`
	EvaluationStatus string           `json:"evaluation_status"`
	Reason           string           `json:"reason"`
}

func TestCatalogueSupportRoutes(t *testing.T) {
	var report struct {
		SchemaVersion int    `json:"schema_version"`
		Scope         string `json:"scope"`
		Summary       struct {
			TotalEntries         int            `json:"total_entries"`
			UnexplainedGaps      int            `json:"unexplained_gaps"`
			EntriesByParseStatus map[string]int `json:"entries_by_parse_status"`
		} `json:"summary"`
		Entries []catalogueSupportEntry `json:"entries"`
	}
	if err := json.Unmarshal(catalogueSupportJSON, &report); err != nil {
		t.Fatal(err)
	}
	definitions := CatalogueEntries()
	if report.SchemaVersion != 1 || len(report.Entries) != len(definitions) || report.Summary.TotalEntries != len(definitions) || report.Summary.UnexplainedGaps != 0 {
		t.Fatalf("incomplete support report: version=%d entries=%d definitions=%d summary=%+v", report.SchemaVersion, len(report.Entries), len(definitions), report.Summary)
	}
	counts := map[string]int{}
	registry := NewRegistry()
	for i, entry := range report.Entries {
		t.Run(fmt.Sprintf("%03d/%s/%s", i, entry.Package, entry.Token), func(t *testing.T) {
			def := definitions[i]
			if entry.Index != i || entry.Token != def.Token || entry.Kind != def.Kind || entry.Package != def.Package || entry.DeclaredStatus != def.Status {
				t.Fatalf("report row differs from canonical definition: %+v vs %+v", entry, def)
			}
			if entry.EvaluationStatus != "not_assessed" {
				t.Fatal("syntax route tests do not prove evaluator coverage")
			}
			counts[entry.ParseStatus]++
			if entry.ParseStatus == "syntactically_preserved" {
				inline, err := ParseInline(entry.Example)
				want := InlineStringRef
				if entry.Kind == KindRegister {
					want = InlineRegisterRef
				}
				if entry.API != "ParseInline" || err != nil || len(inline) != 1 || inline[0].Kind != want || inline[0].Raw != entry.Token || inline[0].Name != def.Name {
					t.Fatalf("reference route failed: %+v, %v", inline, err)
				}
				return
			}
			if entry.ParseStatus != "supported_generated" && entry.ParseStatus != "supported_custom" {
				t.Fatalf("uncovered syntax route: %+v", entry)
			}
			tok, err := registry.ParseLine(entry.Example, entry.Package)
			if err != nil || tok == nil {
				t.Fatalf("claimed generated parser rejected %q: %v", entry.Example, err)
			}
			actualType := reflect.TypeOf(tok)
			if actualType.Kind() == reflect.Pointer {
				actualType = actualType.Elem()
			}
			if "Parse"+actualType.Name() != entry.Parser {
				t.Fatalf("dispatcher returned %T, report claims %s", tok, entry.Parser)
			}
			node, ok := tok.(Node)
			if !ok || node.TokenName() != entry.Token || node.PackageName() != entry.Package || node.TokenKind() != entry.Kind {
				t.Fatalf("generated token metadata differs: %T %+v", tok, tok)
			}
			if entry.ParseStatus == "supported_custom" {
				doc, err := ParseDocument(strings.NewReader(entry.DocumentExample), WithRegistry(registry), WithPackage(entry.Package), WithStrict(true))
				if err != nil || doc == nil || len(doc.Tokens) != 1 || len(doc.Diagnostics) != 0 {
					t.Fatalf("claimed document handler failed: %+v, %v", doc, err)
				}
				actualType := reflect.TypeOf(doc.Tokens[0].Token)
				if actualType.Kind() == reflect.Pointer {
					actualType = actualType.Elem()
				}
				if entry.API != "ParseDocument" || actualType.Name() != entry.DocumentType {
					t.Fatalf("document handler returned %s, report claims %s", actualType.Name(), entry.DocumentType)
				}
				var body []string
				switch token := doc.Tokens[0].Token.(type) {
				case *TokenMacroDefinitionVariant:
					body = token.Body
				case *TokenMacroDefinition:
					body = token.Body
				case *TokenMacroAppend:
					body = token.Body
				case *TokenIgnoredBlock:
					body = token.Body
				case *TokenTableBlock:
					body = token.Lines
				case *TokenEquationBlock:
					body = token.Lines
				case *TokenPictureBlock:
					body = token.Lines
				case *TokenReferenceBlock:
					body = token.Lines
				}
				if len(body) != 1 || body[0] != "body" {
					t.Fatalf("custom handler discarded or reclassified body: %q", body)
				}
			}
		})
	}
	if !reflect.DeepEqual(counts, report.Summary.EntriesByParseStatus) {
		t.Fatalf("summary differs from executed rows: %v vs %v", counts, report.Summary.EntriesByParseStatus)
	}
}
