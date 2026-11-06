package author_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/arran4/golang-gtroff/author"
	"github.com/arran4/golang-gtroff/man"
)

func ExampleNewManPage() {
	page, err := author.NewManPage("mytool", "1", author.WithDate("2026-11-04"))
	if err != nil {
		panic(err)
	}
	if _, err := page.AddSection("NAME", author.Paragraph("mytool - example")); err != nil {
		panic(err)
	}
	section, err := page.AddSection("OPTIONS", author.Definition("--verbose", "Enable verbose output"))
	if err != nil {
		panic(err)
	}
	if err := section.AddDefinition("--quiet", "Reduce output"); err != nil {
		panic(err)
	}
	tree, err := page.AST()
	if err != nil {
		panic(err)
	}
	document, err := man.Build(tree, man.Options{Strict: true})
	if err != nil {
		panic(err)
	}
	fmt.Println(document.Metadata.Title, document.Metadata.Date)
	for _, node := range document.Nodes {
		if node.Kind == man.Item {
			var term strings.Builder
			for _, run := range node.Tag {
				term.WriteString(run.Text)
			}
			fmt.Println(term.String())
		}
	}
	// Output:
	// mytool 2026-11-04
	// --verbose
	// --quiet
}
func ExamplePage_WriteRoff() {
	page, err := author.NewManPage("tool", "1")
	if err != nil {
		panic(err)
	}
	if err := page.WriteRoff(os.Stdout); err != nil {
		panic(err)
	}
	// Output: .TH "tool" "1" "" "" ""
}
func ExampleLiteralRoff() {
	page, err := author.NewManPage("tool", "1")
	if err != nil {
		panic(err)
	}
	if _, err := page.AddSection("EXAMPLE SOURCE", author.LiteralRoff(".so local.roff\n")); err != nil {
		panic(err)
	}
	tree, err := page.AST()
	if err != nil {
		panic(err)
	}
	document, err := man.Build(tree, man.Options{Strict: true})
	if err != nil {
		panic(err)
	}
	fmt.Print(document.Text(1, man.TextOptions{}))
	// Output:
	// EXAMPLE SOURCE
	// .so local.roff
}
func ExampleSection_Append() { // An application-specific helper composes existing typed blocks.
	environment := func(name, description string) author.Block { return author.Definition(name, description) }
	page, err := author.NewManPage("tool", "1")
	if err != nil {
		panic(err)
	}
	section, err := page.AddSection("ENVIRONMENT")
	if err != nil {
		panic(err)
	}
	if err := section.Append(environment("TOOL_HOME", "Home directory."), environment("TOOL_PATH", "Search path.")); err != nil {
		panic(err)
	}
	fmt.Println(section.Heading())
	// Output: ENVIRONMENT
}
