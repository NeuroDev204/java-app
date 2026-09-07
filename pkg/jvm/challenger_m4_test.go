package jvm

import (
	"sync"
	"testing"
)

// TestChallenger_MultipleFillRectCalls tests 100 sequential calls to Graphics.fillRect
// invoked from Java to ensure all calls flow correctly through CGO and are captured in RenderQueue.
func TestChallenger_MultipleFillRectCalls(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	q := GetRenderQueue()
	q.Clear()

	err = vm.InvokeMain("TestMultiple")
	if err != nil {
		t.Fatalf("InvokeMain(\"TestMultiple\") failed: %v", err)
	}

	cmds := q.PopAll()
	// 100 iterations, each calling setColor and fillRect -> total 200 commands
	if len(cmds) != 200 {
		t.Fatalf("expected 200 commands in RenderQueue, got %d", len(cmds))
	}

	for i := 0; i < 100; i++ {
		colorCmd := cmds[i*2]
		fillCmd := cmds[i*2+1]

		if colorCmd.Type != CmdSetColor || colorCmd.RGB != i*10 {
			t.Errorf("cmd %d: expected CmdSetColor RGB 0x%X, got %+v", i*2, i*10, colorCmd)
		}

		expectedX := i
		expectedY := i * 2
		expectedW := i + 10
		expectedH := i + 20

		if fillCmd.Type != CmdFillRect || fillCmd.X != expectedX || fillCmd.Y != expectedY || fillCmd.Width != expectedW || fillCmd.Height != expectedH {
			t.Errorf("cmd %d: expected FillRect(%d,%d,%d,%d), got %+v", i*2+1, expectedX, expectedY, expectedW, expectedH, fillCmd)
		}
	}
}

// TestChallenger_ConcurrentJavaThreads tests multi-threaded Java callbacks calling Graphics.fillRect simultaneously.
func TestChallenger_ConcurrentJavaThreads(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	q := GetRenderQueue()
	q.Clear()

	err = vm.InvokeMain("TestConcurrent")
	if err != nil {
		t.Fatalf("InvokeMain(\"TestConcurrent\") failed: %v", err)
	}

	cmds := q.PopAll()
	// 5 Java threads * 20 iterations * 2 calls per iter (setColor + fillRect) = 200 commands
	if len(cmds) != 200 {
		t.Fatalf("expected 200 commands from multi-threaded Java execution, got %d", len(cmds))
	}

	fillRectCount := 0
	setColorCount := 0
	for _, cmd := range cmds {
		if cmd.Type == CmdFillRect {
			fillRectCount++
		} else if cmd.Type == CmdSetColor {
			setColorCount++
		}
	}

	if fillRectCount != 100 || setColorCount != 100 {
		t.Errorf("expected 100 fillRect and 100 setColor commands, got %d fillRect, %d setColor", fillRectCount, setColorCount)
	}
}

// TestChallenger_RenderQueueConcurrentGoroutines stress tests RenderQueue thread safety with concurrent Go operations.
func TestChallenger_RenderQueueConcurrentGoroutines(t *testing.T) {
	q := GetRenderQueue()
	q.Clear()

	const numGoroutines = 20
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Producers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				q.Push(RenderCommand{
					Type:   CmdFillRect,
					X:      id,
					Y:      j,
					Width:  10,
					Height: 10,
				})
			}
		}(i)
	}

	// Consumers / Inspectors
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = q.Len()
			}
		}()
	}

	wg.Wait()

	if q.Len() != numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d total commands, got %d", numGoroutines*opsPerGoroutine, q.Len())
	}
}

// TestChallenger_InvalidClassNames tests error handling for invalid class names passed to InvokeMain.
func TestChallenger_InvalidClassNames(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	testCases := []struct {
		name      string
		className string
	}{
		{"NonExistentClass", "NoSuchClass"},
		{"EmptyClassName", ""},
		{"SlashFormatInvalid", "invalid/package/DoesNotExist"},
		{"ClassWithoutMain", "TestNoMain"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := vm.InvokeMain(tc.className)
			if err == nil {
				t.Errorf("expected error for invalid class %q, got nil", tc.className)
			}
		})
	}
}

// TestChallenger_NullHandleHandling tests safety when operations are invoked on nil pointers.
func TestChallenger_NullHandleHandling(t *testing.T) {
	var nilJVM *JVM

	if err := nilJVM.RegisterNatives(); err == nil {
		t.Error("expected error calling RegisterNatives on nil JVM, got nil")
	}

	if err := nilJVM.InvokeMain("Test"); err == nil {
		t.Error("expected error calling InvokeMain on nil JVM, got nil")
	}

	emptyJVM := &JVM{}
	if err := emptyJVM.RegisterNatives(); err == nil {
		t.Error("expected error calling RegisterNatives on uninitialized JVM struct, got nil")
	}

	if err := emptyJVM.InvokeMain("Test"); err == nil {
		t.Error("expected error calling InvokeMain on uninitialized JVM struct, got nil")
	}
}
