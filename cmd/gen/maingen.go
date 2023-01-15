package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/arran4/golang-gtroff/gen"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

var (
	//go:embed "in.json"
	inJson []byte
	//go:embed "*.template"
	goTemplates     embed.FS
	outMacrosLoc    = flag.String("macrosloc", "macros.go", "Output file for macros")
	outParselineLoc = flag.String("parselineloc", "parseline.go", "Output file for parse line")
)

type Macro struct {
	Type  string
	Token string
	Args  string
}

func (m *Macro) IsMacro() bool {
	return m.Type == "Macro"
}

func (m *Macro) ArgStruct() (result []*gen.ArgStruct, err error) {
	// TODO
	return
}

var (
	GoTokenSearch = regexp.MustCompile("[[\\]]")
)

func (m *Macro) GoTokenName() string {
	t := m.Token
	t = GoTokenSearch.ReplaceAllStringFunc(t, func(s string) string {
		switch s {
		case "[":
			return "SquareBracketOpen"
		case "]":
			return "SquareBracketClose"
		default:
			return s
		}
	})
	switch m.Type {
	case "Macro":
		return strings.TrimPrefix(t, ".")
	default:
		return t
	}
}

func (m *Macro) StructName() string {
	return fmt.Sprintf("Token%s%s", m.Type, m.GoTokenName())
}

type FlatMacro struct {
	*Macro
	*MacroContainer
	Grouped []*FlatMacro
}

type MacroContainer struct {
	Description string
	Contents    []*Macro
}

func (mc *MacroContainer) DescriptionCommented() string {
	if len(mc.Description) == 0 {
		return ""
	}
	return "// " + strings.Join(strings.Split(mc.Description, "\n"), "\n// ")
}

type Macros []*MacroContainer

func (m Macros) FilterToJustMacros() (result Macros) {
	for _, emc := range m {
		nemc := &MacroContainer{
			Description: emc.Description,
			Contents:    []*Macro{},
		}
		for _, em := range emc.Contents {
			if em.IsMacro() {
				nemc.Contents = append(nemc.Contents, em)
			}
		}
		if len(nemc.Contents) > 0 {
			result = append(result, nemc)
		}
	}
	return
}

func (m Macros) Flatten() (result FlatMacros) {
	for _, emc := range m {
		for _, em := range emc.Contents {
			fm := &FlatMacro{
				Macro:          em,
				MacroContainer: emc,
				Grouped:        []*FlatMacro{},
			}
			fm.Grouped = append(fm.Grouped, fm)
			result = append(result, fm)
		}
	}
	return
}

type FlatMacros []*FlatMacro

func (fm FlatMacros) DescriptionCommented() string {
	if len(fm.Description()) == 0 {
		return ""
	}
	return "// " + strings.Join(strings.Split(fm.Description(), "\n"), "\n// ")
}

func (fm FlatMacros) Description() (result string) {
	for i, em := range fm {
		if i > 0 {
			result += "\n"
		}
		result += em.Description
	}
	return
}

func (fm FlatMacros) MergeDuplicates() (result FlatMacros) {
	seen := map[string]int{}
	for _, em := range fm {
		if p, ok := seen[em.Token]; ok {
			result[p].Grouped = append(result[p].Grouped, em)
		} else {
			seen[em.Token] = len(result)
			result = append(result, em)
		}
	}
	return
}

func main() {
	flag.Parse()
	log.SetFlags(log.Flags() | log.Lshortfile)
	macros := new(Macros)
	if err := json.Unmarshal(inJson, &macros); err != nil {
		log.Panic(err)
	}
	templates := template.Must(template.ParseFS(goTemplates, "*.template"))
	GenerateFile(templates, macros.FilterToJustMacros().Flatten().MergeDuplicates(), "parseline.go.template", *outParselineLoc)
	GenerateFile(templates, macros.FilterToJustMacros().Flatten().MergeDuplicates(), "macros.go.template", *outMacrosLoc)
}

func GenerateFile(templates *template.Template, data any, templateFilename, outputFilename string) {
	log.Printf("Generating: %s", outputFilename)
	f, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Panic(err)
		}
	}()
	if err := templates.ExecuteTemplate(f, templateFilename, data); err != nil {
		log.Panic(err)
	}
}
