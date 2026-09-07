package midp

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// TestFontGlyphRenderingBounds_EmptyString tests rendering an empty string.
func TestFontGlyphRenderingBounds_EmptyString(t *testing.T) {
	g := NewGraphics(240, 320)
	// Background color is (15, 25, 45, 255)
	g.SetColor(255, 255, 255)
	g.DrawString("", 10, 10, 0)

	for y := 0; y < 320; y++ {
		for x := 0; x < 240; x++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if (r >> 8) != 15 || (gVal >> 8) != 25 || (b >> 8) != 45 {
				t.Fatalf("expected untouched background at (%d,%d), got r=%d g=%d b=%d", x, y, r>>8, gVal>>8, b>>8)
			}
		}
	}
}

// TestFontGlyphRenderingBounds_LongString tests rendering a very long string exceeding screen width.
func TestFontGlyphRenderingBounds_LongString(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 255, 255)
	longStr := strings.Repeat("A", 200) // 200 * 8 = 1600 px wide
	g.DrawString(longStr, 10, 10, 0)

	// Pixels within canvas (10..239) should have glyph pixels drawn
	drawnCount := 0
	for x := 10; x < 240; x++ {
		for y := 10; y < 23; y++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
				drawnCount++
			}
		}
	}
	if drawnCount == 0 {
		t.Errorf("expected rendered glyph pixels for long string within canvas bounds, got 0")
	}
}

// TestFontGlyphRenderingBounds_SpecialChars tests rendering special characters and non-ASCII runes.
func TestFontGlyphRenderingBounds_SpecialChars(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 255, 255)
	specialStr := "\x01\x02ñ€✓\t\r"
	g.DrawString(specialStr, 10, 10, 0)

	// Verify glyphs were drawn using default fallback glyph without panicking
	drawnCount := 0
	for x := 10; x < 10+len([]rune(specialStr))*8; x++ {
		for y := 10; y < 23; y++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
				drawnCount++
			}
		}
	}
	if drawnCount == 0 {
		t.Errorf("expected rendered fallback glyph pixels for special characters, got 0")
	}
}

// TestFontGlyphRenderingBounds_MultiLine tests behavior of DrawString with embedded newline characters.
func TestFontGlyphRenderingBounds_MultiLine(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 255, 255)
	multiLineStr := "Line1\nLine2"
	g.DrawString(multiLineStr, 10, 10, 0)

	// DrawString treats \n as a single rune in a single row (11 runes * 8 = 88 px wide, height 13 px)
	// Verify pixels in row y=10..22 are drawn, and y=25..35 are NOT drawn (no line break)
	drawnCountRow1 := 0
	for x := 10; x < 98; x++ {
		for y := 10; y < 23; y++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
				drawnCountRow1++
			}
		}
	}
	if drawnCountRow1 == 0 {
		t.Errorf("expected glyph pixels drawn for multi-line string on single row, got 0")
	}

	drawnCountRow2 := 0
	for x := 10; x < 98; x++ {
		for y := 25; y < 38; y++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
				drawnCountRow2++
			}
		}
	}
	if drawnCountRow2 > 0 {
		t.Errorf("DrawString unexpectedly performed multi-line wrapping for \\n (drawnCountRow2=%d)", drawnCountRow2)
	}
}

// TestFontGlyphRendering_Anchors tests all MIDP 2.0 anchor combinations for DrawString.
func TestFontGlyphRendering_Anchors(t *testing.T) {
	testCases := []struct {
		name       string
		anchor     int
		expectMinX int
		expectMinY int
	}{
		{"TOP_LEFT", 0, 100, 100},                   // LEFT=0, TOP=0
		{"TOP_RIGHT", 8, 100 - 16, 100},             // RIGHT=8, TOP=0 ("HI" is 16px wide)
		{"TOP_HCENTER", 1, 100 - 8, 100},            // HCENTER=1, TOP=0
		{"BOTTOM_LEFT", 32, 100, 100 - 13},          // LEFT=0, BOTTOM=32 (height=13)
		{"VCENTER_HCENTER", 1 | 2, 100 - 8, 100 - 6}, // HCENTER=1, VCENTER=2 (13/2=6)
		{"BASELINE_LEFT", 64, 100, 100 - 10},        // LEFT=0, BASELINE=64 (offset 10)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGraphics(240, 320)
			g.SetColor(255, 255, 255)
			g.DrawString("HI", 100, 100, tc.anchor)

			// Find bounding box of drawn pixels
			minX, minY, maxX, maxY := 999, 999, -1, -1
			for y := 0; y < 320; y++ {
				for x := 0; x < 240; x++ {
					r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
					if r > 0x8000 && gVal > 0x8000 && b > 0x8000 {
						if x < minX {
							minX = x
						}
						if x > maxX {
							maxX = x
						}
						if y < minY {
							minY = y
						}
						if y > maxY {
							maxY = y
						}
					}
				}
			}

			if minX == 999 {
				t.Fatalf("%s: no pixels rendered", tc.name)
			}
			if minX < tc.expectMinX || minY < tc.expectMinY {
				t.Errorf("%s: expected bounding box starting near (%d,%d), got min(%d,%d) max(%d,%d)",
					tc.name, tc.expectMinX, tc.expectMinY, minX, minY, maxX, maxY)
			}
		})
	}
}

// TestFontGlyphRendering_TextColorOverrideBug stress tests font text color rendering for black text.
func TestFontGlyphRendering_TextColorOverrideBug(t *testing.T) {
	g := NewGraphics(240, 320)
	// Clear buffer to white background so black text would be visible
	for y := 0; y < 320; y++ {
		for x := 0; x < 240; x++ {
			g.Buffer.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	// Set color to Black (0, 0, 0)
	g.SetColor(0, 0, 0)
	g.DrawString("HI", 10, 10, 0)

	// Check color of rendered glyph pixels
	blackGlyphPixels := 0
	yellowGlyphPixels := 0
	for y := 10; y < 23; y++ {
		for x := 10; x < 26; x++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			// Pure black pixel in 16-bit RGBA is (0, 0, 0, 65535)
			if r == 0 && gVal == 0 && b == 0 {
				blackGlyphPixels++
			}
			// Yellow pixel is (65535, 56540, 12850, 65535)
			if r > 0xE000 && gVal > 0xC000 && b < 0x4000 {
				yellowGlyphPixels++
			}
		}
	}

	t.Logf("Black text render stats: blackPixels=%d, yellowPixels=%d", blackGlyphPixels, yellowGlyphPixels)
	if yellowGlyphPixels > 0 && blackGlyphPixels == 0 {
		t.Errorf("BUG DETECTED: SetColor(0,0,0) rendered yellow text (%d pixels) instead of black text!", yellowGlyphPixels)
	}
}

// TestGraphicsClipping_EdgeCases stress tests edge cases for Graphics clip rect state.
func TestGraphicsClipping_EdgeCases(t *testing.T) {
	g := NewGraphics(240, 320)

	// Case 1: Negative width and height in SetClip
	g.SetClip(10, 10, -20, -30)
	if g.ClipW != -20 || g.ClipH != -30 {
		t.Errorf("SetClip failed to record negative width/height: got w=%d h=%d", g.ClipW, g.ClipH)
	}
	dst := g.getClippedDst(image.Rect(0, 0, 100, 100))
	if !dst.Empty() {
		t.Errorf("getClippedDst returned non-empty rect for negative clip dimensions: %v", dst)
	}

	// Case 2: Clip outside canvas bounds
	g.SetClip(300, 400, 50, 50)
	g.SetColor(255, 0, 0)
	g.FillRect(300, 400, 50, 50) // Should not draw anything on 240x320 canvas
	dst = g.getClippedDst(image.Rect(300, 400, 350, 450))
	if !dst.Empty() {
		t.Errorf("getClippedDst returned non-empty rect for offscreen clip: %v", dst)
	}

	// Case 3: ClipRect intersection resulting in empty clip
	g.SetClip(10, 10, 20, 20)
	g.ClipRect(50, 50, 20, 20) // Disjoint rectangle
	if g.ClipW != 0 || g.ClipH != 0 {
		t.Errorf("ClipRect on disjoint rectangle expected ClipW=0 ClipH=0, got w=%d h=%d", g.ClipW, g.ClipH)
	}
}
