// Portions Copyright 2016 and 2017 The Go Authors. All rights reserved.
// Adapted from golang.org/x/image/font/opentype and vector under the BSD
// license reproduced in LICENSE-Go-Image.txt.

package renderimage

import (
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// stableFace keeps OpenType metrics and the upstream rasterizer, but flattens
// Bezier curves with explicit float32 rounding. vector's curve interpolation
// permits fused multiply/add on arm64: that changes a few coverage bytes and
// makes otherwise identical PNG/JPEG goldens architecture dependent.
// No compiler flags, global CPU switches or alternate layout engine are needed.
// Like opentype.Face, this face is owned by one render and is not concurrent.
type stableFace struct {
	font.Face
	parsed     *sfnt.Font
	scale      fixed.Int26_6
	buffer     sfnt.Buffer
	raster     vector.Rasterizer
	mask       image.Alpha
	penX, penY float32
}

func (f *stableFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	index, err := f.parsed.GlyphIndex(&f.buffer, r)
	if err != nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	advance, err := f.parsed.GlyphAdvance(&f.buffer, index, f.scale, font.HintingFull)
	if err != nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	// LoadGlyph invalidates previous buffer-backed segments; query metrics first.
	segments, err := f.parsed.LoadGlyph(&f.buffer, index, f.scale, nil)
	if err != nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	bounds := segments.Bounds().Add(dot)
	rect := image.Rect(bounds.Min.X.Floor(), bounds.Min.Y.Floor(), bounds.Max.X.Ceil(), bounds.Max.Y.Ceil())
	width, height := rect.Dx(), rect.Dy()
	if width < 0 || height < 0 || int64(width)*int64(height) > HardMaxPixels {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	n := width * height
	if cap(f.mask.Pix) < n {
		f.mask.Pix = make([]byte, n)
	}
	f.mask.Pix = f.mask.Pix[:n]
	f.mask.Stride = width
	f.mask.Rect = image.Rect(0, 0, width, height)
	bias := dot.Sub(fixed.Point26_6{X: fixed.I(rect.Min.X), Y: fixed.I(rect.Min.Y)})
	f.raster.Reset(width, height)
	f.raster.DrawOp = draw.Src
	f.penX, f.penY = 0, 0
	point := func(p fixed.Point26_6) (float32, float32) { return float32(p.X+bias.X) / 64, float32(p.Y+bias.Y) / 64 }
	for _, segment := range segments {
		x, y := point(segment.Args[0])
		switch segment.Op {
		case sfnt.SegmentOpMoveTo:
			f.penX, f.penY = x, y
			f.raster.MoveTo(x, y)
		case sfnt.SegmentOpLineTo:
			f.line(x, y)
		case sfnt.SegmentOpQuadTo:
			cx, cy := point(segment.Args[1])
			f.quad(x, y, cx, cy)
		case sfnt.SegmentOpCubeTo:
			cx, cy := point(segment.Args[1])
			dx, dy := point(segment.Args[2])
			f.cube(x, y, cx, cy, dx, dy)
		}
	}
	f.raster.Draw(&f.mask, f.mask.Rect, image.Opaque, image.Point{})
	return rect, &f.mask, image.Point{}, advance, index != 0
}
func (f *stableFace) line(x, y float32) { f.raster.LineTo(x, y); f.penX, f.penY = x, y }
func roundedLerp(t, px, py, qx, qy float32) (float32, float32) {
	return px + float32(t*float32(qx-px)), py + float32(t*float32(qy-py))
}
func roundedDeviation(ax, ay, bx, by, cx, cy float32) float32 {
	dx := float32(ax-float32(2*bx)) + cx
	dy := float32(ay-float32(2*by)) + cy
	return float32(dx*dx) + float32(dy*dy)
}
func (f *stableFace) quad(bx, by, cx, cy float32) {
	ax, ay := f.penX, f.penY
	dev := roundedDeviation(ax, ay, bx, by, cx, cy)
	if dev >= 0.333 {
		n := 1 + int(math.Sqrt(math.Sqrt(3*float64(dev))))
		t, step := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += step
			abx, aby := roundedLerp(t, ax, ay, bx, by)
			bcx, bcy := roundedLerp(t, bx, by, cx, cy)
			f.line(roundedLerp(t, abx, aby, bcx, bcy))
		}
	}
	f.line(cx, cy)
}
func (f *stableFace) cube(bx, by, cx, cy, dx, dy float32) {
	ax, ay := f.penX, f.penY
	dev := max(roundedDeviation(ax, ay, bx, by, dx, dy), roundedDeviation(ax, ay, cx, cy, dx, dy))
	if dev >= 0.333 {
		n := 1 + int(math.Sqrt(math.Sqrt(3*float64(dev))))
		t, step := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += step
			abx, aby := roundedLerp(t, ax, ay, bx, by)
			bcx, bcy := roundedLerp(t, bx, by, cx, cy)
			cdx, cdy := roundedLerp(t, cx, cy, dx, dy)
			abcx, abcy := roundedLerp(t, abx, aby, bcx, bcy)
			bcdx, bcdy := roundedLerp(t, bcx, bcy, cdx, cdy)
			f.line(roundedLerp(t, abcx, abcy, bcdx, bcdy))
		}
	}
	f.line(dx, dy)
}
