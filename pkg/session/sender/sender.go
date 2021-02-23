package sender

import (
	"github.com/antonito/gfile/internal/session/rtc"
	"io"
	"sync"

	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/pkg/session/common"
	"github.com/antonito/gfile/pkg/stats"
	"github.com/pion/webrtc/v3"
)

const (
	// Must be <= 16384
	defaultSenderBuffSize = 16384
)

type outputMsg struct {
	n    int
	buff []byte
}

// Session is a sender session
type Session struct {
	sess        *internalSess.Session
	stream      io.Reader
	initialized bool

	dataBuff    []byte
	msgToBeSent []outputMsg
	output      chan outputMsg

	doneCheckLock sync.Mutex
	doneCheck     bool

	// Stats/infos
	readingStats *stats.Stats
}

// New creates a new sender session
func new(sdpIO internalSess.SDPIO, f io.Reader, senderBuffSize int, stun string) *Session {
	if senderBuffSize > defaultSenderBuffSize {
		panic("bufferSize must be <= 16384")
	}

	sess := &Session{
		sess:         nil,
		stream:       f,
		initialized:  false,
		dataBuff:     make([]byte, senderBuffSize),
		output:       make(chan outputMsg, senderBuffSize*10),
		doneCheck:    false,
		readingStats: stats.New(),
	}

	ordered := true
	maxPacketLifeTime := uint16(10000)
	bufferThresholdCpy := uint64(bufferThreshold)

	dataChannelCfg := rtc.DataChannelConfiguration{
		InitParams: &webrtc.DataChannelInit{
			Ordered:           &ordered,
			MaxPacketLifeTime: &maxPacketLifeTime,
		},
		OnOpen: sess.onOpenHandler(),
		OnClose: sess.onCloseHandler(),
		BufferThreshold: &bufferThresholdCpy,
	}

	sess.sess = internalSess.New(internalSess.KindMaster, sdpIO, stun, dataChannelCfg)

	return sess
}

// New creates a new receiver session
func New(f io.Reader) *Session {
	sdpIO := internalSess.SDPIO{
		Input: nil,
		Output: nil,
	}

	return new(sdpIO, f, defaultSenderBuffSize, "")
}

// Config contains custom configuration for a session
type Config struct {
	common.Configuration
	Stream io.Reader // The Stream to read from
}

// NewWith creates a new sender Session with custom configuration
func NewWith(c Config) *Session {
	sdpIO := internalSess.SDPIO{
		Input: c.SDPProvider,
		Output: c.SDPOutput,
	}

	return new(sdpIO, c.Stream, defaultSenderBuffSize, c.STUN)
}

// SetStream changes the stream, useful for WASM integration
func (s *Session) SetStream(stream io.Reader) {
	s.stream = stream
}
