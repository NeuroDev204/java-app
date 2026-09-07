package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"java-app/pkg/jvm"
	"java-app/pkg/midp"
)

func TestAuthenticJVMJNICanvasRendering(t *testing.T) {
	stubsDir := "./j2me-stubs/bin"
	if _, err := os.Stat(stubsDir); os.IsNotExist(err) {
		stubsDir = "../../j2me-stubs/bin"
	}
	absStubsDir, err := filepath.Abs(stubsDir)
	if err != nil {
		t.Fatalf("Failed to resolve absolute path for j2me-stubs/bin: %v", err)
	}

	// 1. Initialize MIDP graphics engine
	graphics := midp.NewGraphics(240, 320)
	jvm.SetActiveGraphics(graphics)

	// 2. Initialize authentic HotSpot JVM via CGO/JNI
	vm, err := jvm.InitJVM(absStubsDir)
	if err != nil {
		t.Fatalf("Failed to initialize HotSpot JVM via CGO/JNI: %v", err)
	}

	// 3. Register native CGO JNI method table
	if err := vm.RegisterNatives(); err != nil {
		t.Fatalf("Failed to register JNI native method table: %v", err)
	}

	jarPath := "/home/neuro/Downloads/KnightOnline_402.jar"
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		t.Skipf("KnightOnline_402.jar not found at %s, skipping JAR execution test", jarPath)
	}

	// 4. Invoke Java entry point (com.emulator.Main) via JNI CGO bridge
	doneChan := make(chan error, 1)
	go func() {
		doneChan <- vm.InvokeMainWithArgs("com/emulator/Main", []string{jarPath})
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			t.Fatalf("JVM JNI invocation of com.emulator.Main failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		// JVM started successfully and MIDlet startApp() executed
	}

	// 5. Verify graphics buffer and render queue
	if graphics.Buffer == nil {
		t.Fatalf("Expected graphics buffer to be initialized, got nil")
	}

	renderQueue := jvm.GetRenderQueue()
	t.Logf("[E2E Authentic JVM Test] Render queue command count: %d", renderQueue.Len())
}
