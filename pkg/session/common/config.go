package common

import (
	"io"
)

// Configuration common to both Sender and Receiver session
type Configuration struct {
	SDPProvider io.Reader // The SDP reader
	SDPOutput   io.Writer // The SDP writer
	STUN        string    // Custom STUN server
}
