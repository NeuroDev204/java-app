package autoupgrade

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	totalRows = 9

	rowEquipment       = 0
	rowTargetLevel     = 1
	rowClover3Min      = 2
	rowClover3Max      = 3
	rowClover4Min      = 4
	rowClover4Max      = 5
	rowStopOutOfClover = 6
	rowDelayMs         = 7
	rowStartStopAction = 8
)

// cleanASCII replaces Vietnamese diacritic characters with basic ASCII for crisp Ebiten font rendering.
func cleanASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ả', 'ã', 'ạ', 'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ', 'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ':
			b.WriteRune('a')
		case 'Á', 'À', 'Ả', 'Ã', 'Ạ', 'Ă', 'Ắ', 'Ằ', 'Ẳ', 'Ẵ', 'Ặ', 'Â', 'Ấ', 'Ầ', 'Ẩ', 'Ẫ', 'Ậ':
			b.WriteRune('A')
		case 'đ':
			b.WriteRune('d')
		case 'Đ':
			b.WriteRune('D')
		case 'é', 'è', 'ẻ', 'ẽ', 'ẹ', 'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ':
			b.WriteRune('e')
		case 'É', 'È', 'Ẻ', 'Ẽ', 'Ẹ', 'Ê', 'Ế', 'Ề', 'Ể', 'Ễ', 'Ệ':
			b.WriteRune('E')
		case 'í', 'ì', 'ỉ', 'ĩ', 'ị':
			b.WriteRune('i')
		case 'Í', 'Ì', 'Ỉ', 'Ĩ', 'Ị':
			b.WriteRune('I')
		case 'ó', 'ò', 'ỏ', 'õ', 'ọ', 'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ', 'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ':
			b.WriteRune('o')
		case 'Ó', 'Ò', 'Ỏ', 'Õ', 'Ọ', 'Ô', 'Ố', 'Ồ', 'Ổ', 'Ỗ', 'Ộ', 'Ơ', 'Ớ', 'Ờ', 'Ở', 'Ỡ', 'Ợ':
			b.WriteRune('O')
		case 'ú', 'ù', 'ủ', 'ũ', 'ụ', 'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự':
			b.WriteRune('u')
		case 'Ú', 'Ù', 'Ủ', 'Ũ', 'Ụ', 'Ư', 'Ứ', 'Ừ', 'Ử', 'Ữ', 'Ự':
			b.WriteRune('U')
		case 'ý', 'ỳ', 'ỷ', 'ỹ', 'ỵ':
			b.WriteRune('y')
		case 'Ý', 'Ỳ', 'Ỷ', 'Ỹ', 'Ỵ':
			b.WriteRune('Y')
		default:
			if r < 128 {
				b.WriteRune(r)
			} else {
				b.WriteRune(' ')
			}
		}
	}
	return b.String()
}

// Overlay manages the in-game HUD and interactive settings configuration menu.
type Overlay struct {
	ctrl        ControllerInterface
	mu          sync.RWMutex
	drawMu      sync.Mutex
	visible     bool
	selectedRow int
	prevKeys    map[ebiten.Key]bool
	statusMsg   string
	updateTick  uint64
}

// NewOverlay creates an initialized Overlay instance tied to the given Controller.
func NewOverlay(ctrl ControllerInterface) *Overlay {
	return &Overlay{
		ctrl:        ctrl,
		visible:     false,
		selectedRow: 0,
		prevKeys:    make(map[ebiten.Key]bool),
	}
}

// Update handles periodic state synchronization while running or visible.
func (o *Overlay) Update() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.ctrl == nil {
		return
	}

	o.updateTick++
	if o.updateTick%15 == 0 {
		if o.ctrl.IsRunning() || o.visible {
			_, _ = o.ctrl.GetStatus()
		}
	}
}

// Toggle flips the visibility state of the overlay. When opened, it refreshes the inventory snapshot.
func (o *Overlay) Toggle() {
	o.mu.Lock()
	o.visible = !o.visible
	visible := o.visible
	ctrl := o.ctrl
	o.mu.Unlock()

	if visible && ctrl != nil {
		inv, _ := ctrl.RefreshInventory()
		settings := ctrl.GetSettings()
		o.syncSelectedItem(settings, inv)
		ctrl.UpdateSettings(settings)
		_, _ = ctrl.GetStatus()
	}
}

// IsVisible returns whether the overlay configuration menu is currently displayed.
func (o *Overlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetVisible explicitly sets the visibility state of the overlay.
func (o *Overlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// GetSelectedRow returns the currently focused menu row index (0..8).
func (o *Overlay) GetSelectedRow() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.selectedRow
}

// SetSelectedRow explicitly sets the active menu row cursor.
func (o *Overlay) SetSelectedRow(row int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if row < 0 {
		row = 0
	} else if row >= totalRows {
		row = totalRows - 1
	}
	o.selectedRow = row
}

// HandleInput processes keyboard events from the engine loop to navigate and modify settings.
func (o *Overlay) HandleInput(keyStates map[ebiten.Key]bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Helper to detect rising-edge key press
	isJustPressed := func(k ebiten.Key) bool {
		down := keyStates[k]
		wasDown := o.prevKeys[k]
		return down && !wasDown
	}

	// Toggle overlay via F3
	if isJustPressed(ebiten.KeyF3) {
		o.visible = !o.visible
		if o.visible && o.ctrl != nil {
			inv, _ := o.ctrl.RefreshInventory()
			settings := o.ctrl.GetSettings()
			o.syncSelectedItem(settings, inv)
			o.ctrl.UpdateSettings(settings)
			_, _ = o.ctrl.GetStatus()
		}
		o.updatePrevKeys(keyStates)
		return
	}

	// Close overlay via Escape
	if isJustPressed(ebiten.KeyEscape) {
		if o.visible {
			o.visible = false
		}
		o.updatePrevKeys(keyStates)
		return
	}

	if !o.visible {
		o.updatePrevKeys(keyStates)
		return
	}

	// Vertical Navigation
	if isJustPressed(ebiten.KeyDown) {
		o.selectedRow = (o.selectedRow + 1) % totalRows
	}
	if isJustPressed(ebiten.KeyUp) {
		o.selectedRow = (o.selectedRow - 1 + totalRows) % totalRows
	}

	if o.ctrl != nil {
		settings := o.ctrl.GetSettings()
		inv := o.ctrl.GetCachedInventory()
		itemCount := 0
		if inv != nil {
			itemCount = len(inv.Items)
		}

		leftPressed := isJustPressed(ebiten.KeyLeft)
		rightPressed := isJustPressed(ebiten.KeyRight)
		enterPressed := isJustPressed(ebiten.KeyEnter)

		switch o.selectedRow {
		case rowEquipment:
			if itemCount > 0 {
				if rightPressed {
					settings.SelectedItemIndex = (settings.SelectedItemIndex + 1) % itemCount
					o.onEquipmentChanged(settings, inv)
				} else if leftPressed {
					settings.SelectedItemIndex = (settings.SelectedItemIndex - 1 + itemCount) % itemCount
					o.onEquipmentChanged(settings, inv)
				}
			}

		case rowTargetLevel:
			if rightPressed && settings.TargetLevel < 15 {
				settings.TargetLevel++
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.TargetLevel > 1 {
				settings.TargetLevel--
				o.ctrl.UpdateSettings(settings)
			}

		case rowClover3Min:
			if rightPressed && settings.Clover3Min < 15 {
				settings.Clover3Min++
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.Clover3Min > 0 {
				settings.Clover3Min--
				o.ctrl.UpdateSettings(settings)
			}

		case rowClover3Max:
			if rightPressed && settings.Clover3Max < 15 {
				settings.Clover3Max++
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.Clover3Max > 0 {
				settings.Clover3Max--
				o.ctrl.UpdateSettings(settings)
			}

		case rowClover4Min:
			if rightPressed && settings.Clover4Min < 15 {
				settings.Clover4Min++
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.Clover4Min > 0 {
				settings.Clover4Min--
				o.ctrl.UpdateSettings(settings)
			}

		case rowClover4Max:
			if rightPressed && settings.Clover4Max < 15 {
				settings.Clover4Max++
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.Clover4Max > 0 {
				settings.Clover4Max--
				o.ctrl.UpdateSettings(settings)
			}

		case rowStopOutOfClover:
			if leftPressed || rightPressed || enterPressed {
				settings.StopIfOutOfClover = !settings.StopIfOutOfClover
				o.ctrl.UpdateSettings(settings)
			}

		case rowDelayMs:
			if rightPressed && settings.DelayMs <= 4950 {
				settings.DelayMs += 50
				o.ctrl.UpdateSettings(settings)
			} else if leftPressed && settings.DelayMs >= 150 {
				settings.DelayMs -= 50
				o.ctrl.UpdateSettings(settings)
			}

		case rowStartStopAction:
			if enterPressed {
				if o.ctrl.IsRunning() {
					_ = o.ctrl.StopUpgrade()
					o.statusMsg = "Da yeu cau dung..."
				} else {
					o.syncSelectedItem(settings, inv)
					err := o.ctrl.StartUpgrade(settings)
					if err != nil {
						o.statusMsg = fmt.Sprintf("Loi: %v", err)
					} else {
						o.statusMsg = "Da bat dau dap..."
					}
				}
			}
		}
	}

	o.updatePrevKeys(keyStates)
}

func (o *Overlay) syncSelectedItem(settings *UpgradeSettings, inv *InventoryData) {
	if inv != nil && len(inv.Items) > 0 {
		if settings.SelectedItemIndex < 0 || settings.SelectedItemIndex >= len(inv.Items) {
			settings.SelectedItemIndex = 0
		}
		item := inv.Items[settings.SelectedItemIndex]
		if item != nil {
			settings.ItemID = item.ID
			settings.ItemType = item.Type
			settings.ItemName = item.Name
			if settings.TargetLevel <= item.Level {
				settings.TargetLevel = item.Level + 1
				if settings.TargetLevel > 15 {
					settings.TargetLevel = 15
				}
			}
		}
	}
}

func (o *Overlay) onEquipmentChanged(settings *UpgradeSettings, inv *InventoryData) {
	o.syncSelectedItem(settings, inv)
	// When switching equipment, reset clover config levels back to default off (+0..+0)
	// and clear any clover attached in the game slot.
	settings.Clover3Min = 0
	settings.Clover3Max = 0
	settings.Clover4Min = 0
	settings.Clover4Max = 0
	if o.ctrl != nil {
		_ = o.ctrl.ResetClover()
		o.ctrl.UpdateSettings(settings)
	}
}

func (o *Overlay) updatePrevKeys(keyStates map[ebiten.Key]bool) {
	for k, v := range keyStates {
		o.prevKeys[k] = v
	}
	for k := range o.prevKeys {
		if !keyStates[k] {
			delete(o.prevKeys, k)
		}
	}
}

var (
	colorTitle     = color.RGBA{R: 255, G: 215, B: 0, A: 255}   // Gold
	colorDivider   = color.RGBA{R: 70, G: 130, B: 210, A: 255}  // Slate Blue
	colorCo3       = color.RGBA{R: 110, G: 255, B: 120, A: 255} // Lime Green
	colorCo4       = color.RGBA{R: 60, G: 240, B: 220, A: 255}  // Cyan Emerald
	colorBH        = color.RGBA{R: 255, G: 140, B: 255, A: 255} // Light Magenta
	colorGold      = color.RGBA{R: 255, G: 220, B: 60, A: 255}  // Bright Gold
	colorRowActive = color.RGBA{R: 255, G: 255, B: 100, A: 255} // Bright Yellow Highlight
	colorRowNormal = color.RGBA{R: 210, G: 230, B: 250, A: 255} // Soft White/Blue
	colorSuccess   = color.RGBA{R: 80, G: 255, B: 120, A: 255}  // Bright Green
	colorFailure   = color.RGBA{R: 255, G: 110, B: 100, A: 255} // Coral Red
	colorWarning   = color.RGBA{R: 255, G: 200, B: 60, A: 255}  // Amber
	colorMuted     = color.RGBA{R: 140, G: 175, B: 215, A: 255} // Muted Blue-Gray
	colorStartBtn  = color.RGBA{R: 70, G: 255, B: 130, A: 255}  // Neon Green
	colorStopBtn   = color.RGBA{R: 255, G: 80, B: 80, A: 255}   // Crimson Red
)

// drawColoredText renders clean ASCII text onto dst at (x, y) with color scaling.
func drawColoredText(dst *ebiten.Image, str string, x, y int, clr color.Color) {
	if str == "" || dst == nil {
		return
	}
	clean := cleanASCII(str)
	w := len(clean)*7 + 8
	h := 16
	if w < 16 {
		w = 16
	}
	txtImg := ebiten.NewImage(w, h)
	ebitenutil.DebugPrintAt(txtImg, clean, 0, 0)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	if clr != nil {
		op.ColorScale.ScaleWithColor(clr)
	}
	dst.DrawImage(txtImg, op)
}

// Draw renders the full interactive settings overlay onto the game canvas.
func (o *Overlay) Draw(screen *ebiten.Image) {
	if screen == nil || !o.IsVisible() {
		return
	}

	o.drawMu.Lock()
	defer o.drawMu.Unlock()

	o.mu.RLock()
	selectedRow := o.selectedRow
	ctrl := o.ctrl
	o.mu.RUnlock()

	// 1. Semi-transparent background box & borders (240x320 bounds)
	vector.FillRect(screen, 4, 4, 232, 312, color.RGBA{R: 10, G: 14, B: 28, A: 245}, false)
	vector.StrokeRect(screen, 4, 4, 232, 312, 1, color.RGBA{R: 60, G: 130, B: 230, A: 255}, false)

	// 2. Header
	drawColoredText(screen, "=== AUTO CUONG HOA (F3) ===", 18, 10, colorTitle)
	drawColoredText(screen, "------------------------------", 10, 24, colorDivider)

	// 3. Materials & Inventory Status
	var inv *InventoryData
	var settings *UpgradeSettings
	var status *Status
	isRunning := false

	if ctrl != nil {
		inv = ctrl.GetCachedInventory()
		settings = ctrl.GetSettings()
		status = ctrl.GetCachedStatus()
		isRunning = ctrl.IsRunning()
	}
	if settings == nil {
		settings = DefaultSettings()
	}

	gold := 0
	c3 := 0
	c4 := 0
	bh := 0
	var items []*Item

	if inv != nil {
		gold = inv.Gold
		c3 = inv.Clover3Count
		c4 = inv.Clover4Count
		bh = inv.InsuranceCount
		items = inv.Items
	}

	// Colored material header
	drawColoredText(screen, fmt.Sprintf("Co3: %d", c3), 10, 36, colorCo3)
	drawColoredText(screen, fmt.Sprintf("Co4: %d", c4), 78, 36, colorCo4)
	drawColoredText(screen, fmt.Sprintf("BH: %d", bh), 146, 36, colorBH)
	drawColoredText(screen, fmt.Sprintf("Vang: %d xu", gold), 10, 48, colorGold)
	drawColoredText(screen, "------------------------------", 10, 60, colorDivider)

	// 4. Configuration Rows (Row 0..8)
	cursorFor := func(row int) string {
		if row == selectedRow {
			return "> "
		}
		return "  "
	}

	colorForRow := func(row int) color.Color {
		if row == selectedRow {
			return colorRowActive
		}
		return colorRowNormal
	}

	drawRowBackground := func(row int, y int) {
		if row == selectedRow {
			vector.FillRect(screen, 6, float32(y-2), 228, 16, color.RGBA{R: 35, G: 65, B: 115, A: 190}, false)
			vector.StrokeRect(screen, 6, float32(y-2), 228, 16, 1, color.RGBA{R: 80, G: 150, B: 240, A: 220}, false)
		}
	}

	// Row 0: Equipment Selector
	drawRowBackground(rowEquipment, 74)
	itemLabel := "(Khong co trang bi)"
	if len(items) > 0 {
		idx := settings.SelectedItemIndex
		if idx < 0 || idx >= len(items) {
			idx = 0
		}
		it := items[idx]
		if it != nil {
			cleanName := cleanASCII(it.Name)
			itemLabel = fmt.Sprintf("[%d/%d] +%d %s", idx+1, len(items), it.Level, cleanName)
		}
	}
	drawColoredText(screen, fmt.Sprintf("%sTrang bi : < %s >", cursorFor(rowEquipment), itemLabel), 10, 74, colorForRow(rowEquipment))

	// Row 1: Target Level
	drawRowBackground(rowTargetLevel, 92)
	drawColoredText(screen, fmt.Sprintf("%sMuc tieu : < +%d >", cursorFor(rowTargetLevel), settings.TargetLevel), 10, 92, colorForRow(rowTargetLevel))

	// Row 2: Clover 3 Min
	drawRowBackground(rowClover3Min, 110)
	drawColoredText(screen, fmt.Sprintf("%sCo 3 Tu  : < +%d >", cursorFor(rowClover3Min), settings.Clover3Min), 10, 110, colorForRow(rowClover3Min))

	// Row 3: Clover 3 Max
	drawRowBackground(rowClover3Max, 128)
	drawColoredText(screen, fmt.Sprintf("%sCo 3 Den : < +%d >", cursorFor(rowClover3Max), settings.Clover3Max), 10, 128, colorForRow(rowClover3Max))

	// Row 4: Clover 4 Min
	drawRowBackground(rowClover4Min, 146)
	drawColoredText(screen, fmt.Sprintf("%sCo 4 Tu  : < +%d >", cursorFor(rowClover4Min), settings.Clover4Min), 10, 146, colorForRow(rowClover4Min))

	// Row 5: Clover 4 Max
	drawRowBackground(rowClover4Max, 164)
	drawColoredText(screen, fmt.Sprintf("%sCo 4 Den : < +%d >", cursorFor(rowClover4Max), settings.Clover4Max), 10, 164, colorForRow(rowClover4Max))

	// Row 6: Stop If Out Of Clover
	drawRowBackground(rowStopOutOfClover, 182)
	stopText := "TAT"
	if settings.StopIfOutOfClover {
		stopText = "BAT"
	}
	drawColoredText(screen, fmt.Sprintf("%sHet co   : < %s >", cursorFor(rowStopOutOfClover), stopText), 10, 182, colorForRow(rowStopOutOfClover))

	// Row 7: Delay ms
	drawRowBackground(rowDelayMs, 200)
	drawColoredText(screen, fmt.Sprintf("%sToc do   : < %dms >", cursorFor(rowDelayMs), settings.DelayMs), 10, 200, colorForRow(rowDelayMs))

	// Row 8: Action Button
	drawRowBackground(rowStartStopAction, 218)
	actionText := "[ BAT DAU DAP (Enter) ]"
	btnColor := colorStartBtn
	if isRunning {
		actionText = "[ TAM DUNG (Enter) ]"
		btnColor = colorStopBtn
	}
	drawColoredText(screen, fmt.Sprintf("%s%s", cursorFor(rowStartStopAction), actionText), 10, 218, btnColor)

	// 5. Status / Metric Footer
	drawColoredText(screen, "------------------------------", 10, 236, colorDivider)

	currentMsg := "San sang"
	msgColor := colorMuted
	attempts := 0
	successes := 0
	failures := 0

	if status != nil {
		if status.Message != "" {
			currentMsg = cleanASCII(status.Message)
		}
		attempts = status.Attempts
		successes = status.Successes
		failures = status.Failures

		lower := strings.ToLower(currentMsg)
		if strings.Contains(lower, "thanh cong") || strings.Contains(lower, "chuc mung") {
			msgColor = colorSuccess
		} else if strings.Contains(lower, "that bai") || strings.Contains(lower, "tut") || strings.Contains(lower, "loi") {
			msgColor = colorFailure
		} else if strings.Contains(lower, "dang dap") || strings.Contains(lower, "bat dau") {
			msgColor = colorWarning
		} else {
			msgColor = colorRowNormal
		}
	}

	drawColoredText(screen, fmt.Sprintf("TT: %s", currentMsg), 10, 248, msgColor)
	drawColoredText(screen, fmt.Sprintf("KQ: %d lan", attempts), 10, 262, colorRowNormal)
	drawColoredText(screen, fmt.Sprintf("(TC: %d)", successes), 105, 262, colorSuccess)
	drawColoredText(screen, fmt.Sprintf("(TB: %d)", failures), 165, 262, colorFailure)

	drawColoredText(screen, "[F3/Esc]: Dong | [Enter]: Chon", 10, 280, colorMuted)
	drawColoredText(screen, "[Up/Down]: Chon | [< >]: Chinh", 10, 295, colorMuted)
}

// DrawHUD renders a compact top-screen status banner when auto upgrade is running in the background.
func (o *Overlay) DrawHUD(screen *ebiten.Image) {
	if screen == nil || o.IsVisible() || o.ctrl == nil || !o.ctrl.IsRunning() {
		return
	}

	o.drawMu.Lock()
	defer o.drawMu.Unlock()

	status := o.ctrl.GetCachedStatus()
	if status == nil {
		return
	}

	// Compact top-screen bar with modern dark gradient feel
	vector.FillRect(screen, 0, 0, 240, 25, color.RGBA{R: 12, G: 16, B: 30, A: 235}, false)
	vector.StrokeRect(screen, 0, 0, 240, 25, 1, color.RGBA{R: 60, G: 140, B: 240, A: 210}, false)

	// Line 1: Auto tag and live counters with colors
	drawColoredText(screen, fmt.Sprintf("[AUTO +%d->+%d]", status.CurrentLevel, status.TargetLevel), 4, 1, colorTitle)
	drawColoredText(screen, fmt.Sprintf("TC:%d", status.Successes), 130, 1, colorSuccess)
	drawColoredText(screen, fmt.Sprintf("TB:%d", status.Failures), 185, 1, colorFailure)

	// Line 2: Status message with adaptive color
	msg := cleanASCII(status.Message)
	if len(msg) > 22 {
		msg = msg[:22] + ".."
	}
	lower := strings.ToLower(msg)
	msgColor := colorWarning
	if strings.Contains(lower, "thanh cong") || strings.Contains(lower, "chuc mung") {
		msgColor = colorSuccess
	} else if strings.Contains(lower, "that bai") || strings.Contains(lower, "tut") || strings.Contains(lower, "loi") {
		msgColor = colorFailure
	} else if strings.Contains(lower, "dang dap") {
		msgColor = colorGold
	}

	drawColoredText(screen, msg, 4, 12, msgColor)
	drawColoredText(screen, "(F3)", 205, 12, colorMuted)
}
