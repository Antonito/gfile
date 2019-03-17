package session

import (
	"io"
	"os"

	"github.com/antonito/gfile/pkg/stats"
	"github.com/pions/webrtc"
)

// Config contains custom configuration for a session
type Config struct {
	Stream      io.Writer // The Stream to write to
	SDPProvider io.Reader // The SDP reader
	SDPOutput   io.Writer // The SDP writer
}

// Session contains informations about a Receiver Session
type Session struct {
	stream         io.Writer
	sdpInput       io.Reader
	sdpOutput      io.Writer
	peerConnection *webrtc.PeerConnection

	msgChannel chan webrtc.DataChannelMessage
	done       chan struct{}

	networkStats stats.Stats
}

// NewSession returns a new Receiver Session
func NewSession(f io.Writer) *Session {
	return &Session{
		stream:     f,
		sdpInput:   os.Stdin,
		sdpOutput:  os.Stdout,
		msgChannel: make(chan webrtc.DataChannelMessage, 4096*2),
		done:       make(chan struct{}),
	}
}

// NewSessionWith createa a new Session with custom configuration
func NewSessionWith(c Config) *Session {
	session := NewSession(c.Stream)
	session.sdpInput = c.SDPProvider
	session.sdpOutput = c.SDPOutput
	return session
}
