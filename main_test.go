package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"java-app/pkg/jvm"
)

func TestGameInterface(t *testing.T) {
	game := &Game{}

	// Test Update
	err := game.Update()
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	if game.tickCount != 1 {
		t.Errorf("expected tickCount 1 after one Update(), got %d", game.tickCount)
	}

	// Test Layout
	sw, sh := game.Layout(640, 480)
	if sw != 240 || sh != 320 {
		t.Errorf("expected layout 240x320, got %dx%d", sw, sh)
	}

	// Test Draw with dummy screen image
	screen := ebiten.NewImage(240, 320)
	game.Draw(screen)
}

func TestJVMInitializationInMain(t *testing.T) {
	vm, err := jvm.InitJVM("./j2me-stubs/bin")
	if err != nil {
		t.Fatalf("InitJVM in main package failed: %v", err)
	}
	if vm == nil {
		t.Fatal("expected non-nil JVM instance")
	}
	if vm.GetEnv() == nil {
		t.Error("expected non-nil JNIEnv pointer")
	}
}
