package gfx

import (
	"os"
	"path/filepath"
	"testing"

	"java-app/pkg/midp"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestKeycodeMapping(t *testing.T) {
	j2meKey := MapEbitenKeyToJ2ME(ebiten.KeyNumpad1)
	if j2meKey != '1' {
		t.Errorf("expected J2ME key '1', got %d", j2meKey)
	}
}

func TestBufferFlushing(t *testing.T) {
	tmpDir := t.TempDir()
	p1 := filepath.Join(tmpDir, "tmp_emulator_buffer.png")
	p2 := filepath.Join(tmpDir, "emulator_buffer.png")
	paths := []string{p1, p2}

	game := &EmulatorGame{
		Graphics:    midp.NewGraphics(240, 320),
		BufferPaths: paths,
	}
	game.Graphics.SetColor(255, 0, 0)
	game.Graphics.FillRect(0, 0, 50, 50)

	screen := ebiten.NewImage(240, 320)
	game.Draw(screen)

	// Verify both buffer files were flushed
	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected buffer file at %s after Draw, but it was not created", p)
		}
	}
}

func TestDefaultBufferFlushingPath(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env_emulator_buffer.png")
	t.Setenv("EMULATOR_BUFFER_PATH", envFile)

	game := &EmulatorGame{
		Graphics: midp.NewGraphics(240, 320),
	}
	game.Graphics.SetColor(0, 0, 255)
	game.Graphics.FillRect(0, 0, 20, 20)

	screen := ebiten.NewImage(240, 320)
	game.Draw(screen)

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Errorf("expected default buffer file via env EMULATOR_BUFFER_PATH at %s, but it was not created", envFile)
	}
}
