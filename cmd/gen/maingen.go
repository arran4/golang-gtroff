package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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

func (m *Macro) MacroGoTokenName() string {
	return strings.TrimPrefix(m.Token, ".")
}

func (m *Macro) StructName() string {
	return fmt.Sprintf("Token%s%s", m.Type, m.MacroGoTokenName())
}

type FlatMacro struct {
	*Macro
	*MacroContainer
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

func (m Macros) Flatten() (result []*FlatMacro) {
	for _, emc := range m {
		for _, em := range emc.Contents {
			result = append(result, &FlatMacro{
				Macro:          em,
				MacroContainer: emc,
			})
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
	GenerateFile(templates, macros, "parseline.go.template", *outParselineLoc)
	GenerateFile(templates, macros.FilterToJustMacros().Flatten(), "macros.go.template", *outMacrosLoc)
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
