// Package renderhtml exports the supported man semantic model as portable HTML.
// It executes no source code and loads no files, URLs or installed fonts.
package renderhtml

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
)

var ErrLimit = errors.New("HTML output budget exceeded")
var ErrStrict = errors.New("HTML strict conversion rejected diagnostics")

type AssetPolicy string

const (
	EmbedFonts AssetPolicy = "embed"
	OmitFonts  AssetPolicy = "omit-fonts"
)

type Palette struct{ Background, Foreground, Accent, CodeBackground string }
type Options struct {
	Title                string
	Theme                string
	Fragment             bool
	NoAnchors            bool
	Section              string
	Node                 man.ID
	Assets               AssetPolicy
	Fonts                fontset.Set
	FontSize, LineHeight float64
	MaxWidth             int
	Palette              Palette
	Strict               bool
	MaxOutputBytes       int
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func defaults(o Options) (Options, error) {
	if o.Theme == "" {
		o.Theme = "light"
	}
	if o.Theme != "light" && o.Theme != "dark" {
		return o, fmt.Errorf("theme must be light or dark")
	}
	if o.Assets == "" {
		o.Assets = EmbedFonts
	}
	if o.Assets != EmbedFonts && o.Assets != OmitFonts {
		return o, fmt.Errorf("assets must be embed or omit-fonts")
	}
	if o.FontSize == 0 {
		o.FontSize = 18
	}
	if o.LineHeight == 0 {
		o.LineHeight = 1.5
	}
	if o.MaxWidth == 0 {
		o.MaxWidth = 960
	}
	if math.IsNaN(o.FontSize) || math.IsInf(o.FontSize, 0) || o.FontSize < 8 || o.FontSize > 72 {
		return o, fmt.Errorf("font size must be 8..72 pixels")
	}
	if math.IsNaN(o.LineHeight) || math.IsInf(o.LineHeight, 0) || o.LineHeight < 1 || o.LineHeight > 3 {
		return o, fmt.Errorf("line height must be 1..3")
	}
	if o.MaxWidth < 320 || o.MaxWidth > 2400 {
		return o, fmt.Errorf("max width must be 320..2400 pixels")
	}
	if o.MaxOutputBytes <= 0 {
		o.MaxOutputBytes = 32 << 20
	}
	if o.MaxOutputBytes > 32<<20 {
		o.MaxOutputBytes = 32 << 20
	}
	if o.Section != "" && o.Node != 0 {
		return o, fmt.Errorf("section and node selection are mutually exclusive")
	}
	base := Palette{"#ffffff", "#202124", "#174ea6", "#f3f4f6"}
	if o.Theme == "dark" {
		base = Palette{"#17191c", "#f1f3f4", "#8ab4f8", "#25282d"}
	}
	colors := []*string{&o.Palette.Background, &o.Palette.Foreground, &o.Palette.Accent, &o.Palette.CodeBackground}
	fallback := []string{base.Background, base.Foreground, base.Accent, base.CodeBackground}
	for i, p := range colors {
		if *p == "" {
			*p = fallback[i]
		}
		if !colorPattern.MatchString(*p) {
			return o, fmt.Errorf("palette values must be six-digit hexadecimal colors")
		}
	}
	if o.Assets == EmbedFonts {
		o.Fonts = o.Fonts.WithDefaults()
		if err := o.Fonts.Validate(); err != nil {
			return o, err
		}
	}
	return o, nil
}
func number(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }

// SafeURL permits only explicit web/email links and same-document fragments.
// Relative/file/data/script URLs and raw or encoded control bytes are rejected.
func SafeURL(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return false
	}
	for _, r := range decoded {
		if unicode.IsControl(r) {
			return false
		}
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != "" && u.User == nil
	case "mailto":
		return u.Opaque != "" && u.Host == "" && !strings.HasPrefix(u.Opaque, "//")
	case "":
		return u.Host == "" && u.Path == "" && u.Fragment != "" && u.RawQuery == ""
	}
	return false
}

var lengthPattern = regexp.MustCompile(`^([+-]?[0-9]+(?:\.[0-9]+)?)(n|m|em)?$`)

func length(value string, fallback float64) (float64, bool) {
	if value == "" {
		return fallback, true
	}
	parts := lengthPattern.FindStringSubmatch(value)
	if parts == nil {
		return fallback, false
	}
	n, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fallback, false
	}
	if parts[2] == "" || parts[2] == "n" {
		n /= 2
	}
	if n < 0 || n > 20 {
		return fallback, false
	}
	return n, true
}

// ValidateOptions checks settings without loading files or writing output.
func ValidateOptions(o Options) error { _, err := defaults(o); return err }

var verticalPattern = regexp.MustCompile(`^([+]?[0-9]+(?:\.[0-9]+)?)(v|n|m|em)?$`)

func vertical(value string, lineHeight float64) (float64, bool) {
	if value == "" {
		return lineHeight, true
	}
	parts := verticalPattern.FindStringSubmatch(value)
	if parts == nil {
		return lineHeight, false
	}
	n, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return lineHeight, false
	}
	switch parts[2] {
	case "", "v":
		n *= lineHeight
	case "n":
		n /= 2
	}
	if n < 0 || n > 20 {
		return lineHeight, false
	}
	return n, true
}
