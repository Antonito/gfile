package session

import (
	"io"
	"sync"
	"time"

	"github.com/pions/webrtc"
)

// Session contains informations about a Send Session
type Session struct {
	stream         io.Reader
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
	dataBuff       []byte

	// Control
	done        chan struct{}
	stopSending chan struct{}

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
		dataBuff:    make([]byte, 4096*2),
		done:        make(chan struct{}),
		stopSending: make(chan struct{}),
		nbBytesRead: 0,
		nbBytesSent: 0,
		doneCheck:   false,
	}
}
