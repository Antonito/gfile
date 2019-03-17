package session

import (
	"io"
	"os"

	"github.com/pions/webrtc"
)

// ReceiverConfig contains custom configuration for a session
type ReceiverConfig struct {
	Stream      io.Writer // The Stream to write to
	SDPProvider io.Reader // The SDP reader
	SDPOutput   io.Writer // The SDP writer
}

type receiver struct {
	session
	stream     io.Writer
	msgChannel chan webrtc.DataChannelMessage
}

func newReceiver(f io.Writer) *receiver {
	return &receiver{
		session: session{
			sdpInput:  os.Stdin,
			sdpOutput: os.Stdout,
			done:      make(chan struct{}),
		},
		stream:     f,
		msgChannel: make(chan webrtc.DataChannelMessage, 4096*2),
	}
}

// NewReceiverWith createa a new receiver Session with custom configuration
func NewReceiverWith(c ReceiverConfig) Session {
	session := newReceiver(c.Stream)
	if c.SDPProvider != nil {
		session.sdpInput = c.SDPProvider
	}
	if c.SDPOutput != nil {
		session.sdpOutput = c.SDPOutput
	}
	return session
}
