package gfx

import "github.com/hajimehoshi/ebiten/v2"

func MapEbitenKeyToJ2ME(key ebiten.Key) int {
	if key >= ebiten.KeyA && key <= ebiten.KeyZ {
		// Many J2ME games expect AWT keycodes (VK_A = 65) for QWERTY typing on emulators
		return int('A') + int(key-ebiten.KeyA)
	}

	switch key {
	case ebiten.KeyNumpad0, ebiten.KeyDigit0:
		return '0'
	case ebiten.KeyNumpad1, ebiten.KeyDigit1:
		return '1'
	case ebiten.KeyNumpad2, ebiten.KeyDigit2:
		return '2'
	case ebiten.KeyNumpad3, ebiten.KeyDigit3:
		return '3'
	case ebiten.KeyNumpad4, ebiten.KeyDigit4:
		return '4'
	case ebiten.KeyNumpad5, ebiten.KeyDigit5:
		return '5'
	case ebiten.KeyNumpad6, ebiten.KeyDigit6:
		return '6'
	case ebiten.KeyNumpad7, ebiten.KeyDigit7:
		return '7'
	case ebiten.KeyNumpad8, ebiten.KeyDigit8:
		return '8'
	case ebiten.KeyNumpad9, ebiten.KeyDigit9:
		return '9'
	case ebiten.KeyUp:
		return -1 // UP
	case ebiten.KeyDown:
		return -2 // DOWN
	case ebiten.KeyLeft:
		return -3 // LEFT
	case ebiten.KeyRight:
		return -4 // RIGHT
	case ebiten.KeyEnter:
		return -5 // FIRE
	case ebiten.KeyBackspace:
		return 8 // BACKSPACE
	case ebiten.KeyF1:
		return -6 // SOFTKEY 1 (Trái)
	case ebiten.KeyF2:
		return -7 // SOFTKEY 2 (Phải)
	case ebiten.KeyTab:
		return 35 // '#' / KEY_POUND (Switch instance)
	case ebiten.KeyNumpadMultiply:
		return 42 // '*' / KEY_STAR (Switch instance)
	default:
		return 0
	}
}
