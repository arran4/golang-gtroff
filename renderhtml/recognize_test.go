package renderhtml

import (
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/fontset"
)

func TestGeneratedStylesheetRecognition(t *testing.T) {
	for _, input := range []Options{{}, {Assets: OmitFonts}, {Theme: "dark"}, {MaxWidth: 721, FontSize: 21.5, LineHeight: 1.25, Palette: Palette{"#AABBCC", "#123456", "#654321", "#CCCCCC"}}, {Fonts: fontset.Set{Regular: fontset.Default().Mono}}} {
		o, err := defaults(input)
		if err != nil {
			t.Fatal(err)
		}
		css := styleSheet(o)
		if !IsGeneratedStylesheet(css) || !IsGeneratedStylesheet("\n"+css) {
			t.Fatal("own CSS not recognized")
		}
		for _, altered := range []string{css + "body{display:none}", strings.Replace(css, "margin:0;", "margin:1px;", 1), strings.Replace(css, "GTroffText,serif", "url(https://example.com/)", 1)} {
			if IsGeneratedStylesheet(altered) {
				t.Fatal("arbitrary CSS recognized")
			}
		}
	}
	for _, css := range []string{"", "p{color:red}", "@import url(https://example.com/);"} {
		if IsGeneratedStylesheet(css) {
			t.Fatal("unknown CSS recognized")
		}
	}
}
