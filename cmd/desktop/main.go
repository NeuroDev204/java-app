package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/sqweek/dialog"

	"java-app/pkg/autoupgrade"
	"java-app/pkg/gamebridge"
	"java-app/pkg/gfx"
)

// GameBridgeAutoUpgradeBridge adapts gamebridge.Bridge to autoupgrade.Bridge.
type GameBridgeAutoUpgradeBridge struct {
	bridge *gamebridge.Bridge
}

func (b *GameBridgeAutoUpgradeBridge) GetInventoryJson() (string, error) {
	if b.bridge == nil {
		return `{"gold":0,"clover3Count":0,"clover4Count":0,"insuranceCount":0,"items":[]}`, nil
	}
	return b.bridge.GetInventoryJSON(), nil
}

func (b *GameBridgeAutoUpgradeBridge) StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error {
	if b.bridge == nil {
		return fmt.Errorf("bridge not initialized")
	}
	cfg := map[string]interface{}{
		"item_id":               itemId,
		"item_type":             itemType,
		"target_level":          targetLevel,
		"clover3_min":           clover3Min,
		"clover3_max":           clover3Max,
		"clover4_min":           clover4Min,
		"clover4_max":           clover4Max,
		"stop_if_out_of_clover": stopOutOfClover,
		"delay_ms":              delayMs,
	}
	bytes, _ := json.Marshal(cfg)
	res := b.bridge.StartUpgrade(string(bytes))
	if res != "" {
		return fmt.Errorf("%s", res)
	}
	return nil
}

func (b *GameBridgeAutoUpgradeBridge) StopUpgrade() error {
	if b.bridge != nil {
		b.bridge.StopUpgrade()
	}
	return nil
}

func (b *GameBridgeAutoUpgradeBridge) GetUpgradeStatusJson() (string, error) {
	if b.bridge == nil {
		return `{"running":false,"connected":false,"in_game":false,"currentLevel":0,"targetLevel":0,"attempts":0,"successes":0,"failures":0,"message":"","stopped":false}`, nil
	}
	raw := b.bridge.GetUpgradeStatus()
	var rawSt struct {
		Running   bool     `json:"running"`
		Connected bool     `json:"connected"`
		InGame    bool     `json:"in_game"`
		Attempts  int      `json:"attempts"`
		Success   int      `json:"success"`
		Fail      int      `json:"fail"`
		ItemLevel int8     `json:"item_level"`
		Logs      []string `json:"logs"`
	}
	if err := json.Unmarshal([]byte(raw), &rawSt); err == nil {
		msg := ""
		if len(rawSt.Logs) > 0 {
			msg = rawSt.Logs[len(rawSt.Logs)-1]
		}
		mapped, _ := json.Marshal(map[string]interface{}{
			"running":      rawSt.Running,
			"connected":    rawSt.Connected,
			"in_game":      rawSt.InGame,
			"currentLevel": rawSt.ItemLevel,
			"attempts":     rawSt.Attempts,
			"successes":    rawSt.Success,
			"failures":     rawSt.Fail,
			"message":      msg,
			"stopped":      !rawSt.Running,
		})
		return string(mapped), nil
	}
	return raw, nil
}

func (b *GameBridgeAutoUpgradeBridge) ResetClover() error {
	return nil
}

// DesktopGame implements ebiten.Game using the pure-Go gamebridge engine and F3 AutoUpgrade overlay.
type DesktopGame struct {
	bridge         *gamebridge.Bridge
	upgradeCtrl    *autoupgrade.Controller
	upgradeOverlay *autoupgrade.Overlay
	keyStates      map[ebiten.Key]bool
	mouseState     bool
	showMenu       bool
	selectedJar    string
	menuIndex      int
	lastError      string
	textInputBuf   string
	tickCount      uint64
}

// NewDesktopGame initializes a desktop game instance.
func NewDesktopGame(initialJar string) *DesktopGame {
	b := gamebridge.NewBridge()
	adapter := &GameBridgeAutoUpgradeBridge{bridge: b}
	ctrl := autoupgrade.NewControllerWithBridge(adapter)
	overlay := autoupgrade.NewOverlay(ctrl)

	g := &DesktopGame{
		bridge:         b,
		upgradeCtrl:    ctrl,
		upgradeOverlay: overlay,
		keyStates:      make(map[ebiten.Key]bool),
		selectedJar:    initialJar,
		showMenu:       initialJar == "",
	}
	return g
}

func (g *DesktopGame) startWithJarFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read jar file: %w", err)
	}
	return g.startWithJarBytes(data)
}

func (g *DesktopGame) startWithJarBytes(data []byte) error {
	// Persist RMS data locally
	gamebridge.SetDataDir(".")

	if g.bridge == nil {
		g.bridge = gamebridge.NewBridge()
	}

	msg := g.bridge.LoadJAR(data)
	if msg != "" {
		return fmt.Errorf("load JAR failed: %s", msg)
	}

	adapter := &GameBridgeAutoUpgradeBridge{bridge: g.bridge}
	g.upgradeCtrl = autoupgrade.NewControllerWithBridge(adapter)
	g.upgradeOverlay = autoupgrade.NewOverlay(g.upgradeCtrl)

	g.showMenu = false
	return nil
}

// Update proceeds the game state at 60 FPS.
func (g *DesktopGame) Update() error {
	g.tickCount++

	// Menu state
	if g.showMenu {
		if ebiten.IsKeyPressed(ebiten.KeyDown) && !g.keyStates[ebiten.KeyDown] {
			g.keyStates[ebiten.KeyDown] = true
			g.menuIndex = (g.menuIndex + 1) % 2
		}
		if ebiten.IsKeyPressed(ebiten.KeyUp) && !g.keyStates[ebiten.KeyUp] {
			g.keyStates[ebiten.KeyUp] = true
			g.menuIndex = (g.menuIndex + 1) % 2
		}

		if ebiten.IsKeyPressed(ebiten.KeyEnter) && !g.keyStates[ebiten.KeyEnter] {
			g.keyStates[ebiten.KeyEnter] = true
			if g.menuIndex == 0 {
				if g.selectedJar != "" {
					if err := g.startWithJarFile(g.selectedJar); err != nil {
						g.lastError = err.Error()
					}
				}
			} else {
				filePath, err := dialog.File().Filter("JAR files (*.jar)", "jar").Load()
				if err == nil && filePath != "" {
					g.selectedJar = filePath
					g.lastError = ""
				}
			}
		}

		g.keyStates[ebiten.KeyDown] = ebiten.IsKeyPressed(ebiten.KeyDown)
		g.keyStates[ebiten.KeyUp] = ebiten.IsKeyPressed(ebiten.KeyUp)
		g.keyStates[ebiten.KeyEnter] = ebiten.IsKeyPressed(ebiten.KeyEnter)
		return nil
	}

	if g.bridge == nil || !g.bridge.IsStarted() {
		return nil
	}

	// In-game F3 Auto Upgrade Overlay navigation & toggle
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

	// Check for active J2ME TextBox text input modal
	textInfoJSON := g.bridge.GetTextInputInfoJSON()
	var textInfo struct {
		Active     bool   `json:"active"`
		Title      string `json:"title"`
		Text       string `json:"text"`
		MaxSize    int    `json:"max_size"`
		IsPassword bool   `json:"is_password"`
	}
	if err := json.Unmarshal([]byte(textInfoJSON), &textInfo); err == nil && textInfo.Active {
		// Capture typed keyboard characters
		chars := ebiten.AppendInputChars(nil)
		if len(chars) > 0 {
			g.textInputBuf += string(chars)
		}
		if ebiten.IsKeyPressed(ebiten.KeyBackspace) && !g.keyStates[ebiten.KeyBackspace] {
			if len(g.textInputBuf) > 0 {
				runes := []rune(g.textInputBuf)
				g.textInputBuf = string(runes[:len(runes)-1])
			}
		}
		if ebiten.IsKeyPressed(ebiten.KeyEnter) && !g.keyStates[ebiten.KeyEnter] {
			g.bridge.SubmitTextInput(g.textInputBuf)
			g.textInputBuf = ""
		}
		if ebiten.IsKeyPressed(ebiten.KeyEscape) && !g.keyStates[ebiten.KeyEscape] {
			g.bridge.CancelTextInput()
			g.textInputBuf = ""
		}
		g.keyStates[ebiten.KeyBackspace] = ebiten.IsKeyPressed(ebiten.KeyBackspace)
		g.keyStates[ebiten.KeyEnter] = ebiten.IsKeyPressed(ebiten.KeyEnter)
		g.keyStates[ebiten.KeyEscape] = ebiten.IsKeyPressed(ebiten.KeyEscape)
		return nil
	}

	// Handle Mouse / Touch
	cx, cy := ebiten.CursorPosition()
	isMouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if isMouseDown && !g.mouseState {
		g.mouseState = true
		g.bridge.SendTouchDown(cx, cy)
	} else if isMouseDown && g.mouseState {
		g.bridge.SendTouchDrag(cx, cy)
	} else if !isMouseDown && g.mouseState {
		g.mouseState = false
		g.bridge.SendTouchUp(cx, cy)
	}

	// Handle Keypad & Softkeys
	keysToCheck := []ebiten.Key{
		ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7,
		ebiten.KeyDigit8, ebiten.KeyDigit9,
		ebiten.KeyNumpad0, ebiten.KeyNumpad1, ebiten.KeyNumpad2, ebiten.KeyNumpad3,
		ebiten.KeyNumpad4, ebiten.KeyNumpad5, ebiten.KeyNumpad6, ebiten.KeyNumpad7,
		ebiten.KeyNumpad8, ebiten.KeyNumpad9,
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyEnter,
		ebiten.KeyBackspace, ebiten.KeySpace,
		ebiten.KeyF1, ebiten.KeyF2,
		ebiten.KeyTab, ebiten.KeyNumpadMultiply,
	}
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		keysToCheck = append(keysToCheck, k)
	}

	for _, k := range keysToCheck {
		isDown := ebiten.IsKeyPressed(k)
		wasDown := g.keyStates[k]
		if isDown != wasDown {
			g.keyStates[k] = isDown
			jKey := gfx.MapEbitenKeyToJ2ME(k)
			if jKey != 0 {
				if isDown {
					g.bridge.SendKeyDown(jKey)
				} else {
					g.bridge.SendKeyUp(jKey)
				}
			}
		}
	}

	return nil
}

// Draw renders the game, menu, text input modal, and F3 auto-upgrade overlay to screen.
func (g *DesktopGame) Draw(screen *ebiten.Image) {
	if g.showMenu {
		g.drawMenu(screen)
		return
	}

	if g.bridge != nil {
		buf := g.bridge.GetFrameBuffer()
		if len(buf) == 240*320*4 {
			screen.WritePixels(buf)
		}
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.0f", ebiten.ActualFPS()))

	// Render Text Input Modal HUD if active
	if g.bridge != nil {
		var textInfo struct {
			Active     bool   `json:"active"`
			Title      string `json:"title"`
			IsPassword bool   `json:"is_password"`
		}
		if err := json.Unmarshal([]byte(g.bridge.GetTextInputInfoJSON()), &textInfo); err == nil && textInfo.Active {
			display := g.textInputBuf
			if textInfo.IsPassword {
				display = strings.Repeat("*", len(g.textInputBuf))
			}
			msg := fmt.Sprintf("\n\n\n\n\n[Nhap: %s]\n> %s_\n(Enter: OK | Esc: Cancel)", textInfo.Title, display)
			ebitenutil.DebugPrint(screen, msg)
		}
	}

	// Render F3 Auto Upgrade HUD or full Overlay
	if g.upgradeOverlay != nil {
		if g.upgradeOverlay.IsVisible() {
			g.upgradeOverlay.Draw(screen)
		} else {
			g.upgradeOverlay.DrawHUD(screen)
		}
	}
}

// drawMenu renders the simple start/file selection menu.
func (g *DesktopGame) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})
	ebitenutil.DebugPrintAt(screen, "J2ME Emulator (Pure-Go)", 50, 20)

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
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s Chon file JAR...", arrow), 40, menuY)

	menuY += 50
	ebitenutil.DebugPrintAt(screen, "File da chon:", 40, menuY)
	menuY += 20
	if g.selectedJar != "" {
		fileName := filepath.Base(g.selectedJar)
		ebitenutil.DebugPrintAt(screen, fileName, 40, menuY)
	} else {
		ebitenutil.DebugPrintAt(screen, "(chua chon)", 40, menuY)
	}

	if g.lastError != "" {
		menuY += 40
		ebitenutil.DebugPrintAt(screen, "Loi: "+g.lastError, 20, menuY)
	}

	menuY = 280
	ebitenutil.DebugPrintAt(screen, "↑↓: Di chuyen  |  Enter: Chon", 25, menuY)
}

// Layout maintains the fixed 240x320 J2ME canvas size, automatically scaled by Ebiten.
func (g *DesktopGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 320
}

func main() {
	var targetJar string

	// Check arguments or local fallback files
	candidates := []string{
		"KnightOnline_402.jar",
		"mobile/assets/KnightOnline_402.jar",
		"x2.jar",
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			targetJar = cand
			break
		}
	}

	for _, arg := range os.Args[1:] {
		arg = strings.TrimSpace(arg)
		if arg != "" && !strings.HasPrefix(arg, "-") && strings.HasSuffix(strings.ToLower(arg), ".jar") {
			targetJar = arg
			break
		}
	}

	game := NewDesktopGame(targetJar)
	if targetJar != "" {
		// Automatically start game if jar is specified
		if err := game.startWithJarFile(targetJar); err != nil {
			log.Printf("Auto-start with %s failed: %v", targetJar, err)
			game.showMenu = true
			game.lastError = err.Error()
		}
	}

	ebiten.SetWindowSize(480, 640)
	ebiten.SetWindowTitle("Knight Online - J2ME Emulator (Pure-Go)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	ebiten.SetRunnableOnUnfocused(true)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("Ebitengine loop terminated: %v", err)
	}
}
