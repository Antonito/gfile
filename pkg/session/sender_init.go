package session

import (
	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *sender) createDataChannel() error {
	ordered := true
	maxPacketLifeTime := uint16(10000)
	dataChannel, err := s.peerConnection.CreateDataChannel("data", &webrtc.DataChannelInit{
		Ordered:           &ordered,
		MaxPacketLifeTime: &maxPacketLifeTime,
	})
	if err != nil {
		return err
	}
	go s.readFile()
	s.dataChannel = dataChannel
	s.dataChannel.OnOpen(s.onOpenHandler())
	s.dataChannel.OnClose(s.onCloseHandler())
	return nil
}

func (s *sender) createOffer() error {
	// Create an offer
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return err
	}
	return s.createSessionDescription(offer)
}

func (s *sender) Connect() error {
	if err := s.createConnection(s.onConnectionStateChange()); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.createDataChannel(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.createOffer(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.readSDP(); err != nil {
		log.Errorln(err)
		return err
	}

	<-s.done
	log.Infoln("Transfer done")
	return nil
}
