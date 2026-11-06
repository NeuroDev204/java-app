package main

import (
	"os"
	"testing"
)

func TestNewDesktopGame(t *testing.T) {
	game := NewDesktopGame("")
	if game == nil {
		t.Fatal("NewDesktopGame returned nil")
	}
	if !game.showMenu {
		t.Error("expected showMenu=true when no jar path provided")
	}

	w, h := game.Layout(480, 640)
	if w != 240 || h != 320 {
		t.Errorf("expected 240x320 layout, got %dx%d", w, h)
	}
}

func TestDesktopGameBootAndTick(t *testing.T) {
	jarPath := "../../mobile/assets/KnightOnline_402.jar"
	data, err := os.ReadFile(jarPath)
	if err != nil {
		t.Skipf("skipping jar test: %v", err)
	}

	game := NewDesktopGame(jarPath)
	if err := game.startWithJarBytes(data); err != nil {
		t.Fatalf("startWithJarBytes: %v", err)
	}

	if game.bridge == nil || !game.bridge.IsStarted() {
		t.Fatal("expected bridge started")
	}

	// Update loop tick
	if err := game.Update(); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	buf := game.bridge.GetFrameBuffer()
	if len(buf) != 240*320*4 {
		t.Fatalf("expected 240*320*4 pixels, got %d", len(buf))
	}
}

func TestDesktopGameF3OverlayToggle(t *testing.T) {
	game := NewDesktopGame("")
	if game.upgradeOverlay == nil {
		t.Fatal("expected upgradeOverlay initialized")
	}
	if game.upgradeOverlay.IsVisible() {
		t.Error("expected overlay not visible initially")
	}

	game.upgradeOverlay.Toggle()
	if !game.upgradeOverlay.IsVisible() {
		t.Error("expected overlay visible after toggle")
	}

	game.upgradeOverlay.Toggle()
	if game.upgradeOverlay.IsVisible() {
		t.Error("expected overlay not visible after second toggle")
	}
}
