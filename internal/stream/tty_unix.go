//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package stream

import (
	"os"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// disableCanonicalMode puts f into non-canonical input mode for the
// duration of a single read, returning a restore closure. Canonical mode
// caps a line at MAX_CANON bytes (1024 on macOS); pasted SDPs routinely
// exceed that and are silently truncated before reaching us.
//
// Returns (nil, false) when f is not a TTY or the ioctl fails.
func disableCanonicalMode(file *os.File) (restore func(), ok bool) {
	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		return nil, false
	}
	orig, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, false
	}
	modified := *orig
	modified.Lflag &^= unix.ICANON
	modified.Cc[unix.VMIN] = 1
	modified.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &modified); err != nil {
		return nil, false
	}
	return func() { _ = unix.IoctlSetTermios(fd, ioctlWriteTermios, orig) }, true
}
