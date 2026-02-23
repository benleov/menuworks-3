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
	r.Register(&XboxSource{})
	r.Register(&ProgramFilesSource{})
}

// RegisterCustomDirs registers one CustomDirSource per entry in dirs.
func RegisterCustomDirs(r *discover.Registry, dirs []discover.DirEntry) {
	for _, d := range dirs {
		r.Register(&CustomDirSource{Dir: d.Dir, MenuName: d.Name, Exclude: d.Exclude})
	}
}
