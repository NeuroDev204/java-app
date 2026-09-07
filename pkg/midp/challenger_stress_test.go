package midp

import (
	"image"
	"image/color"
	"testing"
)

// TestChallenger_DrawString_TextColorVerification empirically stress tests text color rendering in DrawString.
// Verifies exact RGBA pixel values in g.Buffer for:
// - Black (0, 0, 0)
// - Dark Gray (30, 30, 30), (15, 15, 15), (5, 5, 5)
// - Blue (0, 0, 255)
// - Red (255, 0, 0)
// - Green (0, 255, 0)
// - Custom RGBA colors (128, 64, 192, 255)
// - Zero-alpha white fallback (100, 100, 100, 0) -> white (255, 255, 255, 255)
func TestChallenger_DrawString_TextColorVerification(t *testing.T) {
	testCases := []struct {
		name         string
		setColorFunc func(g *Graphics)
		expectedRGBA color.RGBA
	}{
		{
			name: "Black_0_0_0",
			setColorFunc: func(g *Graphics) {
				g.SetColor(0, 0, 0)
			},
			expectedRGBA: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name: "DarkGray_30_30_30",
			setColorFunc: func(g *Graphics) {
				g.SetColor(30, 30, 30)
			},
			expectedRGBA: color.RGBA{R: 30, G: 30, B: 30, A: 255},
		},
		{
			name: "DarkGray_15_15_15",
			setColorFunc: func(g *Graphics) {
				g.SetColor(15, 15, 15)
			},
			expectedRGBA: color.RGBA{R: 15, G: 15, B: 15, A: 255},
		},
		{
			name: "DarkGray_5_5_5",
			setColorFunc: func(g *Graphics) {
				g.SetColor(5, 5, 5)
			},
			expectedRGBA: color.RGBA{R: 5, G: 5, B: 5, A: 255},
		},
		{
			name: "Blue_0_0_255",
			setColorFunc: func(g *Graphics) {
				g.SetColor(0, 0, 255)
			},
			expectedRGBA: color.RGBA{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name: "Red_255_0_0",
			setColorFunc: func(g *Graphics) {
				g.SetColor(255, 0, 0)
			},
			expectedRGBA: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name: "Green_0_255_0",
			setColorFunc: func(g *Graphics) {
				g.SetColor(0, 255, 0)
			},
			expectedRGBA: color.RGBA{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name: "CustomRGBA_128_64_192",
			setColorFunc: func(g *Graphics) {
				g.CurrentColor = color.RGBA{R: 128, G: 64, B: 192, A: 255}
			},
			expectedRGBA: color.RGBA{R: 128, G: 64, B: 192, A: 255},
		},
		{
			name: "ZeroAlpha_WhiteFallback",
			setColorFunc: func(g *Graphics) {
				g.CurrentColor = color.RGBA{R: 100, G: 100, B: 100, A: 0}
			},
			expectedRGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create canvas 240x320 with orange background (255, 128, 0) to contrast against tested text colors
			bg := color.RGBA{R: 255, G: 128, B: 0, A: 255}
			g := NewGraphics(240, 320)
			for y := 0; y < 320; y++ {
				for x := 0; x < 240; x++ {
					g.Buffer.SetRGBA(x, y, bg)
				}
			}

			tc.setColorFunc(g)
			// Draw character 'X' at (20, 20)
			g.DrawString("X", 20, 20, 0)

			// Scan buffer for rendered glyph pixels
			glyphPixelCount := 0
			matchingPixelCount := 0

			for y := 20; y < 20+13; y++ {
				for x := 20; x < 20+8; x++ {
					pix := g.Buffer.RGBAAt(x, y)
					if pix != bg {
						glyphPixelCount++
						if pix == tc.expectedRGBA {
							matchingPixelCount++
						} else {
							t.Errorf("%s: rendered pixel at (%d,%d) has RGBA=%v, expected exact RGBA=%v",
								tc.name, x, y, pix, tc.expectedRGBA)
						}
					}
				}
			}

			if glyphPixelCount == 0 {
				t.Fatalf("%s: expected non-zero rendered glyph pixels for 'X', got 0", tc.name)
			}

			if matchingPixelCount != glyphPixelCount {
				t.Errorf("%s: %d/%d pixels matched expected RGBA", tc.name, matchingPixelCount, glyphPixelCount)
			}
		})
	}
}

// TestChallenger_GetClippedDst_OffscreenClipBounds empirically stress tests clip bound calculations.
func TestChallenger_GetClippedDst_OffscreenClipBounds(t *testing.T) {
	g := NewGraphics(240, 320)

	t.Run("CompletelyOutside_XGreaterThanWidth", func(t *testing.T) {
		g.SetClip(300, 50, 50, 50)
		dst := g.getClippedDst(image.Rect(300, 50, 350, 100))
		if !dst.Empty() || dst != image.ZR {
			t.Errorf("expected image.ZR for x > width clip, got %v", dst)
		}
	})

	t.Run("CompletelyOutside_YGreaterThanHeight", func(t *testing.T) {
		g.SetClip(50, 400, 50, 50)
		dst := g.getClippedDst(image.Rect(50, 400, 100, 450))
		if !dst.Empty() || dst != image.ZR {
			t.Errorf("expected image.ZR for y > height clip, got %v", dst)
		}
	})

	t.Run("CompletelyOutside_NegativeCoordinates", func(t *testing.T) {
		g.SetClip(-100, -100, 50, 50)
		dst := g.getClippedDst(image.Rect(-100, -100, -50, -50))
		if !dst.Empty() || dst != image.ZR {
			t.Errorf("expected image.ZR for negative clip coordinates outside canvas, got %v", dst)
		}
	})

	t.Run("CompletelyOutside_NegativeWidthHeight", func(t *testing.T) {
		g.SetClip(10, 10, -20, -30)
		dst := g.getClippedDst(image.Rect(0, 0, 100, 100))
		if !dst.Empty() || dst != image.ZR {
			t.Errorf("expected image.ZR for negative width/height, got %v", dst)
		}
	})

	t.Run("PartiallyOverlapping_TopLeftBoundary", func(t *testing.T) {
		g.SetClip(-10, -10, 30, 30) // absolute rect: (-10, -10) to (20, 20)
		dst := g.getClippedDst(image.Rect(-10, -10, 20, 20))
		expected := image.Rect(0, 0, 20, 20)
		if dst != expected {
			t.Errorf("expected clipped dst %v for top-left overlap, got %v", expected, dst)
		}
	})

	t.Run("PartiallyOverlapping_BottomRightBoundary", func(t *testing.T) {
		g.SetClip(220, 300, 40, 40) // absolute rect: (220, 300) to (260, 340)
		dst := g.getClippedDst(image.Rect(200, 280, 260, 340))
		expected := image.Rect(220, 300, 240, 320)
		if dst != expected {
			t.Errorf("expected clipped dst %v for bottom-right overlap, got %v", expected, dst)
		}
	})

	t.Run("EqualCanvasBounds", func(t *testing.T) {
		g.SetClip(0, 0, 240, 320)
		dst := g.getClippedDst(image.Rect(0, 0, 240, 320))
		expected := image.Rect(0, 0, 240, 320)
		if dst != expected {
			t.Errorf("expected clipped dst %v for full canvas clip, got %v", expected, dst)
		}

		dstInner := g.getClippedDst(image.Rect(10, 20, 50, 60))
		expectedInner := image.Rect(10, 20, 50, 60)
		if dstInner != expectedInner {
			t.Errorf("expected clipped dst %v for inner rect, got %v", expectedInner, dstInner)
		}
	})

	t.Run("Translation_WithOffscreenClip", func(t *testing.T) {
		g.TransX, g.TransY = 0, 0 // Reset
		g.Translate(200, 300)
		g.SetClip(50, 50, 40, 40) // absolute rect: (250, 350) to (290, 390) -> completely outside 240x320
		dst := g.getClippedDst(image.Rect(250, 350, 290, 390))
		if !dst.Empty() || dst != image.ZR {
			t.Errorf("expected image.ZR for translated offscreen clip, got %v", dst)
		}
	})
}
