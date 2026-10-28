// Package renderimage lays out and paints the semantic man model using pure Go.
// It loads no files, URLs, includes or installed fonts.
package renderimage

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
)

var ErrLimit = errors.New("image resource budget exceeded")
var ErrStrict = errors.New("strict image rendering rejected diagnostics")
var ErrMultiplePages = errors.New("multiple pages require a page sink or numbered files")

const HardMaxPixels int64 = 64 << 20
const HardMaxGlyphs = 250000
const HardMaxPages = 256
const HardMaxOutputBytes = 64 << 20

type Options struct {
	Width, Margin                       int // device pixels; zero margin defaults to 48, NoMargin requests 0
	NoMargin                            bool
	FontSize, DPI, LineHeight           float64 // points (1/72 inch), pixels/inch, line-height ratio
	Theme, Title, Format                string
	PageHeight, MaxHeight               int // pixels; Single derives height, bounded by MaxHeight
	Single                              bool
	MaxPixels                           int64 // total across pages, not merely one allocation
	MaxPages, MaxGlyphs, MaxOutputBytes int
	Quality                             int // JPEG 1..100
	Section                             string
	Node                                man.ID
	Strict                              bool
	Fonts                               fontset.Set
	BoldItalic                          []byte // optional explicit proportional bold/italic face
}

func defaults(o Options) (Options, error) {
	if o.Width == 0 {
		o.Width = 1000
	}
	if o.Margin == 0 && !o.NoMargin {
		o.Margin = 48
	}
	if o.NoMargin {
		o.Margin = 0
	}
	if o.FontSize == 0 {
		o.FontSize = 15
	}
	if o.DPI == 0 {
		o.DPI = 96
	}
	if o.LineHeight == 0 {
		o.LineHeight = 1.35
	}
	if o.Theme == "" {
		o.Theme = "light"
	}
	if o.Format == "" {
		o.Format = "png"
	}
	if o.Format == "jpeg" {
		o.Format = "jpg"
	}
	if o.PageHeight == 0 {
		o.PageHeight = 1400
	}
	if o.MaxHeight == 0 {
		o.MaxHeight = 32768
	}
	if o.MaxPixels == 0 {
		o.MaxPixels = HardMaxPixels
	}
	if o.MaxPages == 0 {
		o.MaxPages = HardMaxPages
	}
	if o.MaxGlyphs == 0 {
		o.MaxGlyphs = HardMaxGlyphs
	}
	if o.MaxOutputBytes == 0 {
		o.MaxOutputBytes = HardMaxOutputBytes
	}
	if o.Quality == 0 {
		o.Quality = 90
	}
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if o.Width < 128 || o.Width > 8192 || o.Margin < 0 || o.Margin > 4096 || o.Width-2*o.Margin < 32 {
		return o, fmt.Errorf("width must be 128..8192 pixels with at least 32 drawable pixels and nonnegative margins")
	}
	if !finite(o.FontSize) || o.FontSize < 6 || o.FontSize > 72 || !finite(o.DPI) || o.DPI < 36 || o.DPI > 300 || !finite(o.LineHeight) || o.LineHeight < 1 || o.LineHeight > 3 {
		return o, fmt.Errorf("font size must be 6..72 points, DPI 36..300 and line height 1..3")
	}
	if o.Theme != "light" && o.Theme != "dark" {
		return o, fmt.Errorf("theme must be light or dark")
	}
	if o.Format != "png" && o.Format != "jpg" {
		return o, fmt.Errorf("image format must be png or jpg")
	}
	if o.PageHeight < 128 || o.PageHeight > 8192 || o.MaxHeight < 128 || o.MaxHeight > 32768 || (!o.Single && o.PageHeight > o.MaxHeight) {
		return o, fmt.Errorf("page height must be 128..8192 and max height 128..32768 pixels")
	}
	height := o.PageHeight
	if o.Single {
		height = o.MaxHeight
	}
	if height-2*o.Margin < 32 {
		return o, fmt.Errorf("margins leave no useful page height")
	}
	if o.MaxPixels < 1 || o.MaxPixels > HardMaxPixels || o.MaxPages < 1 || o.MaxPages > HardMaxPages || o.MaxGlyphs < 1 || o.MaxGlyphs > HardMaxGlyphs || o.MaxOutputBytes < 1 || o.MaxOutputBytes > HardMaxOutputBytes {
		return o, fmt.Errorf("resource limits must be positive and within documented hard ceilings")
	}
	if !o.Single && int64(o.Width)*int64(o.PageHeight) > o.MaxPixels {
		return o, ErrLimit
	}
	if o.Quality < 1 || o.Quality > 100 {
		return o, fmt.Errorf("JPEG quality must be 1..100")
	}
	if o.Section != "" && o.Node != 0 {
		return o, fmt.Errorf("section and node are mutually exclusive")
	}
	return o, nil
}

// ValidateOptions checks geometry and budgets without loading fonts or input.
func ValidateOptions(o Options) error { _, err := defaults(o); return err }

// OutputPaths returns deterministic one-based page names, without filesystem IO.
// A multi-page base foo.png produces foo-001.png, foo-002.png, ... .
func OutputPaths(path, format string, count int) ([]string, error) {
	if format == "jpeg" {
		format = "jpg"
	}
	if format != "png" && format != "jpg" {
		return nil, fmt.Errorf("invalid image format")
	}
	if count < 1 || count > HardMaxPages {
		return nil, ErrLimit
	}
	if path == "-" {
		if count > 1 {
			return nil, ErrMultiplePages
		}
		return []string{"-"}, nil
	}
	ext := filepath.Ext(path)
	lower := strings.ToLower(ext)
	if !(format == "png" && lower == ".png" || format == "jpg" && (lower == ".jpg" || lower == ".jpeg")) {
		return nil, fmt.Errorf("output filename must have an extension matching %s", format)
	}
	if count == 1 {
		return []string{path}, nil
	}
	out := make([]string, count)
	base := strings.TrimSuffix(path, ext)
	for i := range out {
		out[i] = fmt.Sprintf("%s-%03d%s", base, i+1, ext)
	}
	return out, nil
}
