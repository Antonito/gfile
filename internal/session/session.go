package session

import (
	"errors"
	"fmt"
	"net"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"

	"github.com/antonito/gfile/internal/stats"
	"github.com/antonito/gfile/internal/utils"
)

// Session wraps a pion PeerConnection with lifecycle state.
type Session struct {
	Done           chan struct{}
	NetworkStats   *stats.Stats
	peerConnection *webrtc.PeerConnection
	cfg            Config
	detach         bool
}

// ErrNoLocalDescription is returned when ICE gathering completes with no local description.
var ErrNoLocalDescription = errors.New("local description nil after ICE gathering")

// New creates a sender-side Session from cfg.
func New(cfg Config) Session {
	return Session{
		Done:         make(chan struct{}),
		NetworkStats: stats.New(),
		cfg:          cfg,
	}
}

// NewReceiver creates a Session with pion's DetachDataChannels enabled.
// Callers must invoke DataChannel.Detach() in OnOpen and drive their own
// Read loop — OnMessage never fires once detach is on.
func NewReceiver(cfg Config) Session {
	s := New(cfg)
	s.detach = true
	return s
}

// IsLoopbackOnly reports whether the session was configured with LoopbackOnly.
func (s *Session) IsLoopbackOnly() bool {
	return s.cfg.LoopbackOnly
}

// CreateConnection prepares a WebRTC connection.
func (s *Session) CreateConnection(
	onConnectionStateChange func(connectionState webrtc.ICEConnectionState),
) error {
	config := webrtc.Configuration{}
	if len(s.cfg.STUNServers) > 0 {
		config.ICEServers = []webrtc.ICEServer{
			{URLs: s.cfg.STUNServers},
		}
	}

	se := webrtc.SettingEngine{}
	if s.detach {
		se.DetachDataChannels()
	}

	// SCTP CRC32C is redundant under DTLS (RFC 9653); skipping it removes a
	// CPU bottleneck on fast paths. Negotiated — falls back if the peer rejects.
	se.EnableSCTPZeroChecksum(true)

	// Pion skips lo0 by default; opt it in so same-host transfers can pick loopback.
	se.SetIncludeLoopbackCandidate(true)

	// mDNS advertises host candidates as `.local` names instead of raw IPs.
	// Skip under LoopbackOnly — lo0 candidates don't need name resolution.
	if !s.cfg.LoopbackOnly && !s.cfg.DisableMDNS {
		se.SetICEMulticastDNSMode(ice.MulticastDNSModeQueryAndGather)
	}

	if s.cfg.LoopbackOnly {
		// Force loopback: filter non-loopback interfaces and drop STUN so ICE
		// can't nominate the physical path or block on a failing gather.
		se.SetInterfaceFilter(func(name string) bool {
			iface, err := net.InterfaceByName(name)
			if err != nil {
				return false
			}
			return iface.Flags&net.FlagLoopback != 0
		})
		se.SetNetworkTypes([]webrtc.NetworkType{
			webrtc.NetworkTypeUDP4,
			webrtc.NetworkTypeUDP6,
		})
		config.ICEServers = nil
	}

	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return err
	}
	s.peerConnection = peerConnection
	peerConnection.OnICEConnectionStateChange(onConnectionStateChange)

	return nil
}

// CreateChannel creates an outgoing DataChannel wrapped as a *Channel.
func (s *Session) CreateChannel(label string) (*Channel, error) {
	dc, err := s.peerConnection.CreateDataChannel(label, nil)
	if err != nil {
		return nil, err
	}

	return newChannel(dc, s.detach), nil
}

// OnChannel registers a handler invoked for every incoming DataChannel.
func (s *Session) OnChannel(handler func(*Channel)) {
	s.peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		handler(newChannel(dc, s.detach))
	})
}

// MakeOffer creates the offer, sets it locally, and waits for ICE gathering.
func (s *Session) MakeOffer() (string, error) {
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	if err := s.peerConnection.SetLocalDescription(offer); err != nil {
		return "", err
	}

	<-webrtc.GatheringCompletePromise(s.peerConnection)

	desc := s.peerConnection.LocalDescription()
	if desc == nil {
		return "", ErrNoLocalDescription
	}

	return utils.EncodeSDP(*desc)
}

// AcceptOffer sets the remote offer, creates an answer, and waits for ICE gathering.
func (s *Session) AcceptOffer(encodedOffer string) (string, error) {
	offer, err := utils.DecodeSDP(encodedOffer)
	if err != nil {
		return "", fmt.Errorf("decode offer: %w", err)
	}
	if err := s.peerConnection.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("set remote: %w", err)
	}

	answer, err := s.peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	if err := s.peerConnection.SetLocalDescription(answer); err != nil {
		return "", err
	}

	<-webrtc.GatheringCompletePromise(s.peerConnection)

	desc := s.peerConnection.LocalDescription()
	if desc == nil {
		return "", ErrNoLocalDescription
	}
	return utils.EncodeSDP(*desc)
}

// AcceptAnswer sets the remote answer on a PeerConnection whose offer was already sent.
func (s *Session) AcceptAnswer(encodedAnswer string) error {
	answer, err := utils.DecodeSDP(encodedAnswer)
	if err != nil {
		return fmt.Errorf("decode answer: %w", err)
	}
	return s.peerConnection.SetRemoteDescription(answer)
}

// Close releases the PeerConnection. Safe before CreateConnection and idempotent.
func (s *Session) Close() error {
	if s.peerConnection == nil {
		return nil
	}

	return s.peerConnection.Close()
}
