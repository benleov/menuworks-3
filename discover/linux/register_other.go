//go:build !linux

// Package linux provides stubs for non-Linux platforms.
package linux

import (
	"github.com/benworks/menuworks/discover"
)

// RegisterAll is a no-op on non-Linux platforms.
func RegisterAll(r *discover.Registry) {
	// No Linux sources available on this platform
}
