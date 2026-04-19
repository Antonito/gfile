//go:build linux

package stream

import "golang.org/x/sys/unix"

// Linux ioctl names for termios get/set-attributes.
const (
	ioctlReadTermios  = unix.TCGETS
	ioctlWriteTermios = unix.TCSETS
)
