package session

import (
	"io"
	"os"
	"sync"

	"github.com/antonito/gfile/pkg/stats"
	"github.com/pions/webrtc"
)

const (
	// Must be <= 16384
	buffSize = 16384
)

type outputMsg struct {
	n    int
	buff []byte
}

// Session contains informations about a Send Session
type Session struct {
	stream         io.Reader
	sdpInput       io.Reader
	sdpOutput      io.Writer
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
	dataBuff       []byte
	msgToBeSent    []outputMsg

	// Control
	done        chan struct{}
	stopSending chan struct{}
	output      chan outputMsg

	doneCheckLock sync.Mutex
	doneCheck     bool

	// Stats/infos
	readingStats stats.Stats
	networkStats stats.Stats
}

// Config contains custom configuration for a session
type Config struct {
	Stream      io.Reader // The Stream to read from
	SDPProvider io.Reader // The SDP reader
	SDPOutput   io.Writer // The SDP writer
}

// NewSession creates a new Session
func NewSession(f io.Reader) *Session {
	return &Session{
		stream:      f,
		sdpInput:    os.Stdin,
		sdpOutput:   os.Stdout,
		dataBuff:    make([]byte, buffSize),
		done:        make(chan struct{}),
		stopSending: make(chan struct{}, 1),
		output:      make(chan outputMsg, buffSize*10),
		doneCheck:   false,
	}
}

// NewSessionWith createa a new Session with custom configuration
func NewSessionWith(c Config) *Session {
	session := NewSession(c.Stream)
	session.sdpInput = c.SDPProvider
	session.sdpOutput = c.SDPOutput
	return session
}
