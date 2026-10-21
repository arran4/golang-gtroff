package renderhtml

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/arran4/golang-gtroff/fontset"
)

var generatedCSSValues = regexp.MustCompile(`(?s)^\.gtroff-document\{margin:0;background:(#[0-9a-fA-F]{6});color:(#[0-9a-fA-F]{6})\}\n\.man-page\{box-sizing:border-box;max-width:([0-9]+)px;margin:0 auto;padding:2rem;font-family:GTroffText,serif;font-size:([0-9.]+)px;line-height:([0-9.]+);background:#[0-9a-fA-F]{6};color:#[0-9a-fA-F]{6};overflow-wrap:anywhere\}\n\.man-page a\{color:(#[0-9a-fA-F]{6})\}.*?\.man-page pre\{white-space:pre-wrap;overflow:auto;padding:1em;background:(#[0-9a-fA-F]{6});`)

// IsGeneratedStylesheet recognizes this renderer's exact CSS template. Embedded
// font payloads are decoded within the face budget, never interpreted or loaded.
// This lets importers distinguish decorative exporter assets from arbitrary CSS.
func IsGeneratedStylesheet(css string) bool {
	css = strings.TrimPrefix(css, "\n")
	if strings.HasPrefix(css, "/* "+fontset.License()+" */\n") {
		css = strings.TrimPrefix(css, "/* "+fontset.License()+" */\n")
		for _, f := range []struct{ family, style, weight string }{{"GTroffText", "normal", "400"}, {"GTroffText", "normal", "700"}, {"GTroffText", "italic", "400"}, {"GTroffMono", "normal", "400"}} {
			line, rest, ok := strings.Cut(css, "\n")
			if !ok {
				return false
			}
			prefix := fmt.Sprintf("@font-face{font-family:%s;font-style:%s;font-weight:%s;font-display:swap;src:url(data:", f.family, f.style, f.weight)
			if !strings.HasPrefix(line, prefix) {
				return false
			}
			payload := strings.TrimPrefix(line, prefix)
			var data string
			switch {
			case strings.HasPrefix(payload, "font/ttf;base64,") && strings.HasSuffix(payload, ") format('truetype')}"):
				data = strings.TrimSuffix(strings.TrimPrefix(payload, "font/ttf;base64,"), ") format('truetype')}")
			case strings.HasPrefix(payload, "font/otf;base64,") && strings.HasSuffix(payload, ") format('opentype')}"):
				data = strings.TrimSuffix(strings.TrimPrefix(payload, "font/otf;base64,"), ") format('opentype')}")
			default:
				return false
			}
			if len(data) > base64.StdEncoding.EncodedLen(2<<20) {
				return false
			}
			b, err := base64.StdEncoding.DecodeString(data)
			if err != nil || len(b) == 0 || len(b) > 2<<20 {
				return false
			}
			css = rest
		}
	}
	m := generatedCSSValues.FindStringSubmatch(css)
	if m == nil {
		return false
	}
	width, err := strconv.Atoi(m[3])
	if err != nil {
		return false
	}
	size, err := strconv.ParseFloat(m[4], 64)
	if err != nil {
		return false
	}
	height, err := strconv.ParseFloat(m[5], 64)
	if err != nil {
		return false
	}
	o, err := defaults(Options{Assets: OmitFonts, MaxWidth: width, FontSize: size, LineHeight: height, Palette: Palette{m[1], m[2], m[6], m[7]}})
	return err == nil && css == styleSheet(o)
}
