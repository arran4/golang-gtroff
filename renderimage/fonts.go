package renderimage

import (
	"fmt"
	"strconv"

	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

type faceKey struct {
	family string
	size   int
} // size in 1/64 point
func (k faceKey) label() string {
	return k.family + "@" + strconv.FormatFloat(float64(k.size)/64, 'f', -1, 64)
}

type fontBook struct {
	data   map[string][]byte
	parsed map[string]*sfnt.Font
	faces  map[faceKey]font.Face
	dpi    float64
}

func fontData(o Options) (map[string][]byte, error) {
	customMono := len(o.Fonts.Mono) > 0
	s := o.Fonts.WithDefaults()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	bi := o.BoldItalic
	if len(bi) == 0 {
		bi = gobolditalic.TTF
	}
	if len(bi) > fontset.MaxFaceBytes {
		return nil, fmt.Errorf("bold italic font exceeds face budget")
	}
	if _, err := sfnt.Parse(bi); err != nil {
		return nil, fmt.Errorf("bold italic font: %w", err)
	}
	data := map[string][]byte{"regular": s.Regular, "bold": s.Bold, "italic": s.Italic, "bold-italic": bi, "mono": s.Mono, "mono-bold": gomonobold.TTF, "mono-italic": gomonoitalic.TTF, "mono-bold-italic": gomonobolditalic.TTF}
	if customMono {
		data["mono-bold"] = s.Mono
		data["mono-italic"] = s.Mono
		data["mono-bold-italic"] = s.Mono
	}
	// A plan owns its font bytes, including explicit replacements.
	for k, v := range data {
		data[k] = append([]byte(nil), v...)
	}
	return data, nil
}
func newBook(data map[string][]byte, dpi float64) *fontBook {
	return &fontBook{data: data, dpi: dpi, parsed: map[string]*sfnt.Font{}, faces: map[faceKey]font.Face{}}
}
func (b *fontBook) close() {
	for _, f := range b.faces {
		_ = f.Close()
	}
}
func (b *fontBook) get(k faceKey) (font.Face, error) {
	if f := b.faces[k]; f != nil {
		return f, nil
	}
	parsed := b.parsed[k.family]
	if parsed == nil {
		var err error
		parsed, err = sfnt.Parse(b.data[k.family])
		if err != nil {
			return nil, err
		}
		b.parsed[k.family] = parsed
	}
	f, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(k.size) / 64, DPI: b.dpi, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	b.faces[k] = f
	return f, nil
}
func key(s man.Style, size float64) faceKey {
	name := "regular"
	if s.Mono {
		name = "mono"
	}
	if s.Bold && s.Italic {
		if s.Mono {
			name = "mono-bold-italic"
		} else {
			name = "bold-italic"
		}
	} else if s.Bold {
		if s.Mono {
			name = "mono-bold"
		} else {
			name = "bold"
		}
	} else if s.Italic {
		if s.Mono {
			name = "mono-italic"
		} else {
			name = "italic"
		}
	}
	if s.Small {
		size *= .9
	}
	return faceKey{name, int(size*64 + .5)}
}
