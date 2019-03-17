package session

import (
	"fmt"
	"io"

	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/stats"
	"github.com/pions/webrtc"
)

// Session defines a common interface for sender and receiver sessions
type Session interface {
	Connect() error
}

type session struct {
	sdpInput       io.Reader
	sdpOutput      io.Writer
	peerConnection *webrtc.PeerConnection
	done           chan struct{}

	networkStats stats.Stats //nolint
}

func (s *session) createConnection(onConnectionStateChange func(connectionState webrtc.ICEConnectionState)) error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}
	s.peerConnection = peerConnection
	peerConnection.OnICEConnectionStateChange(onConnectionStateChange)

	return nil
}

func (s *session) readSDP() error {
	var sdp webrtc.SessionDescription
	for {
		encoded, err := utils.MustReadStream(s.sdpInput)
		if err == nil {
			if err := utils.Decode(encoded, &sdp); err == nil {
				break
			}
		}
		fmt.Println("Invalid SDP, try again...")
	}
	return s.peerConnection.SetRemoteDescription(sdp)
}

func (s *session) createSessionDescription(desc webrtc.SessionDescription) error {
	// Sets the LocalDescription, and starts our UDP listeners
	if err := s.peerConnection.SetLocalDescription(desc); err != nil {
		return err
	}
	desc.SDP = utils.StripSDP(desc.SDP)

	// Output the SDP in base64 so we can paste it in browser
	resp, err := utils.Encode(desc)
	if err != nil {
		return err
	}
	fmt.Fprintln(s.sdpOutput, resp)
	return nil
}
