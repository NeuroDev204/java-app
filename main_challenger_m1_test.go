package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestChallengerGameInterface verifies interface satisfaction, boundary conditions, and state transitions.
func TestChallengerGameInterface(t *testing.T) {
	var g interface{} = &Game{}
	if _, ok := g.(ebiten.Game); !ok {
		t.Fatalf("*Game does not implement ebiten.Game interface")
	}

	game := &Game{}

	// Verify initial state
	if game.tickCount != 0 {
		t.Errorf("expected initial tickCount 0, got %d", game.tickCount)
	}

	// 1. Boundary & Extreme values for Layout
	testLayoutCases := []struct {
		inW, inH     int
		wantW, wantH int
	}{
		{0, 0, 240, 320},
		{-100, -200, 240, 320},
		{240, 320, 240, 320},
		{1920, 1080, 240, 320},
		{math.MaxInt32, math.MaxInt32, 240, 320},
	}

	for _, tc := range testLayoutCases {
		w, h := game.Layout(tc.inW, tc.inH)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("Layout(%d, %d) = (%d, %d); want (%d, %d)", tc.inW, tc.inH, w, h, tc.wantW, tc.wantH)
		}
	}

	// 2. High-volume Update loop simulation
	const iterations = 1_000_000
	for i := 0; i < iterations; i++ {
		if err := game.Update(); err != nil {
			t.Fatalf("Update() returned error at step %d: %v", i, err)
		}
	}

	if game.tickCount != uint64(iterations) {
		t.Errorf("expected tickCount %d, got %d", iterations, game.tickCount)
	}
}

// TestChallengerTickOverflow verifies behavior near uint64 limits.
func TestChallengerTickOverflow(t *testing.T) {
	game := &Game{tickCount: math.MaxUint64 - 1}

	if err := game.Update(); err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if game.tickCount != math.MaxUint64 {
		t.Errorf("expected tickCount %d, got %d", uint64(math.MaxUint64), game.tickCount)
	}

	// Overflow step
	if err := game.Update(); err != nil {
		t.Fatalf("Update error on overflow: %v", err)
	}
	if game.tickCount != 0 {
		t.Errorf("expected tickCount to wrap to 0, got %d", game.tickCount)
	}
}

// TestChallengerSequentialUpdateStress stress-tests 100,000 sequential Update calls.
func TestChallengerSequentialUpdateStress(t *testing.T) {
	game := &Game{}

	for i := 0; i < 100000; i++ {
		if err := game.Update(); err != nil {
			t.Fatalf("Update failed at frame %d: %v", i, err)
		}
	}

	if game.tickCount != 100000 {
		t.Errorf("expected tickCount 100000 after 100000 updates, got %d", game.tickCount)
	}
}

// BenchmarkGameUpdate benchmarks the single-frame update logic.
func BenchmarkGameUpdate(b *testing.B) {
	game := &Game{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = game.Update()
	}
}

// BenchmarkGameLayout benchmarks logical layout resolution.
func BenchmarkGameLayout(b *testing.B) {
	game := &Game{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = game.Layout(1920, 1080)
	}
}
