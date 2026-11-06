package main

/*
#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux
#cgo LDFLAGS: -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm
*/
import "C"

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/sqweek/dialog"

	"java-app/pkg/autoupgrade"
	"java-app/pkg/gfx"
	"java-app/pkg/jvm"
	"java-app/pkg/midp"
)

// Game implements the ebiten.Game interface for the J2ME Go Engine.
type Game struct {
	graphics       *midp.Graphics
	tickCount      uint64
	keyStates      map[ebiten.Key]bool
	lastKey        string
	showMenu       bool
	selectedJar    string
	menuIndex      int // 0 = Start Game, 1 = Select JAR
	upgradeOverlay *autoupgrade.Overlay
}

// Update proceeds the game state (60 FPS default).
func (g *Game) Update() error {
	g.tickCount++

	// Menu state - don't process game input while in menu
	if g.showMenu {
		// Navigate menu with Up/Down
		if ebiten.IsKeyPressed(ebiten.KeyDown) && !g.keyStates[ebiten.KeyDown] {
			g.keyStates[ebiten.KeyDown] = true
			g.menuIndex = (g.menuIndex + 1) % 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyUp) && !g.keyStates[ebiten.KeyUp] {
			g.keyStates[ebiten.KeyUp] = true
			g.menuIndex = (g.menuIndex + 1) % 2
		}

		// Enter to select
		if ebiten.IsKeyPressed(ebiten.KeyEnter) && !g.keyStates[ebiten.KeyEnter] {
			g.keyStates[ebiten.KeyEnter] = true
			if g.menuIndex == 0 {
				// Start Game - only if JAR is selected
				if g.selectedJar != "" {
					g.showMenu = false
					go launchGame(g.selectedJar)
				}
			} else {
				// Select JAR
				filePath, err := dialog.File().Filter("JAR files", "jar").Load()
				if err == nil && filePath != "" {
					g.selectedJar = filePath
				}
			}
		}

		g.keyStates[ebiten.KeyDown] = ebiten.IsKeyPressed(ebiten.KeyDown)
		g.keyStates[ebiten.KeyUp] = ebiten.IsKeyPressed(ebiten.KeyUp)
		g.keyStates[ebiten.KeyEnter] = ebiten.IsKeyPressed(ebiten.KeyEnter)
		return nil
	}

	globalVm := jvm.GetGlobalJVM()
	if globalVm == nil {
		return nil
	}

	// In-game F3 Auto Upgrade Overlay
	if g.upgradeOverlay != nil {
		g.upgradeOverlay.Update()
		keyMap := make(map[ebiten.Key]bool)
		for _, k := range []ebiten.Key{
			ebiten.KeyF3, ebiten.KeyEscape, ebiten.KeyUp, ebiten.KeyDown,
			ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyEnter,
		} {
			if ebiten.IsKeyPressed(k) {
				keyMap[k] = true
			}
		}
		g.upgradeOverlay.HandleInput(keyMap)
		if g.upgradeOverlay.IsVisible() {
			return nil
		}
	}

	if jvm.IsTextInputActive() {
		chars := ebiten.InputChars()
		backspace := ebiten.IsKeyPressed(ebiten.KeyBackspace) && !g.keyStates[ebiten.KeyBackspace]
		enter := ebiten.IsKeyPressed(ebiten.KeyEnter) && !g.keyStates[ebiten.KeyEnter]

		g.keyStates[ebiten.KeyBackspace] = ebiten.IsKeyPressed(ebiten.KeyBackspace)
		g.keyStates[ebiten.KeyEnter] = ebiten.IsKeyPressed(ebiten.KeyEnter)

		jvm.ProcessTextInput(chars, backspace, enter)
		return nil
	}

	// Forward real user input from Ebitengine to HotSpot JVM via JNI
	allKeys := []ebiten.Key{
		ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7,
		ebiten.KeyDigit8, ebiten.KeyDigit9,
		ebiten.KeyNumpad0, ebiten.KeyNumpad1, ebiten.KeyNumpad2, ebiten.KeyNumpad3,
		ebiten.KeyNumpad4, ebiten.KeyNumpad5, ebiten.KeyNumpad6, ebiten.KeyNumpad7,
		ebiten.KeyNumpad8, ebiten.KeyNumpad9,
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyEnter,
		ebiten.KeyBackspace,
		ebiten.KeyF1, ebiten.KeyF2,
		ebiten.KeyTab, ebiten.KeyNumpadMultiply,
	}
	// Append A-Z dynamically
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		allKeys = append(allKeys, k)
	}

	for _, k := range allKeys {
		isDown := ebiten.IsKeyPressed(k)
		wasDown := g.keyStates[k]
		if isDown != wasDown {
			g.keyStates[k] = isDown
			g.lastKey = fmt.Sprintf("%v (J2ME: %d) pressed: %v", k, gfx.MapEbitenKeyToJ2ME(k), isDown)
			j2meKey := gfx.MapEbitenKeyToJ2ME(k)
			if j2meKey != 0 {
				jvm.DispatchKeyEvent(j2meKey, isDown)
			}
		} else if isDown {
			// KeyRepeated logic
			j2meKey := gfx.MapEbitenKeyToJ2ME(k)
			if j2meKey != 0 && g.tickCount%5 == 0 {
				jvm.DispatchKeyRepeated(j2meKey)
			}
		}
	}

	// Forward Mouse Input
	x, y := ebiten.CursorPosition()
	isMouseBtnDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if isMouseBtnDown && !g.keyStates[ebiten.KeyMax] {
		g.keyStates[ebiten.KeyMax] = true
		jvm.DispatchPointerEvent(x, y, 0) // 0 = pressed
	} else if !isMouseBtnDown && g.keyStates[ebiten.KeyMax] {
		g.keyStates[ebiten.KeyMax] = false
		jvm.DispatchPointerEvent(x, y, 1) // 1 = released
	} else if isMouseBtnDown {
		jvm.DispatchPointerEvent(x, y, 2) // 2 = dragged
	}

	// Force Java GC if VRAM image count is getting too high
	if g.tickCount%300 == 0 {
		if jvm.GetVramCount() > 50 {
			go func() {
				// We do this in a goroutine because JNI calls can be slow
			}()
		}
	}

	return nil
}

// Draw renders the game screen from the MIDP graphics buffer.
func (g *Game) Draw(screen *ebiten.Image) {
	if g.showMenu {
		g.drawMenu(screen)
		return
	}

	if g.graphics != nil && g.graphics.FrontBuffer != nil {
		g.graphics.RLock()
		ebitenImage := ebiten.NewImageFromImage(g.graphics.FrontBuffer)
		g.graphics.RUnlock()
		screen.DrawImage(ebitenImage, nil)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.0f", ebiten.ActualFPS()))

	globalVm := jvm.GetGlobalJVM()
	if globalVm != nil && jvm.IsTextInputActive() {
		msg := fmt.Sprintf("\n\n\n\n\n[Nhap van ban]\n> %s_", jvm.GetInputText())
		ebitenutil.DebugPrint(screen, msg)
	}

	if g.upgradeOverlay != nil {
		if g.upgradeOverlay.IsVisible() {
			g.upgradeOverlay.Draw(screen)
		} else {
			g.upgradeOverlay.DrawHUD(screen)
		}
	}
}

// drawMenu renders the JAR selection menu
func (g *Game) drawMenu(screen *ebiten.Image) {
	// Dark background
	screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})

	// Title
	ebitenutil.DebugPrintAt(screen, "J2ME Emulator", 60, 20)

	// Menu items
	menuY := 80
	arrow := "  "
	if g.menuIndex == 0 {
		arrow = "▶"
	}
	startText := fmt.Sprintf("%s Start Game", arrow)
	if g.selectedJar == "" {
		startText += " (chon JAR truoc)"
	}
	ebitenutil.DebugPrintAt(screen, startText, 40, menuY)

	menuY += 25
	arrow = "  "
	if g.menuIndex == 1 {
		arrow = "▶"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s Chon file JAR", arrow), 40, menuY)

	// Show selected file
	menuY += 50
	ebitenutil.DebugPrintAt(screen, "File da chon:", 40, menuY)
	menuY += 20
	if g.selectedJar != "" {
		fileName := filepath.Base(g.selectedJar)
		ebitenutil.DebugPrintAt(screen, fileName, 40, menuY)
	} else {
		ebitenutil.DebugPrintAt(screen, "(chua chon)", 40, menuY)
	}

	// Instructions
	menuY += 50
	ebitenutil.DebugPrintAt(screen, "↑↓: Di chuyen  |  Enter: Chon", 20, menuY)
}

// launchGame starts the game with the selected JAR
func launchGame(jarPath string) {
	vm := jvm.GetGlobalJVM()
	if vm == nil {
		log.Printf("JVM not initialized")
		return
	}

	if err := vm.InvokeMainWithArgs("com/emulator/Main", []string{jarPath}); err != nil {
		log.Printf("Game finished or failed: %v", err)
	}
}

// Layout returns the logical screen size (240x320 J2ME standard canvas).
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 320
}

func main() {
	runtime.LockOSThread()

	// Parse arguments (prefer KnightOnline_v403.jar as default original game)
	defaultJar := "/home/neuro/Downloads/KnightOnline_v403.jar"
	if _, err := os.Stat(defaultJar); os.IsNotExist(err) {
		defaultJar = "/home/neuro/Downloads/x2.jar"
	}
	for _, arg := range os.Args[1:] {
		arg = strings.TrimSpace(arg)
		if arg != "" && !strings.HasPrefix(arg, "-") {
			defaultJar = arg
		}
	}

	// Initialize graphics
	graphics := midp.NewGraphics(240, 320)
	jvm.SetActiveGraphics(graphics)

	// Initialize embedded HotSpot JVM via CGO/JNI
	vm, err := jvm.InitJVM("./j2me-stubs/bin")
	if err != nil {
		log.Fatalf("Failed to initialize JVM: %v", err)
	}
	fmt.Println("HotSpot JVM initialized successfully via CGO!")

	if err := vm.RegisterNatives(); err != nil {
		log.Fatalf("Failed to register JNI native stubs: %v", err)
	}

	ebiten.SetWindowSize(480, 640)
	ebiten.SetWindowTitle("Knight Online - J2ME Emulator")
	ebiten.SetTPS(60)
	ebiten.SetRunnableOnUnfocused(true)

	// Initialize auto upgrade overlay
	ctrl := autoupgrade.NewController()
	overlay := autoupgrade.NewOverlay(ctrl)

	game := &Game{
		graphics:       graphics,
		keyStates:      make(map[ebiten.Key]bool),
		showMenu:       true,
		selectedJar:    defaultJar,
		menuIndex:      0,
		upgradeOverlay: overlay,
	}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("Ebitengine game loop failed: %v", err)
	}
}
