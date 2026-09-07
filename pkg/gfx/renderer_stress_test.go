package gfx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"java-app/pkg/midp"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestContinuousBufferFlushing stress tests offscreen buffer flushing across 20 consecutive paint cycles.
func TestContinuousBufferFlushing(t *testing.T) {
	tmpDir := t.TempDir()
	paths := []string{
		filepath.Join(tmpDir, "tmp_emulator_buffer.png"),
		filepath.Join(tmpDir, "emulator_buffer.png"),
	}

	g := midp.NewGraphics(240, 320)
	g.SetColor(0, 255, 0) // Green
	g.FillRect(10, 10, 100, 100)

	game := &EmulatorGame{
		Graphics:    g,
		BufferPaths: paths,
	}

	screen := ebiten.NewImage(240, 320)

	// Run 20 consecutive draw/flush frames
	for frame := 1; frame <= 20; frame++ {
		// Change color on frame 10 to simulate animated paint updates
		if frame == 10 {
			g.SetColor(255, 0, 0) // Red
			g.FillRect(50, 50, 80, 80)
		}

		game.Draw(screen)

		for _, p := range paths {
			info, err := os.Stat(p)
			if err != nil {
				t.Fatalf("Frame %d: file %s does not exist: %v", frame, p, err)
			}
			if info.Size() < 100 {
				t.Fatalf("Frame %d: file %s size too small (%d bytes)", frame, p, info.Size())
			}

			// Read file header and verify PNG magic bytes
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("Frame %d: failed to read %s: %v", frame, p, err)
			}
			pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			if !bytes.HasPrefix(data, pngHeader) {
				t.Fatalf("Frame %d: file %s is not a valid PNG image", frame, p)
			}
		}
	}
}
