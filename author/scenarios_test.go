package author_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/author"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var authorCases embed.FS
var updateAuthor = flag.Bool("update-author", false, "explicitly regenerate author scenario output")

type blockData struct {
	Type, Text, Term, Heading string
	Children                  []blockData
}

func dataBlocks(data []blockData) []author.Block {
	blocks := make([]author.Block, 0, len(data))
	for _, entry := range data {
		var block author.Block
		switch entry.Type {
		case "paragraph":
			block = author.Paragraph(entry.Text)
		case "definition":
			block = author.Definition(entry.Term, entry.Text)
		case "literal":
			block = author.LiteralRoff(entry.Text)
		case "preformatted":
			block = author.Preformatted(entry.Text)
		case "subsection":
			block = author.Subsection(entry.Heading, dataBlocks(entry.Children)...)
		}
		blocks = append(blocks, block)
	}
	return blocks
}
func TestAuthorScenarios(t *testing.T) {
	entries, err := authorCases.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
			continue
		}
		path := "testdata/" + entry.Name()
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := authorCases.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse(data)
			files := map[string][]byte{}
			for _, f := range archive.Files {
				if _, exists := files[f.Name]; exists {
					t.Fatal("duplicate", f.Name)
				}
				files[f.Name] = f.Data
			}
			var spec struct {
				Title, Section, Date string
				Sections             []struct {
					Heading string
					Blocks  []blockData
				}
			}
			if err := json.Unmarshal(files["input.json"], &spec); err != nil {
				t.Fatal(err)
			}
			page, buildErr := author.NewManPage(spec.Title, spec.Section, author.WithDate(spec.Date))
			if buildErr == nil {
				for _, section := range spec.Sections {
					if _, buildErr = page.AddSection(section.Heading, dataBlocks(section.Blocks)...); buildErr != nil {
						break
					}
				}
			}
			var output bytes.Buffer
			if buildErr == nil {
				buildErr = page.WriteRoff(&output)
			}
			errorText := ""
			if buildErr != nil {
				errorText = buildErr.Error() + "\n"
			}
			for _, result := range []struct {
				name string
				data []byte
			}{{"output.roff", output.Bytes()}, {"error.txt", []byte(errorText)}} {
				if *updateAuthor {
					found := false
					for i := range archive.Files {
						if archive.Files[i].Name == result.name {
							archive.Files[i].Data = result.data
							found = true
							break
						}
					}
					if !found {
						archive.Files = append(archive.Files, txtar.File{Name: result.name, Data: result.data})
					}
				} else if !bytes.Equal(files[result.name], result.data) {
					t.Errorf("%s mismatch\ngot: %s\nwant: %s", result.name, result.data, files[result.name])
				}
			}
			if *updateAuthor {
				if err := os.WriteFile(path, txtar.Format(archive), 0644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
