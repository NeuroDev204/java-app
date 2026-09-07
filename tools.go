//go:build tools

package tools

// Keep golang.org/x/mobile in go.mod so `gomobile bind` can import
// golang.org/x/mobile/bind. go mod tidy drops it otherwise.
import (
	_ "golang.org/x/mobile/bind"
)
