package session

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Antonito/gfile/pkg/utils"
	"github.com/pions/webrtc"
)

type packetBuffer struct {
	buf []byte
	n   int
}

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

// Connect starts a connection and waits till it ends
func (s *Session) Connect() error {
	if err := s.createConnection(); err != nil {
		return err
	}
	if err := s.createDataChannel(); err != nil {
		return err
	}

	sdpChan := utils.HTTPSDPServer()

	if err := s.createOffer(); err != nil {
		return err
	}

	// Wait for the answer to be pasted
	fmt.Println(`Please, provide the SDP via:
curl localhost:8080/sdp --data "$SDP"`)
	answer := webrtc.SessionDescription{}
	for {
		if err := utils.Decode(<-sdpChan, &answer); err == nil {
			break
		}
		fmt.Println("Invalid SDP, try aagain...")
	}

	// Apply the answer as the remote description
	if err := s.peerConnection.SetRemoteDescription(answer); err != nil {
		return err
	}

	<-s.done
	return nil
}

func (s *Session) createConnection() error {
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
	s.setStateManager()

	return nil
}

func (s *Session) createOffer() error {
	// Create an offer to send to the browser
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err := s.peerConnection.SetLocalDescription(offer); err != nil {
		return err
	}

	// Output the offer in base64 so we can paste it in browser
	encoded, err := utils.Encode(offer)
	if err != nil {
		return err
	}
	fmt.Println(encoded)
	return nil
}

func (s *Session) createDataChannel() error {
	ordered := true
	maxPacketLifeTime := uint16(5000)
	dataChannel, err := s.peerConnection.CreateDataChannel("data", &webrtc.DataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	})
	if err != nil {
		return err
	}
	s.dataChannel = dataChannel
	s.dataChannel.OnOpen(s.onOpenHandler())
	s.dataChannel.OnClose(s.onCloseHandler())
	return nil
}
