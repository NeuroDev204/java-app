package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"java-app/pkg/gfx"
	"java-app/pkg/midp"
	"java-app/pkg/vm"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestChallenger_E2EPixelThresholdBoundaries tests the pixel count threshold (>50,000 non-zero pixels).
func TestChallenger_E2EPixelThresholdBoundaries(t *testing.T) {
	// Screen size 240x320 = 76,800 total pixels
	g := midp.NewGraphics(240, 320)
	
	// Default NewGraphics fills background with #0F192D (R=15, G=25, B=45, A=255)
	// Verify that all 76,800 pixels are non-zero
	nonZeroCount := 0
	for y := 0; y < 320; y++ {
		for x := 0; x < 240; x++ {
			r, gVal, b, a := g.Buffer.At(x, y).RGBA()
			if r != 0 || gVal != 0 || b != 0 || a != 0 {
				nonZeroCount++
			}
		}
	}
	if nonZeroCount != 76800 {
		t.Fatalf("Expected 76,800 non-zero pixels in default canvas, got %d", nonZeroCount)
	}
	if nonZeroCount <= 50000 {
		t.Fatalf("Pixel count %d is not strictly > 50,000 threshold", nonZeroCount)
	}

	t.Logf("[CHALLENGER-E2E] Default canvas non-zero pixel count: %d / 76800 (Passed > 50,000 threshold)", nonZeroCount)
}

// TestChallenger_E2EButtonColorRegions verifies exact button colors and UI element regions.
func TestChallenger_E2EButtonColorRegions(t *testing.T) {
	g := midp.NewGraphics(240, 320)

	// Button 1: Crimson (160, 30, 30) at (40, 100, 160, 30)
	g.SetColor(160, 30, 30)
	g.FillRect(40, 100, 160, 30)

	// Button 2: Blue (30, 80, 160) at (40, 145, 160, 30)
	g.SetColor(30, 80, 160)
	g.FillRect(40, 145, 160, 30)

	// Sample center of Button 1 (120, 115)
	r1, g1, b1, a1 := g.Buffer.At(120, 115).RGBA()
	// RGBA returns 16-bit values scaled from 8-bit (e.g. 160 -> 0xA0A0)
	r18, g18, b18, a18 := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8), uint8(a1>>8)
	if r18 != 160 || g18 != 30 || b18 != 30 || a18 != 255 {
		t.Fatalf("Button 1 color mismatch: expected RGBA(160,30,30,255), got RGBA(%d,%d,%d,%d)", r18, g18, b18, a18)
	}

	// Sample center of Button 2 (120, 160)
	r2, g2, b2, a2 := g.Buffer.At(120, 160).RGBA()
	r28, g28, b28, a28 := uint8(r2>>8), uint8(g2>>8), uint8(b2>>8), uint8(a2>>8)
	if r28 != 30 || g28 != 80 || b28 != 160 || a28 != 255 {
		t.Fatalf("Button 2 color mismatch: expected RGBA(30,80,160,255), got RGBA(%d,%d,%d,%d)", r28, g28, b28, a28)
	}

	t.Logf("[CHALLENGER-E2E] Button color sampling verified: Btn1=(%d,%d,%d), Btn2=(%d,%d,%d)", r18, g18, b18, r28, g28, b28)
}

// TestChallenger_E2EBitmapFontRendering verifies font rendering non-zero pixels inside text bounding box.
func TestChallenger_E2EBitmapFontRendering(t *testing.T) {
	g := midp.NewGraphics(240, 320)
	
	// Draw Gold Title "KNIGHT ONLINE J2ME"
	g.SetColor(255, 215, 0)
	text := "KNIGHT ONLINE J2ME"
	g.DrawString(text, 120, 20, 1) // HCENTER=1

	// Text length 18 chars * 8px = 144px width, 13px height.
	// Centered at X=120 -> X from (120-72)=48 to 192. Y from 20 to 33.
	fontPixels := 0
	for y := 20; y < 33; y++ {
		for x := 48; x < 192; x++ {
			r, gVal, b, _ := g.Buffer.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(gVal>>8), uint8(b>>8)
			if r8 == 255 && g8 == 215 && b8 == 0 {
				fontPixels++
			}
		}
	}

	if fontPixels == 0 {
		t.Fatalf("E2E Bitmap Font test failed: zero gold font pixels found in text bounding box!")
	}
	t.Logf("[CHALLENGER-E2E] Bitmap font rendering verified: found %d gold font pixels for string '%s'", fontPixels, text)
}

// TestChallenger_E2EBufferClearing verifies genuine full-screen black FillRect clears the screen buffer.
func TestChallenger_E2EBufferClearing(t *testing.T) {
	g := midp.NewGraphics(240, 320)
	
	// Draw initial state (Gold rect)
	g.SetColor(255, 215, 0)
	g.FillRect(50, 50, 100, 100)

	// Perform full-screen black clear (0, 0, 240, 320)
	g.SetColor(0, 0, 0)
	g.FillRect(0, 0, 240, 320)

	// Verify buffer at (60, 60) was genuinely cleared to black
	r, gVal, b, _ := g.Buffer.At(60, 60).RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(gVal>>8), uint8(b>>8)
	if r8 != 0 || g8 != 0 || b8 != 0 {
		t.Fatalf("Full-screen black FillRect failed to clear buffer! Expected Black at (60,60), got RGBA(%d,%d,%d)", r8, g8, b8)
	}

	t.Logf("[CHALLENGER-E2E] Genuine buffer clearing verified: Full-screen black FillRect clears screen buffer")
}

// TestChallenger_E2EPNGContinuousFlushingStress tests continuous frame flushing and PNG file integrity.
func TestChallenger_E2EPNGContinuousFlushingStress(t *testing.T) {
	tmpDir := t.TempDir()
	bufPath := filepath.Join(tmpDir, "stress_buffer.png")

	g := midp.NewGraphics(240, 320)
	vmCtx := vm.NewVMContext(nil, nil)
	_ = vmCtx

	game := &gfx.EmulatorGame{
		Graphics:    g,
		BufferPaths: []string{bufPath},
	}
	screen := ebiten.NewImage(240, 320)

	// Run 30 consecutive frame render cycles
	for frame := 0; frame < 30; frame++ {
		// Change color per frame
		cVal := uint8((frame * 7) % 250)
		g.SetColor(int(cVal), 100, 200)
		g.FillRect(10, 10, 50, 50)

		game.Draw(screen)

		// Decode PNG and verify file validity
		f, err := os.Open(bufPath)
		if err != nil {
			t.Fatalf("Frame %d: PNG buffer file not found: %v", frame, err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatalf("Frame %d: PNG decode failed (truncated file): %v", frame, err)
		}

		if img.Bounds().Dx() != 240 || img.Bounds().Dy() != 320 {
			t.Fatalf("Frame %d: PNG bounds mismatch: expected 240x320, got %v", frame, img.Bounds())
		}
	}

	// Multi-goroutine concurrent flush stress test
	var wg sync.WaitGroup
	for worker := 0; worker < 5; worker++ {
		wg.Add(1)
		go func(wId int) {
			defer wg.Done()
			wPath := filepath.Join(tmpDir, fmt.Sprintf("worker_%d_buffer.png", wId))
			wGame := &gfx.EmulatorGame{
				Graphics:    g,
				BufferPaths: []string{wPath},
			}
			wScreen := ebiten.NewImage(240, 320)
			for i := 0; i < 10; i++ {
				wGame.Draw(wScreen)
				f, err := os.Open(wPath)
				if err != nil {
					t.Errorf("Worker %d cycle %d: PNG open error: %v", wId, i, err)
					return
				}
				_, err = png.Decode(f)
				f.Close()
				if err != nil {
					t.Errorf("Worker %d cycle %d: PNG decode error: %v", wId, i, err)
					return
				}
			}
		}(worker)
	}
	wg.Wait()

	t.Logf("[CHALLENGER-E2E] PNG continuous flushing and multi-threaded stress test PASSED cleanly")
}
