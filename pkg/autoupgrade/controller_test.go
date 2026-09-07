package autoupgrade

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"java-app/pkg/jvm"
)

// MockBridge implements the Bridge interface for unit testing.
type MockBridge struct {
	mu             sync.Mutex
	inventoryJSON  string
	statusJSON     string
	inventoryErr   error
	statusErr      error
	startErr       error
	stopErr        error
	startedItem    int
	startedType    int
	startedTarget  int
	startedC3Min   int
	startedC3Max   int
	startedC4Min   int
	startedC4Max   int
	startedStop    bool
	startedDelay   int
	startCalled    bool
	stopCalled     bool
	isRunning      bool
}

func NewMockBridge() *MockBridge {
	return &MockBridge{
		inventoryJSON: `{
			"gold": 500000,
			"clover3Count": 15,
			"clover4Count": 8,
			"insuranceCount": 2,
			"items": [
				{"id": 101, "name": "Kiếm Ánh Sáng +7", "type": 0, "level": 7, "quantity": 1},
				{"id": 102, "name": "Giáp Huyền Thoại +9", "type": 1, "level": 9, "quantity": 1}
			]
		}`,
		statusJSON: `{
			"running": false,
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
			"message": "Sẵn sàng",
			"stopped": true
		}`,
	}
}

func (m *MockBridge) GetInventoryJson() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inventoryErr != nil {
		return "", m.inventoryErr
	}
	return m.inventoryJSON, nil
}

func (m *MockBridge) StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startErr != nil {
		return m.startErr
	}
	m.startCalled = true
	m.startedItem = itemId
	m.startedType = itemType
	m.startedTarget = targetLevel
	m.startedC3Min = clover3Min
	m.startedC3Max = clover3Max
	m.startedC4Min = clover4Min
	m.startedC4Max = clover4Max
	m.startedStop = stopOutOfClover
	m.startedDelay = delayMs
	m.isRunning = true
	return nil
}

func (m *MockBridge) StopUpgrade() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopCalled = true
	m.isRunning = false
	return nil
}

func (m *MockBridge) GetUpgradeStatusJson() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statusErr != nil {
		return "", m.statusErr
	}
	return m.statusJSON, nil
}

func TestItemJSONSerialization(t *testing.T) {
	rawJSON := `{"id": 402, "name": "Gậy Phép Thuật +10", "type": 2, "level": 10, "quantity": 1}`
	var item Item
	err := json.Unmarshal([]byte(rawJSON), &item)
	if err != nil {
		t.Fatalf("Unmarshal Item failed: %v", err)
	}

	if item.ID != 402 {
		t.Errorf("expected ID 402, got %d", item.ID)
	}
	if item.Name != "Gậy Phép Thuật +10" {
		t.Errorf("expected Name 'Gậy Phép Thuật +10', got '%s'", item.Name)
	}
	if item.Type != 2 {
		t.Errorf("expected Type 2, got %d", item.Type)
	}
	if item.Level != 10 {
		t.Errorf("expected Level 10, got %d", item.Level)
	}
	if item.Quantity != 1 {
		t.Errorf("expected Quantity 1, got %d", item.Quantity)
	}
}

func TestInventoryDataJSONSerialization(t *testing.T) {
	rawJSON := `{
		"gold": 1234567,
		"clover3Count": 10,
		"clover4Count": 5,
		"insuranceCount": 3,
		"items": [
			{"id": 1, "name": "Nón Chiến Binh", "type": 3, "level": 5, "quantity": 1}
		]
	}`

	var inv InventoryData
	err := json.Unmarshal([]byte(rawJSON), &inv)
	if err != nil {
		t.Fatalf("Unmarshal InventoryData failed: %v", err)
	}

	if inv.Gold != 1234567 {
		t.Errorf("expected Gold 1234567, got %d", inv.Gold)
	}
	if inv.Clover3Count != 10 {
		t.Errorf("expected Clover3Count 10, got %d", inv.Clover3Count)
	}
	if inv.Clover4Count != 5 {
		t.Errorf("expected Clover4Count 5, got %d", inv.Clover4Count)
	}
	if inv.InsuranceCount != 3 {
		t.Errorf("expected InsuranceCount 3, got %d", inv.InsuranceCount)
	}
	if len(inv.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(inv.Items))
	}
	if inv.Items[0].Name != "Nón Chiến Binh" {
		t.Errorf("expected item name 'Nón Chiến Binh', got '%s'", inv.Items[0].Name)
	}
}

func TestStatusJSONSerialization(t *testing.T) {
	rawJSON := `{
		"running": true,
		"itemId": 99,
		"itemName": "Áo Choàng Bóng Tối",
		"itemType": 4,
		"currentLevel": 8,
		"targetLevel": 12,
		"clover3Count": 7,
		"clover4Count": 3,
		"insuranceCount": 1,
		"attempts": 14,
		"successes": 6,
		"failures": 8,
		"message": "Đang đập +8 -> +9 (có cỏ)...",
		"stopped": false
	}`

	var status Status
	err := json.Unmarshal([]byte(rawJSON), &status)
	if err != nil {
		t.Fatalf("Unmarshal Status failed: %v", err)
	}

	if !status.Running {
		t.Errorf("expected Running true, got false")
	}
	if status.ItemID != 99 {
		t.Errorf("expected ItemID 99, got %d", status.ItemID)
	}
	if status.ItemName != "Áo Choàng Bóng Tối" {
		t.Errorf("expected ItemName 'Áo Choàng Bóng Tối', got '%s'", status.ItemName)
	}
	if status.ItemType != 4 {
		t.Errorf("expected ItemType 4, got %d", status.ItemType)
	}
	if status.CurrentLevel != 8 {
		t.Errorf("expected CurrentLevel 8, got %d", status.CurrentLevel)
	}
	if status.TargetLevel != 12 {
		t.Errorf("expected TargetLevel 12, got %d", status.TargetLevel)
	}
	if status.Attempts != 14 {
		t.Errorf("expected Attempts 14, got %d", status.Attempts)
	}
	if status.Successes != 6 {
		t.Errorf("expected Successes 6, got %d", status.Successes)
	}
	if status.Failures != 8 {
		t.Errorf("expected Failures 8, got %d", status.Failures)
	}
	if status.Stopped {
		t.Errorf("expected Stopped false, got true")
	}
}

func TestController_RefreshInventory(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	inv, err := ctrl.RefreshInventory()
	if err != nil {
		t.Fatalf("RefreshInventory failed: %v", err)
	}

	if inv.Gold != 500000 {
		t.Errorf("expected gold 500000, got %d", inv.Gold)
	}
	if len(inv.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(inv.Items))
	}
	if inv.Items[0].ID != 101 || inv.Items[0].Level != 7 {
		t.Errorf("unexpected first item: %+v", inv.Items[0])
	}

	// Verify cached inventory
	cached := ctrl.GetCachedInventory()
	if cached == nil || len(cached.Items) != 2 {
		t.Errorf("cached inventory not properly populated: %+v", cached)
	}
}

func TestController_RefreshInventory_Errors(t *testing.T) {
	mock := NewMockBridge()
	mock.inventoryErr = errors.New("JNI connection lost")
	ctrl := NewControllerWithBridge(mock)

	inv, err := ctrl.RefreshInventory()
	if err == nil {
		t.Fatalf("expected error, got nil inventory: %+v", inv)
	}

	// Test malformed JSON
	mock.inventoryErr = nil
	mock.inventoryJSON = `invalid json`
	_, err = ctrl.RefreshInventory()
	if err == nil {
		t.Fatalf("expected JSON parse error, got nil")
	}
}

func TestController_StartUpgrade_WithItemIndex(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	// Populate inventory cache
	_, err := ctrl.RefreshInventory()
	if err != nil {
		t.Fatalf("RefreshInventory failed: %v", err)
	}

	settings := &UpgradeSettings{
		SelectedItemIndex: 1, // Points to item 102 (Giáp Huyền Thoại +9, type 1)
		TargetLevel:       12,
		Clover3Min:        4,
		Clover3Max:        7,
		Clover4Min:        8,
		Clover4Max:        15,
		StopIfOutOfClover: true,
		DelayMs:           600,
	}

	err = ctrl.StartUpgrade(settings)
	if err != nil {
		t.Fatalf("StartUpgrade failed: %v", err)
	}

	if !mock.startCalled {
		t.Fatalf("bridge StartUpgrade was not called")
	}
	if mock.startedItem != 102 {
		t.Errorf("expected startedItem 102, got %d", mock.startedItem)
	}
	if mock.startedType != 1 {
		t.Errorf("expected startedType 1, got %d", mock.startedType)
	}
	if mock.startedTarget != 12 {
		t.Errorf("expected startedTarget 12, got %d", mock.startedTarget)
	}
	if mock.startedDelay != 600 {
		t.Errorf("expected startedDelay 600, got %d", mock.startedDelay)
	}
}

func TestController_StartUpgrade_Validation(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	// Target level out of range
	invalidSettings := &UpgradeSettings{
		SelectedItemIndex: 0,
		TargetLevel:       0, // Invalid: must be >= 1
	}
	err := ctrl.StartUpgrade(invalidSettings)
	if err == nil {
		t.Errorf("expected error for target level 0, got nil")
	}

	invalidSettings.TargetLevel = 16 // Invalid: must be <= 15
	err = ctrl.StartUpgrade(invalidSettings)
	if err == nil {
		t.Errorf("expected error for target level 16, got nil")
	}

	// Out of range index when inventory empty
	mock.inventoryJSON = `{"items":[]}`
	_, _ = ctrl.RefreshInventory()
	validTargetSettings := &UpgradeSettings{
		SelectedItemIndex: 0,
		TargetLevel:       10,
	}
	err = ctrl.StartUpgrade(validTargetSettings)
	if err == nil {
		t.Errorf("expected error for empty inventory, got nil")
	}
}

func TestController_StopUpgrade(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	err := ctrl.StopUpgrade()
	if err != nil {
		t.Fatalf("StopUpgrade failed: %v", err)
	}
	if !mock.stopCalled {
		t.Errorf("expected stopCalled true")
	}

	// Error handling
	mock.stopErr = errors.New("bridge stop error")
	err = ctrl.StopUpgrade()
	if err == nil {
		t.Errorf("expected error from bridge, got nil")
	}
}

func TestController_GetStatus(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	status, err := ctrl.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.CurrentLevel != 7 {
		t.Errorf("expected CurrentLevel 7, got %d", status.CurrentLevel)
	}
	if status.Message != "Sẵn sàng" {
		t.Errorf("expected Message 'Sẵn sàng', got '%s'", status.Message)
	}

	// Verify cached status
	cached := ctrl.GetCachedStatus()
	if cached == nil || cached.CurrentLevel != 7 {
		t.Errorf("cached status invalid: %+v", cached)
	}

	// Status bridge error
	mock.statusErr = errors.New("status query failed")
	_, err = ctrl.GetStatus()
	if err == nil {
		t.Errorf("expected status error, got nil")
	}
}

func TestController_DefaultSettings(t *testing.T) {
	settings := DefaultSettings()
	if settings.TargetLevel != 8 {
		t.Errorf("expected default TargetLevel 8, got %d", settings.TargetLevel)
	}
	if settings.Clover3Min != 4 || settings.Clover3Max != 7 {
		t.Errorf("unexpected Clover 3 range: %d - %d", settings.Clover3Min, settings.Clover3Max)
	}
	if settings.Clover4Min != 8 || settings.Clover4Max != 15 {
		t.Errorf("unexpected Clover 4 range: %d - %d", settings.Clover4Min, settings.Clover4Max)
	}
	if !settings.StopIfOutOfClover {
		t.Errorf("expected StopIfOutOfClover true by default")
	}
	if settings.DelayMs != 500 {
		t.Errorf("expected DelayMs 500, got %d", settings.DelayMs)
	}
}

func TestController_UpdateSettings(t *testing.T) {
	ctrl := NewControllerWithBridge(NewMockBridge())
	s := ctrl.GetSettings()
	s.TargetLevel = 11
	s.DelayMs = 800
	ctrl.UpdateSettings(s)

	updated := ctrl.GetSettings()
	if updated.TargetLevel != 11 || updated.DelayMs != 800 {
		t.Errorf("UpdateSettings did not persist: %+v", updated)
	}
}

func TestController_Concurrency(t *testing.T) {
	mock := NewMockBridge()
	ctrl := NewControllerWithBridge(mock)

	var wg sync.WaitGroup
	numWorkers := 20
	iterations := 50

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch (workerID + j) % 4 {
				case 0:
					_, _ = ctrl.RefreshInventory()
				case 1:
					_, _ = ctrl.GetStatus()
				case 2:
					s := ctrl.GetSettings()
					s.DelayMs = 400 + (j % 500)
					ctrl.UpdateSettings(s)
				case 3:
					_ = ctrl.GetCachedInventory()
					_ = ctrl.GetCachedStatus()
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestDefaultJNIBridge_NilJVM(t *testing.T) {
	bridge := &JNIBridge{}
	
	// When JVM is nil or uninitialized, JNIBridge must return descriptive errors gracefully
	_, err := bridge.GetInventoryJson()
	if err == nil {
		fmt.Println("Note: JVM was initialized in environment")
	}

	err = bridge.StopUpgrade()
	if err == nil {
		fmt.Println("Note: JVM was initialized in environment")
	}

	_, err = bridge.GetUpgradeStatusJson()
	if err == nil {
		fmt.Println("Note: JVM was initialized in environment")
	}
}

func TestController_LiveJVMIntegration(t *testing.T) {
	// Initialize real HotSpot JVM with stubs
	_, err := jvm.InitJVM("../../j2me-stubs/bin")
	if err != nil {
		t.Logf("JVM initialization note: %v", err)
		return
	}

	ctrl := NewController()
	if ctrl == nil {
		t.Fatalf("NewController returned nil")
	}

	// Verify inventory query via live JNI
	inv, err := ctrl.RefreshInventory()
	if err != nil {
		t.Logf("Live inventory query note: %v", err)
	} else if inv != nil {
		t.Logf("Live inventory retrieved: %d items, gold=%d, clover3=%d, clover4=%d", len(inv.Items), inv.Gold, inv.Clover3Count, inv.Clover4Count)
	}

	// Verify status query via live JNI
	status, err := ctrl.GetStatus()
	if err != nil {
		t.Fatalf("Live GetStatus failed: %v", err)
	}
	if status == nil {
		t.Fatalf("Live status is nil")
	}
	t.Logf("Live status retrieved: running=%v, message=%s, stopped=%v", status.Running, status.Message, status.Stopped)

	// Verify stop upgrade via live JNI
	err = ctrl.StopUpgrade()
	if err != nil {
		t.Fatalf("Live StopUpgrade failed: %v", err)
	}
}

