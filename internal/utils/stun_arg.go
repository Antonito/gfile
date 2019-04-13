package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSTUN checks if a STUN addr is valid
func ParseSTUN(stunAddr string) error {
	arr := strings.Split(stunAddr, ":")
	if len(arr) != 2 {
		return fmt.Errorf("invalid stun adress")
	}
	port, err := strconv.Atoi(arr[1])
	if err != nil || (port <= 0 || port > 0xffff) {
		return fmt.Errorf("invalid port %v", port)
	}
	return nil
}
