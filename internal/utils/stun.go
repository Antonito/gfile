package utils

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

// ParseSTUN validates a STUN address of the form "host:port"
func ParseSTUN(stunAddr string) error {
	_, portStr, err := net.SplitHostPort(stunAddr)
	if err != nil {
		return errors.New("invalid stun address")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || (port <= 0 || port > 0xffff) {
		return fmt.Errorf("invalid port %v", port)
	}

	return nil
}
