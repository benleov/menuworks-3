//go:build windows

// Package windows provides Windows-specific application discovery sources.
package windows

import (
	"github.com/benworks/menuworks/discover"
)

// RegisterAll registers all Windows discovery sources with the given registry.
func RegisterAll(r *discover.Registry) {
	r.Register(&StartMenuSource{})
	r.Register(&SteamSource{})
	r.Register(&ProgramFilesSource{})
}
