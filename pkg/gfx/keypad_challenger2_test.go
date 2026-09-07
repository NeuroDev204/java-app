package gfx

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestEmpirical_KeypadMapping_Digits4to9AndAllKeys(t *testing.T) {
	tests := []struct {
		key      ebiten.Key
		expected int
		name     string
	}{
		{ebiten.KeyDigit0, '0', "Digit 0"},
		{ebiten.KeyDigit1, '1', "Digit 1"},
		{ebiten.KeyDigit2, '2', "Digit 2"},
		{ebiten.KeyDigit3, '3', "Digit 3"},
		{ebiten.KeyDigit4, '4', "Digit 4"},
		{ebiten.KeyDigit5, '5', "Digit 5"},
		{ebiten.KeyDigit6, '6', "Digit 6"},
		{ebiten.KeyDigit7, '7', "Digit 7"},
		{ebiten.KeyDigit8, '8', "Digit 8"},
		{ebiten.KeyDigit9, '9', "Digit 9"},
		{ebiten.KeyNumpad0, '0', "Numpad 0"},
		{ebiten.KeyNumpad1, '1', "Numpad 1"},
		{ebiten.KeyNumpad2, '2', "Numpad 2"},
		{ebiten.KeyNumpad3, '3', "Numpad 3"},
		{ebiten.KeyNumpad4, '4', "Numpad 4"},
		{ebiten.KeyNumpad5, '5', "Numpad 5"},
		{ebiten.KeyNumpad6, '6', "Numpad 6"},
		{ebiten.KeyNumpad7, '7', "Numpad 7"},
		{ebiten.KeyNumpad8, '8', "Numpad 8"},
		{ebiten.KeyNumpad9, '9', "Numpad 9"},
		{ebiten.KeyUp, -1, "Up Arrow"},
		{ebiten.KeyDown, -2, "Down Arrow"},
		{ebiten.KeyLeft, -3, "Left Arrow"},
		{ebiten.KeyRight, -4, "Right Arrow"},
		{ebiten.KeyEnter, -5, "Enter / Fire"},
		{ebiten.KeyA, 'A', "Mapped Key A"},
		{ebiten.KeySpace, 0, "Unmapped Key Space"},
	}

	for _, tt := range tests {
		got := MapEbitenKeyToJ2ME(tt.key)
		if got != tt.expected {
			t.Errorf("FAIL [%s]: key %v mapped to %d (%q), expected %d (%q)",
				tt.name, tt.key, got, rune(got), tt.expected, rune(tt.expected))
		} else {
			t.Logf("PASS [%s]: key %v mapped to %d (%q)", tt.name, tt.key, got, rune(got))
		}
	}
}
