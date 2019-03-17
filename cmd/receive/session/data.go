package session

import (
	log "github.com/sirupsen/logrus"

	"github.com/pions/webrtc"
)

func (s *Session) receiveData() {
	log.Infoln("Starting to receive data...")
	defer log.Infoln("Stopped receiving data...")

	// Consume the message channel, until done
	// Does not stop on error
	for {
		select {
		case <-s.done:
			s.networkStats.Stop()
			log.Infof("Stats: %s\n", s.networkStats.String())
			return
		case msg := <-s.msgChannel:
			n, err := s.stream.Write(msg.Data)

			if err != nil {
				log.Errorln(err)
			} else {
				s.networkStats.AddBytes(uint64(n))
			}
		}
	}
}

func (s *Session) onMessage() func(msg webrtc.DataChannelMessage) {
	return func(msg webrtc.DataChannelMessage) {
		// Store each message in the message channel
		s.msgChannel <- msg
	}
}

func (s *Session) onClose() func() {
	return func() {
		close(s.done)
	}
}
