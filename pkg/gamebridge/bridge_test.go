package gamebridge_test

import (
	"os"
	"testing"
	"time"

	"java-app/pkg/gamebridge"
)

// TestNewBridge verifies that a fresh Bridge starts in the uninitialised state.
func TestNewBridge(t *testing.T) {
	b := gamebridge.NewBridge()
	if b == nil {
		t.Fatal("NewBridge returned nil")
	}
	if b.IsStarted() {
		t.Error("expected IsStarted=false before LoadJAR")
	}
	if b.GetWidth() != 240 {
		t.Errorf("expected width 240, got %d", b.GetWidth())
	}
	if b.GetHeight() != 320 {
		t.Errorf("expected height 320, got %d", b.GetHeight())
	}
}

// TestGetFrameBufferBeforeLoad verifies that GetFrameBuffer returns a zeroed
// buffer of the correct size even before a JAR is loaded.
func TestGetFrameBufferBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	buf := b.GetFrameBuffer()
	expected := 240 * 320 * 4
	if len(buf) != expected {
		t.Errorf("expected frame buffer length %d, got %d", expected, len(buf))
	}
}

// TestLoadJARInvalidBytes verifies that LoadJAR returns a non-empty error
// message when given corrupt data instead of a valid ZIP/JAR.
func TestLoadJARInvalidBytes(t *testing.T) {
	b := gamebridge.NewBridge()
	result := b.LoadJAR([]byte("this is not a jar file"))
	if result == "" {
		t.Error("expected non-empty error for invalid JAR bytes, got empty string")
	}
}

// TestSendTouchBeforeLoad verifies that touch events sent before LoadJAR do
// not panic — they are silently dropped.
func TestSendTouchBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	// Should not panic
	b.SendTouchDown(120, 160)
	b.SendTouchUp(120, 160)
}

// TestStopBeforeLoad verifies that Stop() is safe to call on an uninitialised Bridge.
func TestStopBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	b.Stop() // must not block or panic
}

// TestTickBeforeLoad verifies that Tick() is safe to call on an uninitialised Bridge.
func TestTickBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	b.Tick() // must not panic
}

func TestStartUpgradeNotInGame(t *testing.T) {
	b := gamebridge.NewBridge()
	got := b.StartUpgrade(`{"item_id":1,"item_type":0,"target_level":10,"clover3_min":4,"clover3_max":7,"clover4_min":8,"clover4_max":15,"stop_if_out_of_clover":true,"delay_ms":500}`)
	if got == "" {
		t.Fatal("expected error before in_game")
	}
}

func TestSetShouldPaint(t *testing.T) {
	b := gamebridge.NewBridge()
	if !b.ShouldPaint() {
		t.Fatal("default shouldPaint true")
	}
	b.SetShouldPaint(false)
	if b.ShouldPaint() {
		t.Fatal("expected false")
	}
	buf := b.GetFrameBuffer()
	if len(buf) != 240*320*4 {
		t.Fatalf("len=%d", len(buf))
	}
}

func TestSendTouchDragDoesNotPanic(t *testing.T) {
	b := gamebridge.NewBridge()
	b.SendTouchDrag(10, 20)
}

func TestGetUpgradeStatusJSON(t *testing.T) {
	b := gamebridge.NewBridge()
	raw := b.GetUpgradeStatus()
	if raw == "" {
		t.Fatal("expected json")
	}
}

func TestLoadJARPaintsNonDefaultPixels(t *testing.T) {
	data, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Skip(err)
	}
	b := gamebridge.NewBridge()
	defer b.Stop()
	if msg := b.LoadJAR(data); msg != "" {
		t.Fatalf("LoadJAR: %s", msg)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		buf := b.GetFrameBuffer()
		changed := 0
		for i := 0; i+3 < len(buf); i += 4 {
			if buf[i] != 15 || buf[i+1] != 25 || buf[i+2] != 45 {
				changed++
			}
		}
		if changed > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("framebuffer stayed default navy RGB(15,25,45) after boot — screen paint did not run")
}

func TestTextInputBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	info := b.GetTextInputInfoJSON()
	if info != `{"active":false}` {
		t.Fatalf("expected inactive json, got %s", info)
	}
	// Calling submit or cancel before load should not panic
	b.SubmitTextInput("hello")
	b.CancelTextInput()
}

// TestSendKeyPressBeforeLoad verifies that SendKeyPress, SendKeyDown, and SendKeyUp
// do not panic when invoked before a JAR is loaded.
func TestSendKeyPressBeforeLoad(t *testing.T) {
	b := gamebridge.NewBridge()
	b.SendKeyDown(-5)
	b.SendKeyUp(-5)
	b.SendKeyPress(-5)
	time.Sleep(70 * time.Millisecond)
}

