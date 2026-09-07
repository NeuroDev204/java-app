package main

/*
#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux
#cgo LDFLAGS: -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm
*/
import "C"

import (
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"java-app/pkg/gfx"
	"java-app/pkg/jvm"
	"java-app/pkg/midp"
)

const AppTitle = "J2ME Go Emulator"

type Game struct {
	graphics  *midp.Graphics
	keyStates map[ebiten.Key]bool
}

func (g *Game) Update() error {
	allKeys := []ebiten.Key{
		ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7,
		ebiten.KeyDigit8, ebiten.KeyDigit9,
		ebiten.KeyNumpad0, ebiten.KeyNumpad1, ebiten.KeyNumpad2, ebiten.KeyNumpad3,
		ebiten.KeyNumpad4, ebiten.KeyNumpad5, ebiten.KeyNumpad6, ebiten.KeyNumpad7,
		ebiten.KeyNumpad8, ebiten.KeyNumpad9,
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyEnter,
	}

	for _, k := range allKeys {
		isDown := ebiten.IsKeyPressed(k)
		wasDown := g.keyStates[k]
		if isDown != wasDown {
			g.keyStates[k] = isDown
			j2meKey := gfx.MapEbitenKeyToJ2ME(k)
			if j2meKey != 0 {
				jvm.DispatchKeyEvent(j2meKey, isDown)
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.graphics != nil && g.graphics.Buffer != nil {
		ebitenImage := ebiten.NewImageFromImage(g.graphics.Buffer)
		screen.DrawImage(ebitenImage, nil)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 320
}

func main() {
	runtime.LockOSThread()

	var jarPath string
	if len(os.Args) >= 2 {
		jarPath = os.Args[1]
	} else {
		jarPath = "/home/neuro/Downloads/KnightOnline_402.jar"
	}

	jarPath = strings.TrimSpace(jarPath)
	log.Printf("[J2ME Launcher] Target JAR: %s", jarPath)

	graphics := midp.NewGraphics(240, 320)
	jvm.SetActiveGraphics(graphics)

	vm, err := jvm.InitJVM("./j2me-stubs/bin")
	if err != nil {
		log.Fatalf("Failed to initialize JVM via CGO: %v", err)
	}
	log.Printf("Embedded HotSpot JVM initialized via CGO/JNI")

	if err := vm.RegisterNatives(); err != nil {
		log.Fatalf("Failed to register JNI native stubs: %v", err)
	}

	// Invoke com.emulator.Main via JNI in background thread to execute J2ME game
	go func() {
		if err := vm.InvokeMainWithArgs("com/emulator/Main", []string{jarPath}); err != nil {
			log.Printf("com.emulator.Main execution notice: %v", err)
		}
	}()

	ebiten.SetWindowSize(480, 640)
	ebiten.SetWindowTitle(AppTitle)

	game := &Game{
		graphics:  graphics,
		keyStates: make(map[ebiten.Key]bool),
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
