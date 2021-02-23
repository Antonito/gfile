package session

import (
	"fmt"
	"io"
	"os"

	"github.com/antonito/gfile/internal/session/rtc"
	"github.com/antonito/gfile/pkg/stats"
	"github.com/pion/webrtc/v3"
)

// CompletionHandler to be called when transfer is done
type CompletionHandler func()

type SDPIO struct {
	Input  io.Reader
	Output io.Writer
}

// Session contains common elements to perform send/receive
type Session struct {
	kind  Kind
	sdpIO SDPIO

	Done         chan struct{}
	NetworkStats *stats.Stats

	rtcClient *rtc.Client

	stunServers []string
}

// New creates a new Session
func New(kind Kind, sdpIO SDPIO, customSTUN string, dataChannelConfiguration rtc.DataChannelConfiguration) *Session {
	sess := Session{
		kind:         kind,
		sdpIO:        sdpIO,
		Done:         make(chan struct{}),
		NetworkStats: stats.New(),
		stunServers:  []string{fmt.Sprintf("stun:%s", customSTUN)},
	}

	if sdpIO.Input == nil {
		sess.sdpIO.Input = os.Stdin
	}
	if sdpIO.Output == nil {
		sess.sdpIO.Output = os.Stdout
	}
	if customSTUN == "" {
		sess.stunServers = []string{"stun:stun.l.google.com:19302"}
	}

	sess.rtcClient = rtc.NewClient(rtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: sess.stunServers,
			},
		},
		DataChannel: dataChannelConfiguration,
	})

	return &sess
}

/// Start a session, according to its kind.
func (s *Session) Start() error {
	switch s.kind {
	case KindMaster:
		return s.startMasterNode()
	case KindNode:
		return s.startNode()
	default:
		// This statement can never be reached
		panic("not possible")
	}
}

/// Close the session.
func (s *Session) Close() {
	s.rtcClient.Close()
}

func (s *Session) startMasterNode() error {
	localOffer, err := s.rtcClient.MakeLocalOffer()
	if err != nil {
		return err
	}

	s.printSDPToOutput(*localOffer)

	remoteAnswer := s.readSDPFromInput()

	if err := s.rtcClient.SetAnswer(remoteAnswer); err != nil {
		return err
	}

	return nil
}

func (s *Session) startNode() error {
	remoteOffer := s.readSDPFromInput()

	if err := s.rtcClient.SetRemoteOffer(remoteOffer); err != nil {
		return err
	}

	localAnswer, err := s.rtcClient.MakeAnswer()
	if err != nil {
		return err
	}

	s.printSDPToOutput(*localAnswer)

	return nil
}
