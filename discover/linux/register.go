//go:build linux

// Package linux provides Linux-specific application discovery sources.
package linux

import (
	"github.com/benworks/menuworks/discover"
)

// RegisterAll registers all Linux discovery sources with the given registry.
func RegisterAll(r *discover.Registry) {
	r.Register(&DesktopSource{})
	r.Register(&SteamSource{})
	r.Register(&FlatpakSource{})
	r.Register(&SnapSource{})
}
