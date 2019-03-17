package session

import (
	"io"
	"os"
	"sync"

	"github.com/antonito/gfile/pkg/stats"
	"github.com/pions/webrtc"
)

// SenderConfig contains custom configuration for a session
type SenderConfig struct {
	Stream      io.Reader // The Stream to write to
	SDPProvider io.Reader // The SDP reader
	SDPOutput   io.Writer // The SDP writer
}

const (
	// Must be <= 16384
	senderBuffSize = 16384
)

type outputMsg struct {
	n    int
	buff []byte
}

type sender struct {
	session
	stream io.Reader

	dataChannel *webrtc.DataChannel
	dataBuff    []byte
	msgToBeSent []outputMsg
	stopSending chan struct{}
	output      chan outputMsg

	doneCheckLock sync.Mutex
	doneCheck     bool

	// Stats/infos
	readingStats stats.Stats
	networkStats stats.Stats
}

func newSender(f io.Reader) *sender {
	return &sender{
		session: session{
			sdpInput:  os.Stdin,
			sdpOutput: os.Stdout,
			done:      make(chan struct{}),
		},
		stream:      f,
		dataBuff:    make([]byte, senderBuffSize),
		stopSending: make(chan struct{}, 1),
		output:      make(chan outputMsg, senderBuffSize*10),
		doneCheck:   false,
	}
}

// NewSenderWith createa a new sender Session with custom configuration
func NewSenderWith(c SenderConfig) Session {
	session := newSender(c.Stream)
	if c.SDPProvider != nil {
		session.sdpInput = c.SDPProvider
	}
	if c.SDPOutput != nil {
		session.sdpOutput = c.SDPOutput
	}
	return session
}
