package midp

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"sync"
)

type Graphics struct {
	sync.RWMutex
	Width, Height  int
	Buffer         *image.RGBA
	FrontBuffer    *image.RGBA // true double buffering for Ebiten Draw
	Revision       uint64      // increments on every Flush()
	Dirty          bool        // tracks if Buffer was drawn to since last Flush()
	isPainting     bool        // true while paint() executes; prevents premature intermediate flushes
	CurrentColor   color.Color
	TransX, TransY int
	ClipX, ClipY   int
	ClipW, ClipH   int
}

func NewGraphics(w, h int) *Graphics {
	bg := color.RGBA{R: 15, G: 25, B: 45, A: 255}
	buf := image.NewRGBA(image.Rect(0, 0, w, h))
	fbuf := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(buf, buf.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	draw.Draw(fbuf, fbuf.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	return &Graphics{
		Width:        w,
		Height:       h,
		ClipX:        0,
		ClipY:        0,
		ClipW:        w,
		ClipH:        h,
		Buffer:       buf,
		FrontBuffer:  fbuf,
		CurrentColor: color.White,
		Dirty:        true,
	}
}

// BeginPaint marks that a paint() cycle has started. While painting, Flush() will
// ignore requests to prevent tearing or intermediate cleared buffers from leaking.
func (g *Graphics) BeginPaint() {
	g.Lock()
	g.isPainting = true
	g.Unlock()
}

// EndPaint marks that a paint() cycle has completed and flushes the fully-drawn
// back-buffer to the front-buffer atomically.
func (g *Graphics) EndPaint() {
	g.Lock()
	defer g.Unlock()
	g.isPainting = false
	if !g.Dirty {
		return
	}
	g.Dirty = false
	if g.FrontBuffer != nil && g.Buffer != nil {
		if len(g.FrontBuffer.Pix) == len(g.Buffer.Pix) {
			copy(g.FrontBuffer.Pix, g.Buffer.Pix)
		} else {
			draw.Draw(g.FrontBuffer, g.FrontBuffer.Bounds(), g.Buffer, image.Point{}, draw.Src)
		}
		g.Revision++
	}
}

func (g *Graphics) Flush() {
	g.Lock()
	defer g.Unlock()
	if !g.Dirty || g.isPainting {
		return
	}
	g.Dirty = false
	if g.FrontBuffer != nil && g.Buffer != nil {
		if len(g.FrontBuffer.Pix) == len(g.Buffer.Pix) {
			copy(g.FrontBuffer.Pix, g.Buffer.Pix)
		} else {
			draw.Draw(g.FrontBuffer, g.FrontBuffer.Bounds(), g.Buffer, image.Point{}, draw.Src)
		}
		g.Revision++
	}
}

func (g *Graphics) SetColor(r, gVal, b int) {
	g.CurrentColor = color.RGBA{R: uint8(r), G: uint8(gVal), B: uint8(b), A: 255}
}

func (g *Graphics) Translate(x, y int) {
	g.TransX += x
	g.TransY += y
}

func (g *Graphics) SetClip(x, y, w, h int) {
	g.Lock()
	defer g.Unlock()
	g.ClipX = x + g.TransX
	g.ClipY = y + g.TransY
	g.ClipW = w
	g.ClipH = h
}

func (g *Graphics) ClipRect(x, y, w, h int) {
	g.Lock()
	defer g.Unlock()
	newClip := image.Rect(x+g.TransX, y+g.TransY, x+g.TransX+w, y+g.TransY+h)
	currClip := image.Rect(g.ClipX, g.ClipY, g.ClipX+g.ClipW, g.ClipY+g.ClipH)
	inter := currClip.Intersect(newClip)
	g.ClipX = inter.Min.X
	g.ClipY = inter.Min.Y
	g.ClipW = inter.Dx()
	g.ClipH = inter.Dy()
}

func (g *Graphics) getClippedDst(dst image.Rectangle) image.Rectangle {
	if g.ClipW <= 0 || g.ClipH <= 0 {
		return image.ZR
	}
	clip := image.Rect(g.ClipX, g.ClipY, g.ClipX+g.ClipW, g.ClipY+g.ClipH)
	return dst.Intersect(clip).Intersect(image.Rect(0, 0, g.Width, g.Height))
}

var drawLogCount = 0

func (g *Graphics) FillRect(x, y, w, h int) {
	g.Lock()
	defer g.Unlock()
	x += g.TransX
	y += g.TransY
	dstRect := g.getClippedDst(image.Rect(x, y, x+w, y+h))
	if dstRect.Empty() {
		if drawLogCount < 20 {
			drawLogCount++
			log.Printf("[FILLRECT-EMPTY-CLIP] x=%d y=%d w=%d h=%d TransX=%d TransY=%d Clip=(%d,%d,%d,%d)", x, y, w, h, g.TransX, g.TransY, g.ClipX, g.ClipY, g.ClipW, g.ClipH)
		}
		return
	}
	r, gVal, b, a := g.CurrentColor.RGBA()
	if drawLogCount < 20 {
		drawLogCount++
		log.Printf("[GFX-EXEC] FillRect (%d,%d,%d,%d) clipped=%v color=(%d,%d,%d,%d)", x, y, w, h, dstRect, r>>8, gVal>>8, b>>8, a>>8)
	}
	draw.Draw(g.Buffer, dstRect, &image.Uniform{C: g.CurrentColor}, image.Point{}, draw.Src)
	g.Dirty = true
}

func (g *Graphics) DrawRect(x, y, w, h int) {
	g.Lock()
	defer g.Unlock()
	x += g.TransX
	y += g.TransY
	g.DrawLineRaw(x, y, x+w, y)
	g.DrawLineRaw(x+w, y, x+w, y+h)
	g.DrawLineRaw(x+w, y+h, x, y+h)
	g.DrawLineRaw(x, y+h, x, y)
}

func (g *Graphics) DrawLine(x1, y1, x2, y2 int) {
	g.Lock()
	defer g.Unlock()
	x1 += g.TransX
	y1 += g.TransY
	x2 += g.TransX
	y2 += g.TransY
	g.DrawLineRaw(x1, y1, x2, y2)
}

func (g *Graphics) DrawLineRaw(x1, y1, x2, y2 int) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		if x1 >= g.ClipX && x1 < g.ClipX+g.ClipW && y1 >= g.ClipY && y1 < g.ClipY+g.ClipH {
			if x1 >= 0 && x1 < g.Width && y1 >= 0 && y1 < g.Height {
				g.Buffer.Set(x1, y1, g.CurrentColor)
				g.Dirty = true
			}
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func (g *Graphics) DrawImage(img image.Image, x, y, anchor int) {
	g.Lock()
	defer g.Unlock()
	if img == nil {
		return
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	drawX := x
	if anchor&8 != 0 { // RIGHT = 8
		drawX = x - w
	} else if anchor&1 != 0 { // HCENTER = 1
		drawX = x - w/2
	}

	drawY := y
	if anchor&32 != 0 { // BOTTOM = 32
		drawY = y - h
	} else if anchor&2 != 0 { // VCENTER = 2
		drawY = y - h/2
	}

	realX := drawX + g.TransX
	realY := drawY + g.TransY
	dstRect := g.getClippedDst(image.Rect(realX, realY, realX+w, realY+h))
	if !dstRect.Empty() {
		srcPt := img.Bounds().Min.Add(dstRect.Min.Sub(image.Pt(realX, realY)))
		draw.Draw(g.Buffer, dstRect, img, srcPt, draw.Over)
		g.Dirty = true
	}
}

func (g *Graphics) DrawRegion(img image.Image, xSrc, ySrc, w, h, transform, xDest, yDest, anchor int) {
	g.Lock()
	defer g.Unlock()
	if img == nil || w <= 0 || h <= 0 {
		return
	}

	outW, outH := w, h
	switch transform {
	case 4, 5, 6, 7: // TRANS_MIRROR_ROT270 (4), TRANS_ROT90 (5), TRANS_ROT270 (6), TRANS_MIRROR_ROT90 (7)
		outW, outH = h, w
	}

	drawX := xDest
	if anchor&8 != 0 { // RIGHT
		drawX = xDest - outW
	} else if anchor&1 != 0 { // HCENTER
		drawX = xDest - outW/2
	}

	drawY := yDest
	if anchor&32 != 0 { // BOTTOM
		drawY = yDest - outH
	} else if anchor&2 != 0 { // VCENTER
		drawY = yDest - outH/2
	}

	realX := drawX + g.TransX
	realY := drawY + g.TransY

	dstRect := g.getClippedDst(image.Rect(realX, realY, realX+outW, realY+outH))
	if dstRect.Empty() {
		return
	}

	transImg := image.NewRGBA(image.Rect(0, 0, outW, outH))
	srcBounds := img.Bounds()

	for sy := 0; sy < h; sy++ {
		for sx := 0; sx < w; sx++ {
			srcX := xSrc + sx
			srcY := ySrc + sy
			if srcX < srcBounds.Min.X || srcX >= srcBounds.Max.X || srcY < srcBounds.Min.Y || srcY >= srcBounds.Max.Y {
				continue
			}

			dx, dy := sx, sy
			switch transform {
			case 0: // TRANS_NONE
				dx, dy = sx, sy
			case 1: // TRANS_MIRROR_ROT180 (Vertical flip)
				dx, dy = sx, h-1-sy
			case 2: // TRANS_MIRROR (Horizontal flip)
				dx, dy = w-1-sx, sy
			case 3: // TRANS_ROT180 (180 degree rotation)
				dx, dy = w-1-sx, h-1-sy
			case 4: // TRANS_MIRROR_ROT270
				dx, dy = sy, sx
			case 5: // TRANS_ROT90
				dx, dy = h-1-sy, sx
			case 6: // TRANS_ROT270
				dx, dy = sy, w-1-sx
			case 7: // TRANS_MIRROR_ROT90
				dx, dy = h-1-sy, w-1-sx
			}

			if dx >= 0 && dx < outW && dy >= 0 && dy < outH {
				transImg.Set(dx, dy, img.At(srcX, srcY))
			}
		}
	}

	srcPt := transImg.Bounds().Min.Add(dstRect.Min.Sub(image.Pt(realX, realY)))
	draw.Draw(g.Buffer, dstRect, transImg, srcPt, draw.Over)
	g.Dirty = true
}

func (g *Graphics) DrawString(str string, x, y, anchor int) {
	g.Lock()
	defer g.Unlock()
	runes := []rune(str)
	if len(runes) == 0 {
		return
	}

	totalWidth := len(runes) * 8
	totalHeight := 13

	drawX := x
	if anchor&8 != 0 { // RIGHT
		drawX = x - totalWidth
	} else if anchor&1 != 0 { // HCENTER
		drawX = x - totalWidth/2
	}

	drawY := y
	if anchor&32 != 0 { // BOTTOM
		drawY = y - totalHeight
	} else if anchor&2 != 0 { // VCENTER
		drawY = y - totalHeight/2
	} else if anchor&64 != 0 { // BASELINE
		drawY = y - 10
	}

	realX := drawX + g.TransX
	realY := drawY + g.TransY

	rect := g.getClippedDst(image.Rect(realX, realY, realX+totalWidth, realY+totalHeight))
	if rect.Empty() {
		return
	}

	textColor := g.CurrentColor
	_, _, _, a := textColor.RGBA()
	if a == 0 {
		textColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	for i, ch := range runes {
		glyph := getGlyph8x13(ch)
		charX := realX + i*8
		for row := 0; row < 13; row++ {
			py := realY + row
			if py < g.ClipY || py >= g.ClipY+g.ClipH || py < 0 || py >= g.Height {
				continue
			}
			bits := glyph[row]
			for col := 0; col < 8; col++ {
				if (bits & (1 << (7 - col))) != 0 {
					px := charX + col
					if px >= g.ClipX && px < g.ClipX+g.ClipW && px >= 0 && px < g.Width {
						g.Buffer.Set(px, py, textColor)
						g.Dirty = true
					}
				}
			}
		}
	}
}

func (g *Graphics) DrawRGB(rgbData []int32, offset, scanlength, x, y, width, height int, processAlpha bool) {
	g.Lock()
	defer g.Unlock()
	if len(rgbData) == 0 || width <= 0 || height <= 0 {
		return
	}
	realX := x + g.TransX
	realY := y + g.TransY
	dstRect := g.getClippedDst(image.Rect(realX, realY, realX+width, realY+height))
	if dstRect.Empty() {
		return
	}
	g.Dirty = true

	for row := 0; row < height; row++ {
		py := realY + row
		if py < g.ClipY || py >= g.ClipY+g.ClipH || py < 0 || py >= g.Height {
			continue
		}
		for col := 0; col < width; col++ {
			px := realX + col
			if px < g.ClipX || px >= g.ClipX+g.ClipW || px < 0 || px >= g.Width {
				continue
			}
			idx := offset + row*scanlength + col
			if idx < 0 || idx >= len(rgbData) {
				continue
			}
			pixel := rgbData[idx]
			if processAlpha {
				a := uint8((pixel >> 24) & 0xFF)
				if a == 0 {
					continue
				}
				r := uint8((pixel >> 16) & 0xFF)
				gVal := uint8((pixel >> 8) & 0xFF)
				b := uint8(pixel & 0xFF)
				if a == 255 {
					g.Buffer.SetRGBA(px, py, color.RGBA{R: r, G: gVal, B: b, A: 255})
				} else {
					bg := g.Buffer.RGBAAt(px, py)
					outR := (uint32(r)*uint32(a) + uint32(bg.R)*uint32(255-a)) / 255
					outG := (uint32(gVal)*uint32(a) + uint32(bg.G)*uint32(255-a)) / 255
					outB := (uint32(b)*uint32(a) + uint32(bg.B)*uint32(255-a)) / 255
					g.Buffer.SetRGBA(px, py, color.RGBA{R: uint8(outR), G: uint8(outG), B: uint8(outB), A: 255})
				}
			} else {
				r := uint8((pixel >> 16) & 0xFF)
				gVal := uint8((pixel >> 8) & 0xFF)
				b := uint8(pixel & 0xFF)
				g.Buffer.SetRGBA(px, py, color.RGBA{R: r, G: gVal, B: b, A: 255})
			}
		}
	}
}

func DecodeImage(imgData []byte) (image.Image, error) {
	if len(imgData) >= 8 && imgData[0] == 0x8A && imgData[1] == 0x53 && imgData[2] == 0x4D && imgData[3] == 0x44 {
		decrypted := make([]byte, len(imgData))
		for i, b := range imgData {
			decrypted[i] = b ^ 0x03
		}
		imgData = decrypted
	}
	img, _, err := image.Decode(bytes.NewReader(imgData))
	return img, err
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
