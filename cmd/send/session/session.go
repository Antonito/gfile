package session

import (
	"io"
	"os"
	"sync"
	"time"

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
	nbBytesRead uint64
	nbBytesSent uint64
	timeStart   time.Time
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
		nbBytesRead: 0,
		nbBytesSent: 0,
		doneCheck:   false,
	}
}
