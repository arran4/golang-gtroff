package renderimage

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/arran4/golang-gtroff/man"
	"golang.org/x/image/math/fixed"
)

type palette struct{ background, foreground, code, link, warning color.RGBA }

func colors(theme string) palette {
	if theme == "dark" {
		return palette{color.RGBA{23, 25, 28, 255}, color.RGBA{241, 243, 244, 255}, color.RGBA{37, 40, 45, 255}, color.RGBA{138, 180, 248, 255}, color.RGBA{255, 183, 77, 255}}
	}
	return palette{color.RGBA{255, 255, 255, 255}, color.RGBA{32, 33, 36, 255}, color.RGBA{243, 244, 246, 255}, color.RGBA{23, 78, 166, 255}, color.RGBA{140, 55, 0, 255}}
}

// Image paints one zero-based page. Each call owns its faces and image; no giant
// stitched image or shared mutable glyph cache is required.
func (p *Plan) Image(index int) (image.Image, error) {
	if p == nil || index < 0 || index >= len(p.pages) {
		return nil, fmt.Errorf("page index out of range")
	}
	if p.o.Strict && len(p.result.Diagnostics) > 0 {
		return nil, ErrStrict
	}
	pg := p.pages[index]
	pixels := int64(pg.info.Width) * int64(pg.info.Height)
	if pixels < 1 || pixels > p.o.MaxPixels {
		return nil, ErrLimit
	}
	img := image.NewRGBA(image.Rect(0, 0, pg.info.Width, pg.info.Height))
	pal := colors(p.o.Theme)
	draw.Draw(img, img.Bounds(), image.NewUniform(pal.background), image.Point{}, draw.Src)
	book := newBook(p.data, p.o.DPI)
	defer book.close()
	drawable := image.Rect(p.o.Margin, p.o.Margin, pg.info.Width-p.o.Margin, pg.info.Height-p.o.Margin)
	for _, line := range pg.lines {
		if line.info.Kind == man.Preformatted || line.info.Kind == man.Unsupported {
			rect := image.Rect(line.left, line.info.Top, line.right, line.info.Bottom)
			draw.Draw(img, rect, image.NewUniform(pal.code), image.Point{}, draw.Src)
		}
		if line.info.Kind == man.Unsupported {
			draw.Draw(img, image.Rect(line.left, line.info.Top, line.left+2, line.info.Bottom), image.NewUniform(pal.warning), image.Point{}, draw.Src)
		}
		for _, g := range line.glyphs {
			face, err := book.get(g.face)
			if err != nil {
				return nil, err
			}
			dr, mask, maskp, _, ok := face.Glyph(fixed.Point26_6{X: g.x, Y: line.baseline}, g.char)
			if !ok {
				return nil, fmt.Errorf("planned glyph is unavailable during painting")
			}
			if !dr.Empty() && !dr.In(drawable) {
				return nil, fmt.Errorf("glyph exceeds planned drawable bounds; no clipped image emitted")
			}
			fg := pal.foreground
			if g.link != "" {
				fg = pal.link
			}
			if mask != nil {
				draw.DrawMask(img, dr, image.NewUniform(fg), image.Point{}, mask, maskp, draw.Over)
			}
		}
		for _, f := range line.info.Fragments {
			if f.Link != "" {
				x := int(f.X)
				y := line.info.Baseline + 2
				if y < line.info.Bottom && y < drawable.Max.Y {
					end := min(line.right, int(f.X+f.Advance))
					draw.Draw(img, image.Rect(x, y, end, y+1), image.NewUniform(pal.link), image.Point{}, draw.Src)
				}
			}
		}
	}
	return img, nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, ErrLimit
	}
	return b.Buffer.Write(data)
}
func (p *Plan) encoded(index, limit int) ([]byte, error) {
	img, err := p.Image(index)
	if err != nil {
		return nil, err
	}
	out := &boundedBuffer{limit: limit}
	if p.o.Format == "jpg" {
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: p.o.Quality})
	} else {
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		err = enc.Encode(out, img)
	}
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// WritePage buffers encoding before touching w. Writer failures may be partial.
func (p *Plan) WritePage(w io.Writer, index int) (int, error) {
	if p == nil || w == nil {
		return 0, fmt.Errorf("nil image writer")
	}
	data, err := p.encoded(index, p.o.MaxOutputBytes)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}

// Stream paints one image at a time. The sink owns each image it receives; any
// images retained by it consume additional memory outside the renderer's budget.
func (p *Plan) Stream(sink func(Page, image.Image) error) (Result, error) {
	if p == nil || sink == nil {
		return Result{}, fmt.Errorf("nil plan or image sink")
	}
	result := p.Result()
	for i := range p.pages {
		img, err := p.Image(i)
		if err != nil {
			return result, err
		}
		if err = sink(copyPage(p.pages[i].info), img); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Encode streams bounded encoded pages. Earlier callbacks may have committed
// pages when a later sink/encoding error occurs; the caller owns transactionality.
func (p *Plan) Encode(sink func(Page, []byte) error) (Result, error) {
	if p == nil || sink == nil {
		return Result{}, fmt.Errorf("nil plan or encoded page sink")
	}
	result := p.Result()
	for i := range p.pages {
		data, err := p.encoded(i, p.o.MaxOutputBytes-result.Bytes)
		if err != nil {
			return result, err
		}
		result.Bytes += len(data)
		if err = sink(copyPage(p.pages[i].info), data); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Render writes exactly one image. Multi-page input fails before writing; use
// Layout plus Plan.Encode/Stream for multi-page consumers.
func Render(w io.Writer, d *man.Document, o Options) (Result, error) {
	p, err := Layout(d, o)
	if err != nil {
		if p != nil {
			return p.Result(), err
		}
		return Result{}, err
	}
	result := p.Result()
	if len(p.pages) != 1 {
		return result, ErrMultiplePages
	}
	result.Bytes, err = p.WritePage(w, 0)
	return result, err
}
