//go:build !windows

// Package windows provides stubs for non-Windows platforms.
package windows

import (
	"github.com/benworks/menuworks/discover"
)

// RegisterAll is a no-op on non-Windows platforms.
func RegisterAll(r *discover.Registry) {
	// No Windows sources available on this platform
}
