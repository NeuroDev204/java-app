package midp

import (
	"image/color"
	"testing"
)

func TestEmpirical_FillRect_FullScreenBlackBufferClearing(t *testing.T) {
	w, h := 240, 320
	g := NewGraphics(w, h)

	// Step 1: Draw non-black objects (gold rectangle) to populate VRAM/buffer
	g.SetColor(255, 215, 0)
	g.FillRect(0, 0, w, h)

	// Verify all pixels are gold (255, 215, 0, 255)
	goldCount := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := g.Buffer.RGBAAt(x, y)
			if c.R == 255 && c.G == 215 && c.B == 0 && c.A == 255 {
				goldCount++
			}
		}
	}
	if goldCount != w*h {
		t.Fatalf("Expected initial buffer to have %d gold pixels, got %d", w*h, goldCount)
	}

	// Step 2: FillRect full-screen with Black (0, 0, 0)
	g.SetColor(0, 0, 0)
	g.FillRect(0, 0, w, h)

	// Step 3: Empirically inspect every single pixel in the buffer
	blackCount := 0
	nonBlackCount := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := g.Buffer.RGBAAt(x, y)
			if c.R == 0 && c.G == 0 && c.B == 0 && c.A == 255 {
				blackCount++
			} else {
				nonBlackCount++
			}
		}
	}

	t.Logf("Full-screen black FillRect result: %d black pixels, %d non-black pixels out of %d total pixels",
		blackCount, nonBlackCount, w*h)

	if nonBlackCount > 0 {
		t.Fatalf("FAIL: Erase suppression hack detected or FillRect failed! %d pixels were not cleared to black", nonBlackCount)
	}
	if blackCount != w*h {
		t.Fatalf("FAIL: Expected %d black pixels, got %d", w*h, blackCount)
	}
}

func TestEmpirical_FillRect_ClippedAndTranslatedClearing(t *testing.T) {
	w, h := 100, 100
	g := NewGraphics(w, h)

	// Fill background with Red
	g.SetColor(255, 0, 0)
	g.FillRect(0, 0, w, h)

	// Set clip to (20, 20, 50, 50)
	g.SetClip(20, 20, 50, 50)

	// FillRect (0, 0, 100, 100) with Black
	g.SetColor(0, 0, 0)
	g.FillRect(0, 0, w, h)

	// Check pixels: inside clip (20..69, 20..69) must be Black, outside clip must remain Red
	blackCount := 0
	redCount := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := g.Buffer.RGBAAt(x, y)
			if x >= 20 && x < 70 && y >= 20 && y < 70 {
				if c == (color.RGBA{R: 0, G: 0, B: 0, A: 255}) {
					blackCount++
				}
			} else {
				if c == (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
					redCount++
				}
			}
		}
	}

	if blackCount != 50*50 {
		t.Errorf("Expected 2500 black pixels inside clip, got %d", blackCount)
	}
	if redCount != (100*100 - 50*50) {
		t.Errorf("Expected %d red pixels outside clip, got %d", 100*100-50*50, redCount)
	}
}
