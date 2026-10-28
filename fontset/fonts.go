// Package fontset supplies the four bundled Go fonts without filesystem lookup.
// Callers may explicitly replace any face with validated TTF/OTF bytes.
package fontset

import (
	_ "embed"
	"fmt"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

const MaxFaceBytes = 2 << 20

//go:embed LICENSE-Go-Fonts.txt
var license string

func License() string { return license }

type Set struct{ Regular, Bold, Italic, Mono []byte }

// Default returns caller-owned font data. No installed fonts are consulted.
func Default() Set {
	return Set{append([]byte(nil), goregular.TTF...), append([]byte(nil), gobold.TTF...), append([]byte(nil), goitalic.TTF...), append([]byte(nil), gomono.TTF...)}
}

// WithDefaults fills missing faces; supplied data remains owned by the caller.
func (s Set) WithDefaults() Set {
	if len(s.Regular) == 0 {
		s.Regular = append([]byte(nil), goregular.TTF...)
	}
	if len(s.Bold) == 0 {
		s.Bold = append([]byte(nil), gobold.TTF...)
	}
	if len(s.Italic) == 0 {
		s.Italic = append([]byte(nil), goitalic.TTF...)
	}
	if len(s.Mono) == 0 {
		s.Mono = append([]byte(nil), gomono.TTF...)
	}
	return s
}
func (s Set) Validate() error {
	for _, face := range []struct {
		name string
		data []byte
	}{{"regular", s.Regular}, {"bold", s.Bold}, {"italic", s.Italic}, {"mono", s.Mono}} {
		if len(face.data) == 0 || len(face.data) > MaxFaceBytes {
			return fmt.Errorf("%s font must contain 1..%d bytes", face.name, MaxFaceBytes)
		}
		if _, err := sfnt.Parse(face.data); err != nil {
			return fmt.Errorf("%s font: %w", face.name, err)
		}
	}
	return nil
}
