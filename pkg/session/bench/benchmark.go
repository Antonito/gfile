package bench

import (
	"sync"

	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/pkg/session/common"
	"github.com/antonito/gfile/pkg/stats"
	"github.com/pions/webrtc"
)

// Session is a benchmark session
type Session struct {
	sess   internalSess.Session
	master bool
	wg     sync.WaitGroup

	startPhase2          chan struct{}
	uploadDataChannel    *webrtc.DataChannel
	uploadNetworkStats   stats.Stats
	downloadDone         chan bool
	downloadNetworkStats stats.Stats
}

// New creates a new sender session
func new(s internalSess.Session, isMaster bool) *Session {
	return &Session{
		sess:   s,
		master: isMaster,

		startPhase2:  make(chan struct{}),
		downloadDone: make(chan bool),
	}
}

// Config contains custom configuration for a session
type Config struct {
	common.Configuration
	Master bool // Will create the SDP offer ?
}

// NewWith createa a new benchmark Session with custom configuration
func NewWith(c Config) *Session {
	return new(internalSess.New(c.SDPProvider, c.SDPOutput), c.Master)
}
