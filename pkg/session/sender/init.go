package sender

import (
	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

// Start initializes the connection and the file transfer
func (s *Session) Start() error {
	if err := s.sess.CreateConnection(s.onConnectionStateChange()); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.createDataChannel(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.sess.CreateOffer(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.sess.ReadSDP(); err != nil {
		log.Errorln(err)
		return err
	}
	<-s.sess.Done
	s.sess.OnCompletion()
	return nil
}

func (s *Session) createDataChannel() error {
	ordered := true
	maxPacketLifeTime := uint16(10000)
	dataChannel, err := s.sess.CreateDataChannel(&webrtc.DataChannelInit{
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
