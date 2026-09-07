package jvm

import (
	"testing"
)

func TestInitJVM(t *testing.T) {
	vm, err := InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	if vm == nil {
		t.Fatal("expected non-nil JVM instance")
	}

	if vm.GetEnv() == nil {
		t.Error("expected non-nil JNIEnv pointer")
	}

	if vm.GetVM() == nil {
		t.Error("expected non-nil JavaVM pointer")
	}

	if vm.Env() == nil {
		t.Error("expected non-nil Env() pointer")
	}

	if vm.VM() == nil {
		t.Error("expected non-nil VM() pointer")
	}
}

func TestInitJVMIdempotent(t *testing.T) {
	vm1, err1 := InitJVM("./j2me-stubs/bin")
	if err1 != nil {
		t.Fatalf("first InitJVM failed: %v", err1)
	}

	vm2, err2 := InitJVM("./j2me-stubs/bin")
	if err2 != nil {
		t.Fatalf("second InitJVM failed: %v", err2)
	}

	if vm1 != vm2 {
		t.Errorf("expected idempotent InitJVM to return same pointer %p, got %p", vm1, vm2)
	}
}

func TestGetJVM(t *testing.T) {
	vm, err := InitJVM("")
	if err != nil {
		t.Fatalf("InitJVM failed: %v", err)
	}

	global := GetJVM()
	if global != vm {
		t.Errorf("expected GetJVM() %p to match InitJVM() %p", global, vm)
	}
}

func TestNilJVMSafety(t *testing.T) {
	var nilJVM *JVM

	if nilJVM.GetEnv() != nil {
		t.Error("expected nil for nilJVM.GetEnv()")
	}
	if nilJVM.GetVM() != nil {
		t.Error("expected nil for nilJVM.GetVM()")
	}
	if nilJVM.Env() != nil {
		t.Error("expected nil for nilJVM.Env()")
	}
	if nilJVM.VM() != nil {
		t.Error("expected nil for nilJVM.VM()")
	}

	// Calling Destroy on nil should not panic
	nilJVM.Destroy()
}
