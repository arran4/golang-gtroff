package renderhtml

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/arran4/golang-gtroff/fontset"
)

func styleSheet(o Options) string {
	var b strings.Builder
	if o.Assets == EmbedFonts {
		b.WriteString("/* " + fontset.License() + " */\n")
		for _, f := range []struct {
			family, weight, style string
			data                  []byte
		}{{"GTroffText", "400", "normal", o.Fonts.Regular}, {"GTroffText", "700", "normal", o.Fonts.Bold}, {"GTroffText", "400", "italic", o.Fonts.Italic}, {"GTroffMono", "400", "normal", o.Fonts.Mono}} {
			mime, format := "font/ttf", "truetype"
			if len(f.data) > 4 && string(f.data[:4]) == "OTTO" {
				mime = "font/otf"
				format = "opentype"
			}
			fmt.Fprintf(&b, "@font-face{font-family:%s;font-style:%s;font-weight:%s;font-display:swap;src:url(data:%s;base64,%s) format('%s')}\n", f.family, f.style, f.weight, mime, base64.StdEncoding.EncodeToString(f.data), format)
		}
	}
	fmt.Fprintf(&b, `.gtroff-document{margin:0;background:%s;color:%s}
.man-page{box-sizing:border-box;max-width:%dpx;margin:0 auto;padding:2rem;font-family:GTroffText,serif;font-size:%spx;line-height:%s;background:%s;color:%s;overflow-wrap:anywhere}
.man-page a{color:%s}.man-page h1{font-size:1.7em}.man-page h2{font-size:1.3em;margin-top:1.7em}.man-page h3{font-size:1.1em;margin-top:1.3em}
.man-page p{margin:.75em 0}.man-page pre{white-space:pre-wrap;overflow:auto;padding:1em;background:%s;border-radius:.25em}
.man-page code{font-family:GTroffMono,monospace}.man-page small{font-size:.9em}.man-page dt{font-weight:normal;margin-top:.75em}.man-page dd{margin-inline-start:2em}.man-page dd>:first-child{margin-top:0}
.man-page .manual-meta{opacity:.8;font-size:.9em}.man-page .permalink{font-size:.8em;text-decoration:none;margin-inline-start:.5em}.man-page .unsupported{border-inline-start:.25em solid currentColor}.man-page .conversion-notes{border:1px solid currentColor;padding:1em;margin-top:2em}
@media(max-width:600px){.man-page{padding:1rem}.man-page dd{margin-inline-start:1em}}
@media print{.man-page{max-width:none;color:#000000;background:#ffffff;padding:0}.man-page .permalink{display:none}.man-page pre{break-inside:avoid}}
`, o.Palette.Background, o.Palette.Foreground, o.MaxWidth, number(o.FontSize), number(o.LineHeight), o.Palette.Background, o.Palette.Foreground, o.Palette.Accent, o.Palette.CodeBackground)
	return b.String()
}
