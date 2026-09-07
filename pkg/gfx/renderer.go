package gfx

/*
#cgo LDFLAGS: -L${SRCDIR}/../../lib
*/
import "C"

import (
	"image"
	"image/png"
	"log"
	"os"
	"java-app/pkg/midp"
	"github.com/hajimehoshi/ebiten/v2"
)

type EmulatorGame struct {
	Graphics      *midp.Graphics
	PaintCallback func()
	InputCallback func(keyCode int)
	offscreen     *ebiten.Image
	pressedKeys   map[ebiten.Key]bool
	BufferPaths   []string
}

func (g *EmulatorGame) Update() error {
	if g.pressedKeys == nil {
		g.pressedKeys = make(map[ebiten.Key]bool)
	}

	keysToCheck := []ebiten.Key{
		ebiten.KeyEnter, ebiten.KeySpace, ebiten.KeyUp, ebiten.KeyDown,
		ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyDigit0, ebiten.KeyDigit1,
		ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
		ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
		ebiten.KeyNumpad0, ebiten.KeyNumpad1, ebiten.KeyNumpad2, ebiten.KeyNumpad3,
		ebiten.KeyNumpad4, ebiten.KeyNumpad5, ebiten.KeyNumpad6, ebiten.KeyNumpad7,
		ebiten.KeyNumpad8, ebiten.KeyNumpad9,
	}

	for _, k := range keysToCheck {
		if ebiten.IsKeyPressed(k) {
			if !g.pressedKeys[k] {
				g.pressedKeys[k] = true
				if g.InputCallback != nil {
					jKey := MapEbitenKeyToJ2ME(k)
					g.InputCallback(jKey)
				}
			}
		} else {
			g.pressedKeys[k] = false
		}
	}

	if g.PaintCallback != nil {
		g.PaintCallback()
	}
	return nil
}

var drawLogCnt = 0

func flushBufferToDisk(buf *image.RGBA, paths []string) {
	if len(paths) == 0 {
		if envPath := os.Getenv("EMULATOR_BUFFER_PATH"); envPath != "" {
			paths = []string{envPath}
		} else {
			paths = []string{"/tmp/emulator_buffer.png", "emulator_buffer.png"}
		}
	}
	for _, p := range paths {
		tmpPath := p + ".tmp"
		f, err := os.Create(tmpPath)
		if err != nil {
			continue
		}

		encodeErr := png.Encode(f, buf)
		syncErr := f.Sync()
		closeErr := f.Close()

		if encodeErr == nil && syncErr == nil && closeErr == nil {
			_ = os.Rename(tmpPath, p)
		} else {
			_ = os.Remove(tmpPath)
		}
	}
}

func (g *EmulatorGame) Draw(screen *ebiten.Image) {
	if g.Graphics != nil && g.Graphics.Buffer != nil {
		if g.offscreen == nil {
			g.offscreen = ebiten.NewImage(g.Graphics.Width, g.Graphics.Height)
		}
		if drawLogCnt < 5 {
			drawLogCnt++
			nonZero := 0
			for _, b := range g.Graphics.Buffer.Pix {
				if b != 0 {
					nonZero++
				}
			}
			pix := g.Graphics.Buffer.Pix
			log.Printf("[RENDER-TRACE] g.Graphics.Buffer Pix non-zero = %d / %d | Sample[0..15] = (%d,%d,%d,%d), (%d,%d,%d,%d), (%d,%d,%d,%d), (%d,%d,%d,%d)",
				nonZero, len(pix),
				pix[0], pix[1], pix[2], pix[3],
				pix[4], pix[5], pix[6], pix[7],
				pix[8], pix[9], pix[10], pix[11],
				pix[12], pix[13], pix[14], pix[15])
		}
		flushBufferToDisk(g.Graphics.Buffer, g.BufferPaths)
		g.offscreen.WritePixels(g.Graphics.Buffer.Pix)
		screen.DrawImage(g.offscreen, nil)
	}
}

func (g *EmulatorGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 320
}
