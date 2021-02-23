package receiver

import (
	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/internal/session/rtc"
	"github.com/antonito/gfile/pkg/session/common"
	"github.com/pion/webrtc/v3"
	"io"
)

// Session is a receiver session
type Session struct {
	sess        *internalSess.Session
	stream      io.Writer
	msgChannel  chan webrtc.DataChannelMessage
	initialized bool
}

func new(sdpIO internalSess.SDPIO, f io.Writer, stun string) *Session {
	sess := &Session{
		sess:        nil,
		stream:      f,
		msgChannel:  make(chan webrtc.DataChannelMessage, 4096*2),
		initialized: false,
	}

	dataChannelCfg := rtc.DataChannelConfiguration{
		InitParams: nil,
		OnOpen: sess.onOpen(),
		OnMessage: sess.onMessage(),
		OnClose: sess.onClose(),
	}

	sess.sess = internalSess.New(internalSess.KindNode, sdpIO, stun, dataChannelCfg)

	return sess
}

// New creates a new receiver session
func New(f io.Writer) *Session {
	sdpIO := internalSess.SDPIO{
		Input: nil,
		Output: nil,
	}

	return new(sdpIO, f, "")
}

// Config contains custom configuration for a session
type Config struct {
	common.Configuration
	Stream io.Writer // The Stream to write to
}

// NewWith creates a new receiver Session with custom configuration
func NewWith(c Config) *Session {
	sdpIO := internalSess.SDPIO{
		Input: c.SDPProvider,
		Output: c.SDPOutput,
	}

	return new(sdpIO, c.Stream, c.STUN)
}

// SetStream changes the stream, useful for WASM integration
func (s *Session) SetStream(stream io.Writer) {
	s.stream = stream
}

