package jvm

import (
	"sync"
	"testing"
)

// TestChallengerMultipleFillRect verifies that multiple sequential calls to fillRect from Java
// are correctly captured in order and pushed into the RenderQueue.
func TestChallengerMultipleFillRect(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	q := GetRenderQueue()
	q.Clear()

	err = vm.InvokeMain("MultiFillRectTest")
	if err != nil {
		t.Fatalf("InvokeMain(\"MultiFillRectTest\") failed: %v", err)
	}

	cmds := q.PopAll()
	if len(cmds) != 100 {
		t.Fatalf("expected 100 fillRect commands, got %d", len(cmds))
	}

	for i, cmd := range cmds {
		if cmd.Type != CmdFillRect {
			t.Errorf("command %d: expected CmdFillRect, got %v", i, cmd.Type)
		}
		if cmd.X != i || cmd.Y != i*2 || cmd.Width != 10+i || cmd.Height != 20+i {
			t.Errorf("command %d mismatch: got X=%d Y=%d W=%d H=%d, expected X=%d Y=%d W=%d H=%d",
				i, cmd.X, cmd.Y, cmd.Width, cmd.Height, i, i*2, 10+i, 20+i)
		}
	}
}

// TestChallengerConcurrentJavaThreads verifies concurrent JNI callbacks from multiple Java threads
// combined with concurrent Go operations on the RenderQueue.
func TestChallengerConcurrentJavaThreads(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	q := GetRenderQueue()
	q.Clear()

	var goWg sync.WaitGroup
	stopGoWorkers := make(chan struct{})

	// Goroutine 1: Continuously pops commands
	goWg.Add(1)
	go func() {
		defer goWg.Done()
		for {
			select {
			case <-stopGoWorkers:
				return
			default:
				_ = q.PopAll()
			}
		}
	}()

	// Goroutine 2: Continuously pushes background Go commands
	goWg.Add(1)
	go func() {
		defer goWg.Done()
		for {
			select {
			case <-stopGoWorkers:
				return
			default:
				q.Push(RenderCommand{Type: CmdSetColor, RGB: 0x123456})
			}
		}
	}()

	// Invoke multi-threaded Java test (10 Java threads x 50 calls = 500 calls)
	err = vm.InvokeMain("ConcurrentJavaTest")
	if err != nil {
		t.Fatalf("InvokeMain(\"ConcurrentJavaTest\") failed: %v", err)
	}

	close(stopGoWorkers)
	goWg.Wait()
}

// TestChallengerInvalidClassNames verifies graceful error handling for missing classes,
// classes without main methods, and classes whose main method throws Java exceptions.
func TestChallengerInvalidClassNames(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	// 1. Missing class
	err = vm.InvokeMain("NonExistentClass")
	if err == nil {
		t.Error("expected error for non-existent class, got nil")
	}

	// 2. Class without main method
	err = vm.InvokeMain("javax/microedition/lcdui/Graphics")
	if err == nil {
		t.Error("expected error for class without main method, got nil")
	}

	// 3. Class throwing Java exception in main
	err = vm.InvokeMain("ExceptionTest")
	if err == nil {
		t.Error("expected error for class throwing exception in main, got nil")
	}
}

// TestChallengerNullHandleHandling verifies safety against nil receiver or zeroed JVM structs.
func TestChallengerNullHandleHandling(t *testing.T) {
	var nilJVM *JVM

	if err := nilJVM.InvokeMain("Test"); err == nil {
		t.Error("expected error calling InvokeMain on nil JVM pointer, got nil")
	}

	if err := nilJVM.RegisterNatives(); err == nil {
		t.Error("expected error calling RegisterNatives on nil JVM pointer, got nil")
	}

	zeroJVM := &JVM{}
	if err := zeroJVM.InvokeMain("Test"); err == nil {
		t.Error("expected error calling InvokeMain on zeroed JVM struct, got nil")
	}

	if err := zeroJVM.RegisterNatives(); err == nil {
		t.Error("expected error calling RegisterNatives on zeroed JVM struct, got nil")
	}

	// Safety of getters on nil
	if nilJVM.GetEnv() != nil || nilJVM.GetVM() != nil || nilJVM.Env() != nil || nilJVM.VM() != nil {
		t.Error("expected nil pointers from nil JVM getters")
	}

	// Destroy on nil receiver or zeroed struct should not panic
	nilJVM.Destroy()
	zeroJVM.Destroy()
}
