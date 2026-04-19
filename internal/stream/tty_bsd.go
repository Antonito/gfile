//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package stream

import "golang.org/x/sys/unix"

// BSD-family (incl. darwin) ioctl names for termios get/set-attributes.
const (
	ioctlReadTermios  = unix.TIOCGETA
	ioctlWriteTermios = unix.TIOCSETA
)
