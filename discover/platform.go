package discover

import "runtime"

// getGOOS returns the runtime GOOS value. Exists for testability.
func getGOOS() string {
	return runtime.GOOS
}
