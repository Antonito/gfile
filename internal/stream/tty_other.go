//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package stream

import "os"

// disableCanonicalMode is a no-op on platforms without POSIX termios
// (notably Windows); input there is typically piped or redirected.
func disableCanonicalMode(*os.File) (func(), bool) {
	return nil, false
}
