package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"flag"
	"log"
	"os"
	"text/template"
)

var (
	//go:embed "in.json"
	inJson []byte
	//go:embed "*.template"
	goTemplates     embed.FS
	outMacrosLoc    = flag.String("macrosloc", "../../macros.go", "Output file for macros")
	outParselineLoc = flag.String("parselineloc", "../../parseline.go", "Output file for parse line")
)

type Macro struct {
	Type  string
	Token string
	Args  string
}

type MacroContainer struct {
	Description string
	Contents    []Macro
}

type Macros []MacroContainer

func main() {
	flag.Parse()
	log.SetFlags(log.Flags() | log.Lshortfile)
	macros := new(Macros)
	if err := json.Unmarshal(inJson, &macros); err != nil {
		log.Panic(err)
	}
	templates := template.Must(template.ParseFS(goTemplates, "*.template"))
	GenerateFile(templates, macros, "parseline.go.template", *outParselineLoc)
	//GenerateFile(templates, macros, "macros.go.template", *outMacrosLoc)
}

func GenerateFile(templates *template.Template, data any, templateFilename, outputFilename string) {
	f, err := os.Create(outputFilename)
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
