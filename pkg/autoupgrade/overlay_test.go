package autoupgrade

import (
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestOverlay_New(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)

	if overlay == nil {
		t.Fatal("expected non-nil Overlay")
	}
	if overlay.IsVisible() {
		t.Errorf("expected overlay to start hidden, got visible=true")
	}
	if overlay.GetSelectedRow() != 0 {
		t.Errorf("expected initial selectedRow=0, got %d", overlay.GetSelectedRow())
	}
}

func TestOverlay_Toggle(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)

	if overlay.IsVisible() {
		t.Errorf("expected initially hidden")
	}

	// Toggle ON -> should become visible and refresh inventory
	overlay.Toggle()
	if !overlay.IsVisible() {
		t.Errorf("expected overlay to become visible")
	}

	inv := ctrl.GetCachedInventory()
	if inv == nil || len(inv.Items) != 2 {
		t.Errorf("expected inventory to be refreshed on toggle, got %+v", inv)
	}

	// Toggle OFF -> should become hidden
	overlay.Toggle()
	if overlay.IsVisible() {
		t.Errorf("expected overlay to become hidden")
	}
}

func TestOverlay_Navigation_UpDown(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)

	// Step down through all 9 rows: 0 -> 1 -> 2 -> ... -> 8 -> 0
	for i := 1; i <= 8; i++ {
		keys := map[ebiten.Key]bool{ebiten.KeyDown: true}
		overlay.HandleInput(keys)
		overlay.HandleInput(map[ebiten.Key]bool{}) // release key
		if overlay.GetSelectedRow() != i {
			t.Fatalf("step %d: expected selectedRow=%d, got %d", i, i, overlay.GetSelectedRow())
		}
	}

	// Down again -> wrap to 0
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyDown: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if overlay.GetSelectedRow() != 0 {
		t.Fatalf("expected wrap around to selectedRow=0, got %d", overlay.GetSelectedRow())
	}

	// Step up from 0 -> wrap to 8
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyUp: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if overlay.GetSelectedRow() != 8 {
		t.Fatalf("expected wrap around to selectedRow=8, got %d", overlay.GetSelectedRow())
	}

	// Step up again -> 7
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyUp: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if overlay.GetSelectedRow() != 7 {
		t.Fatalf("expected selectedRow=7, got %d", overlay.GetSelectedRow())
	}
}

func TestOverlay_HandleInput_EquipmentCycle(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	_, _ = ctrl.RefreshInventory() // loads 2 items
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)
	overlay.SetSelectedRow(0) // Row 0 = Equipment

	settings := ctrl.GetSettings()
	if settings.SelectedItemIndex != 0 {
		t.Fatalf("expected default SelectedItemIndex=0, got %d", settings.SelectedItemIndex)
	}

	// Press Right -> cycle to item 1
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	settings = ctrl.GetSettings()
	if settings.SelectedItemIndex != 1 {
		t.Errorf("expected SelectedItemIndex=1, got %d", settings.SelectedItemIndex)
	}
	if settings.Clover3Min != 0 || settings.Clover3Max != 0 || settings.Clover4Min != 0 || settings.Clover4Max != 0 {
		t.Errorf("expected clover settings reset to 0, got c3=(%d..%d), c4=(%d..%d)",
			settings.Clover3Min, settings.Clover3Max, settings.Clover4Min, settings.Clover4Max)
	}
	if !mockBridge.resetCloverCalled {
		t.Errorf("expected mockBridge.ResetClover to have been called on equipment switch")
	}

	// Press Right again -> cycle wrap to item 0
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	settings = ctrl.GetSettings()
	if settings.SelectedItemIndex != 0 {
		t.Errorf("expected SelectedItemIndex=0 after wrap, got %d", settings.SelectedItemIndex)
	}

	// Press Left -> cycle wrap to item 1
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	settings = ctrl.GetSettings()
	if settings.SelectedItemIndex != 1 {
		t.Errorf("expected SelectedItemIndex=1 after left wrap, got %d", settings.SelectedItemIndex)
	}
}

func TestOverlay_HandleInput_TargetLevel(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)
	overlay.SetSelectedRow(1) // Row 1 = Target Level (default 8)

	// Press Right -> 9
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().TargetLevel != 9 {
		t.Errorf("expected TargetLevel=9, got %d", ctrl.GetSettings().TargetLevel)
	}

	// Press Left twice -> 7
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().TargetLevel != 7 {
		t.Errorf("expected TargetLevel=7, got %d", ctrl.GetSettings().TargetLevel)
	}

	// Test upper bound clamping (max 15)
	for i := 0; i < 20; i++ {
		overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
		overlay.HandleInput(map[ebiten.Key]bool{})
	}
	if ctrl.GetSettings().TargetLevel != 15 {
		t.Errorf("expected TargetLevel clamped to 15, got %d", ctrl.GetSettings().TargetLevel)
	}

	// Test lower bound clamping (min 1)
	for i := 0; i < 25; i++ {
		overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
		overlay.HandleInput(map[ebiten.Key]bool{})
	}
	if ctrl.GetSettings().TargetLevel != 1 {
		t.Errorf("expected TargetLevel clamped to 1, got %d", ctrl.GetSettings().TargetLevel)
	}
}

func TestOverlay_HandleInput_CloverBounds(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)

	// Row 2: Clover 3 Min (default 4)
	overlay.SetSelectedRow(2)
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().Clover3Min != 5 {
		t.Errorf("expected Clover3Min=5, got %d", ctrl.GetSettings().Clover3Min)
	}

	// Row 3: Clover 3 Max (default 7)
	overlay.SetSelectedRow(3)
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().Clover3Max != 6 {
		t.Errorf("expected Clover3Max=6, got %d", ctrl.GetSettings().Clover3Max)
	}

	// Row 4: Clover 4 Min (default 8)
	overlay.SetSelectedRow(4)
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().Clover4Min != 9 {
		t.Errorf("expected Clover4Min=9, got %d", ctrl.GetSettings().Clover4Min)
	}

	// Row 5: Clover 4 Max (default 15)
	overlay.SetSelectedRow(5)
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().Clover4Max != 14 {
		t.Errorf("expected Clover4Max=14, got %d", ctrl.GetSettings().Clover4Max)
	}
}

func TestOverlay_HandleInput_StopIfOutOfClover(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)
	overlay.SetSelectedRow(6) // Row 6 = StopIfOutOfClover (default true)

	// Toggle with Right -> false
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().StopIfOutOfClover != false {
		t.Errorf("expected StopIfOutOfClover=false, got %v", ctrl.GetSettings().StopIfOutOfClover)
	}

	// Toggle with Left -> true
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().StopIfOutOfClover != true {
		t.Errorf("expected StopIfOutOfClover=true, got %v", ctrl.GetSettings().StopIfOutOfClover)
	}

	// Toggle with Enter -> false
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyEnter: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().StopIfOutOfClover != false {
		t.Errorf("expected StopIfOutOfClover=false after Enter, got %v", ctrl.GetSettings().StopIfOutOfClover)
	}
}

func TestOverlay_HandleInput_DelayMs(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)
	overlay.SetSelectedRow(7) // Row 7 = DelayMs (default 500)

	// Press Right -> 550ms
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyRight: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().DelayMs != 550 {
		t.Errorf("expected DelayMs=550, got %d", ctrl.GetSettings().DelayMs)
	}

	// Press Left twice -> 450ms
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyLeft: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if ctrl.GetSettings().DelayMs != 450 {
		t.Errorf("expected DelayMs=450, got %d", ctrl.GetSettings().DelayMs)
	}
}

func TestOverlay_HandleInput_StartAndStop(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	_, _ = ctrl.RefreshInventory()
	overlay := NewOverlay(ctrl)
	overlay.SetVisible(true)
	overlay.SetSelectedRow(8) // Row 8 = Start / Stop Action

	// Press Enter to Start
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyEnter: true})
	overlay.HandleInput(map[ebiten.Key]bool{})

	if !mockBridge.startCalled {
		t.Errorf("expected StartUpgrade to be called on bridge")
	}

	// Press Enter again to Stop
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyEnter: true})
	overlay.HandleInput(map[ebiten.Key]bool{})

	if !mockBridge.stopCalled {
		t.Errorf("expected StopUpgrade to be called on bridge")
	}
}

func TestOverlay_HandleInput_F3_And_Escape(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	overlay := NewOverlay(ctrl)

	// F3 to open
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyF3: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if !overlay.IsVisible() {
		t.Errorf("expected overlay visible after F3")
	}

	// Escape to close
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyEscape: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if overlay.IsVisible() {
		t.Errorf("expected overlay hidden after Escape")
	}

	// F3 to open again
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyF3: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if !overlay.IsVisible() {
		t.Errorf("expected overlay visible after second F3")
	}

	// F3 to close
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyF3: true})
	overlay.HandleInput(map[ebiten.Key]bool{})
	if overlay.IsVisible() {
		t.Errorf("expected overlay hidden after closing F3")
	}
}

func TestOverlay_Draw_And_HUD(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	_, _ = ctrl.RefreshInventory()
	overlay := NewOverlay(ctrl)

	// Create test image buffer (240x320)
	screen := ebiten.NewImage(240, 320)

	// Draw while hidden -> should not draw or panic
	overlay.Draw(screen)

	// DrawHUD while not running -> should not draw
	overlay.DrawHUD(screen)

	// Show and Draw
	overlay.SetVisible(true)
	overlay.Draw(screen)

	// Simulate running upgrade
	mockBridge.isRunning = true
	mockBridge.statusJSON = `{
		"running": true,
		"itemId": 101,
		"itemName": "Kiếm Ánh Sáng +7",
		"itemType": 0,
		"currentLevel": 7,
		"targetLevel": 10,
		"clover3Count": 15,
		"clover4Count": 8,
		"insuranceCount": 2,
		"attempts": 5,
		"successes": 2,
		"failures": 3,
		"message": "Đang đập +7 -> +8...",
		"stopped": false
	}`
	_, _ = ctrl.GetStatus()

	// Draw while running
	overlay.Draw(screen)

	// Hide and DrawHUD while running
	overlay.SetVisible(false)
	overlay.DrawHUD(screen)

	// Safe with nil screen
	overlay.Draw(nil)
	overlay.DrawHUD(nil)
}

func TestOverlay_NilController(t *testing.T) {
	overlay := NewOverlay(nil)
	if overlay == nil {
		t.Fatal("expected non-nil overlay")
	}
	overlay.Toggle()
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyDown: true})
	overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyEnter: true})
	screen := ebiten.NewImage(240, 320)
	overlay.Draw(screen)
	overlay.DrawHUD(screen)
}

func TestOverlay_Concurrency(t *testing.T) {
	mockBridge := NewMockBridge()
	ctrl := NewControllerWithBridge(mockBridge)
	_, _ = ctrl.RefreshInventory()
	overlay := NewOverlay(ctrl)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			overlay.Toggle()
			overlay.HandleInput(map[ebiten.Key]bool{ebiten.KeyDown: true})
			_ = overlay.IsVisible()
			_ = overlay.GetSelectedRow()
			screen := ebiten.NewImage(240, 320)
			overlay.Draw(screen)
			overlay.DrawHUD(screen)
		}(i)
	}
	wg.Wait()
}
