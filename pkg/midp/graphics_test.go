package midp

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestGraphicsFillRect(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 0, 0) // Red
	g.FillRect(0, 0, 10, 10)

	r, gVal, b, a := g.Buffer.At(5, 5).RGBA()
	if r == 0 || gVal != 0 || b != 0 || a == 0 {
		t.Errorf("expected red pixel at (5,5), got r=%d, g=%d, b=%d, a=%d", r, gVal, b, a)
	}
}

func TestGraphicsClipping(t *testing.T) {
	g := NewGraphics(240, 320)
	// Set clip to 0,0,0,0 (empty)
	g.SetClip(0, 0, 0, 0)
	dst := g.getClippedDst(image.Rect(0, 0, 100, 100))
	if !dst.Empty() {
		t.Errorf("expected empty destination rectangle for 0-width clip, got %v", dst)
	}

	g.SetColor(255, 0, 0)
	g.FillRect(0, 0, 100, 100)
	r, gVal, b, _ := g.Buffer.At(5, 5).RGBA()
	if r != 15>>8 && gVal != 25>>8 && b != 45>>8 { // Should remain background color
		// Check background color didn't change to red
		r2, _, _, _ := g.Buffer.At(5, 5).RGBA()
		if r2 > 0x8000 {
			t.Errorf("FillRect drew pixels despite empty clip state")
		}
	}

	// Test valid clip with translation
	g.Translate(20, 20)
	g.SetClip(10, 10, 30, 30) // Dynamic clip relative to translation -> absolute (30,30,60,60)
	if g.ClipX != 30 || g.ClipY != 30 || g.ClipW != 30 || g.ClipH != 30 {
		t.Errorf("unexpected clip bounds: got (%d,%d,%d,%d)", g.ClipX, g.ClipY, g.ClipW, g.ClipH)
	}
}

func TestGraphicsDrawStringGlyphs(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 255, 255)
	g.DrawString("HI", 10, 10, 0)

	// Verify that not all pixels in the bounding box (10..26, 10..23) are filled (unlike solid rectangle)
	filledCount := 0
	for y := 10; y < 23; y++ {
		for x := 10; x < 26; x++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
				filledCount++
			}
		}
	}
	if filledCount == 0 {
		t.Errorf("expected rendered glyph pixels for DrawString('HI'), got 0")
	}
	// Bounding box area = 16 * 13 = 208. Glyph pixels should be a fraction, not 208
	if filledCount == 208 {
		t.Errorf("DrawString still drew a solid filled box instead of character glyphs")
	}
}

func TestGraphicsDrawImageAnchors(t *testing.T) {
	srcImg := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			srcImg.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	g := NewGraphics(240, 320)
	// HCENTER = 1, VCENTER = 2 -> (100 - 10, 100 - 10) = (90, 90)
	g.DrawImage(srcImg, 100, 100, 1|2)

	r, gVal, b, _ := g.Buffer.At(95, 95).RGBA()
	if gVal < 0x8000 || r > 0x1000 || b > 0x1000 {
		t.Errorf("expected green pixel at (95,95) for centered DrawImage, got gVal=%d", gVal)
	}
}

func TestGraphicsDrawRegionTransforms(t *testing.T) {
	// Create 2x2 image with distinct colors:
	// (0,0) Red, (1,0) Green, (0,1) Blue, (1,1) White
	srcImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	srcImg.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	srcImg.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	srcImg.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	srcImg.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	g := NewGraphics(240, 320)
	// TRANS_MIRROR = 2 (Horizontal flip): (0,0) becomes Green, (1,0) becomes Red
	g.DrawRegion(srcImg, 0, 0, 2, 2, 2, 50, 50, 0)

	r0, g0, _, _ := g.Buffer.At(50, 50).RGBA()
	if g0 < 0x8000 || r0 > 0x1000 {
		t.Errorf("expected Green at (50,50) after TRANS_MIRROR, got r=%d g=%d", r0, g0)
	}

	r1, g1, _, _ := g.Buffer.At(51, 50).RGBA()
	if r1 < 0x8000 || g1 > 0x1000 {
		t.Errorf("expected Red at (51,50) after TRANS_MIRROR, got r=%d g=%d", r1, g1)
	}
}

func TestGraphicsDrawRGB(t *testing.T) {
	g := NewGraphics(240, 320)
	rgb := []int32{
		-0x10000,   // 0xFFFF0000 Red
		-0xFF0100,  // 0xFF00FF00 Green
		-0xFFFF01,  // 0xFF0000FF Blue
		-1,         // 0xFFFFFFFF White
	}
	g.DrawRGB(rgb, 0, 2, 10, 10, 2, 2, true)

	r, _, _, _ := g.Buffer.At(10, 10).RGBA()
	if r < 0x8000 {
		t.Errorf("expected red pixel at (10,10) from DrawRGB, got r=%d", r)
	}
}

func TestDecodeImage(t *testing.T) {
	// 1. Test valid PNG decoding
	srcImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			srcImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, srcImg); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}

	decoded, err := DecodeImage(buf.Bytes())
	if err != nil {
		t.Fatalf("DecodeImage failed for valid PNG: %v", err)
	}
	if decoded == nil || decoded.Bounds().Dx() != 10 || decoded.Bounds().Dy() != 10 {
		t.Errorf("DecodeImage returned unexpected image bounds: %v", decoded)
	}

	// 2. Test invalid image byte stream
	_, err = DecodeImage([]byte("not an image"))
	if err == nil {
		t.Errorf("expected error decoding invalid image bytes, got nil")
	}
}

func TestDrawRegionAllTransforms(t *testing.T) {
	// Use an asymmetric 2x3 image (w=2, h=3) with 6 distinct colors to verify all 8 transforms
	w, h := 2, 3
	srcImg := image.NewRGBA(image.Rect(0, 0, w, h))

	colors := map[[2]int]color.RGBA{
		{0, 0}: {R: 255, G: 0, B: 0, A: 255},     // Red
		{1, 0}: {R: 0, G: 255, B: 0, A: 255},     // Green
		{0, 1}: {R: 0, G: 0, B: 255, A: 255},     // Blue
		{1, 1}: {R: 255, G: 255, B: 0, A: 255},   // Yellow
		{0, 2}: {R: 255, G: 0, B: 255, A: 255},   // Magenta
		{1, 2}: {R: 0, G: 255, B: 255, A: 255},   // Cyan
	}
	for pt, col := range colors {
		srcImg.Set(pt[0], pt[1], col)
	}

	tests := []struct {
		name      string
		transform int
		expW      int
		expH      int
		mapCoord  func(sx, sy int) (dx, dy int)
	}{
		{
			name:      "TRANS_NONE (0)",
			transform: 0,
			expW:      w,
			expH:      h,
			mapCoord:  func(sx, sy int) (int, int) { return sx, sy },
		},
		{
			name:      "TRANS_MIRROR_ROT180 (1)",
			transform: 1,
			expW:      w,
			expH:      h,
			mapCoord:  func(sx, sy int) (int, int) { return sx, h - 1 - sy },
		},
		{
			name:      "TRANS_MIRROR (2)",
			transform: 2,
			expW:      w,
			expH:      h,
			mapCoord:  func(sx, sy int) (int, int) { return w - 1 - sx, sy },
		},
		{
			name:      "TRANS_ROT180 (3)",
			transform: 3,
			expW:      w,
			expH:      h,
			mapCoord:  func(sx, sy int) (int, int) { return w - 1 - sx, h - 1 - sy },
		},
		{
			name:      "TRANS_MIRROR_ROT270 (4)",
			transform: 4,
			expW:      h,
			expH:      w,
			mapCoord:  func(sx, sy int) (int, int) { return sy, sx },
		},
		{
			name:      "TRANS_ROT90 (5)",
			transform: 5,
			expW:      h,
			expH:      w,
			mapCoord:  func(sx, sy int) (int, int) { return h - 1 - sy, sx },
		},
		{
			name:      "TRANS_ROT270 (6)",
			transform: 6,
			expW:      h,
			expH:      w,
			mapCoord:  func(sx, sy int) (int, int) { return sy, w - 1 - sx },
		},
		{
			name:      "TRANS_MIRROR_ROT90 (7)",
			transform: 7,
			expW:      h,
			expH:      w,
			mapCoord:  func(sx, sy int) (int, int) { return h - 1 - sy, w - 1 - sx },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			destX, destY := 20, 30
			g := NewGraphics(100, 100)
			g.DrawRegion(srcImg, 0, 0, w, h, tc.transform, destX, destY, 0)

			// 1. Verify every source pixel is placed at the exact destination coordinate
			for sy := 0; sy < h; sy++ {
				for sx := 0; sx < w; sx++ {
					expectedDx, expectedDy := tc.mapCoord(sx, sy)
					actualPixel := g.Buffer.At(destX+expectedDx, destY+expectedDy)
					r, gVal, b, a := actualPixel.RGBA()
					expectedCol := colors[[2]int{sx, sy}]

					if r>>8 != uint32(expectedCol.R) || gVal>>8 != uint32(expectedCol.G) || b>>8 != uint32(expectedCol.B) || a>>8 != uint32(expectedCol.A) {
						t.Errorf("transform %s: src (%d,%d) mapped to (%d,%d) had color (R=%d,G=%d,B=%d), expected (R=%d,G=%d,B=%d)",
							tc.name, sx, sy, expectedDx, expectedDy,
							r>>8, gVal>>8, b>>8,
							expectedCol.R, expectedCol.G, expectedCol.B)
					}
				}
			}

			// 2. Verify boundary: pixels outside the destination box (destX..destX+tc.expW, destY..destY+tc.expH) are not overwritten
			boundaryChecks := [][2]int{
				{destX - 1, destY},
				{destX, destY - 1},
				{destX + tc.expW, destY},
				{destX, destY + tc.expH},
			}
			for _, pt := range boundaryChecks {
				r, gVal, b, _ := g.Buffer.At(pt[0], pt[1]).RGBA()
				if r>>8 != 15 || gVal>>8 != 25 || b>>8 != 45 {
					t.Errorf("transform %s: pixel outside destination bounding box at (%d,%d) was altered", tc.name, pt[0], pt[1])
				}
			}
		})
	}
}

func TestDrawImageAllAnchors(t *testing.T) {
	srcImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			srcImg.Set(x, y, color.RGBA{R: 255, G: 255, B: 0, A: 255})
		}
	}

	anchors := []struct {
		anchor   int
		expected image.Rectangle
	}{
		{0, image.Rect(100, 100, 110, 110)},             // TOP|LEFT
		{8, image.Rect(90, 100, 100, 110)},              // TOP|RIGHT
		{1, image.Rect(95, 100, 105, 110)},              // TOP|HCENTER
		{32, image.Rect(100, 90, 110, 100)},             // BOTTOM|LEFT
		{32 | 8, image.Rect(90, 90, 100, 100)},          // BOTTOM|RIGHT
		{32 | 1, image.Rect(95, 90, 105, 100)},          // BOTTOM|HCENTER
		{2 | 1, image.Rect(95, 95, 105, 105)},           // VCENTER|HCENTER
	}

	for _, tc := range anchors {
		g := NewGraphics(240, 320)
		g.DrawImage(srcImg, 100, 100, tc.anchor)

		// Check inside expected rectangle
		cx := (tc.expected.Min.X + tc.expected.Max.X) / 2
		cy := (tc.expected.Min.Y + tc.expected.Max.Y) / 2
		r, gVal, _, _ := g.Buffer.At(cx, cy).RGBA()
		if r < 0x8000 || gVal < 0x8000 {
			t.Errorf("anchor %d failed: expected yellow pixel at (%d,%d), got r=%d gVal=%d", tc.anchor, cx, cy, r, gVal)
		}
	}
}

