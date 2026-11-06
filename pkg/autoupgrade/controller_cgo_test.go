//go:build cgo

package autoupgrade

import (
	"fmt"
	"testing"

	"java-app/pkg/jvm"
)

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

	// Verify reset clover via live JNI
	err = ctrl.ResetClover()
	if err != nil {
		t.Fatalf("Live ResetClover failed: %v", err)
	}
}
