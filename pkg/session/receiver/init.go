package receiver

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
	s.createDataHandler()
	if err := s.sess.ReadSDP(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := s.sess.CreateAnswer(); err != nil {
		log.Errorln(err)
		return err
	}

	// Handle data
	s.receiveData()
	s.sess.OnCompletion()
	return nil
}

func (s *Session) createDataHandler() {
	s.sess.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Debugf("New DataChannel %s %d\n", d.Label, d.ID)
		s.sess.NetworkStats.Start()
		d.OnMessage(s.onMessage())
		d.OnClose(s.onClose())
	})
}

func (s *Session) receiveData() {
	log.Infoln("Starting to receive data...")
	defer log.Infoln("Stopped receiving data...")

	// Consume the message channel, until done
	// Does not stop on error
	for {
		select {
		case <-s.sess.Done:
			s.sess.NetworkStats.Stop()
			return
		case msg := <-s.msgChannel:
			n, err := s.stream.Write(msg.Data)

			if err != nil {
				log.Errorln(err)
			} else {
				s.sess.NetworkStats.AddBytes(uint64(n))
			}
		}
	}
}
