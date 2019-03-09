package session

import (
	"io"

	"github.com/pions/webrtc"
)

// Session contains informations about a Receiver Session
type Session struct {
	stream         io.Writer
	peerConnection *webrtc.PeerConnection

	msgChannel chan webrtc.DataChannelMessage
	done       chan struct{}
}

// NewSession returns a new Receiver Session
func NewSession(f io.Writer) *Session {
	return &Session{
		stream:     f,
		msgChannel: make(chan webrtc.DataChannelMessage, 4096*2),
		done:       make(chan struct{}),
	}
}
