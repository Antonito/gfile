package bench

import (
	"fmt"

	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *Session) onNewDataChannel() func(d *webrtc.DataChannel) {
	return func(d *webrtc.DataChannel) {
		if d == nil || d.ID() == nil {
			return
		}
		channelID := *d.ID()

		log.Tracef("New DataChannel %s (id: %x)\n", d.Label(), channelID)

		if channelID == s.downloadChannelID() {
			log.Traceln("Created Download data channel")
			d.OnClose(s.onCloseHandlerDownload())
			go s.onOpenHandlerDownload(d)()
		} else if channelID == s.uploadChannelID() {
			log.Traceln("Created Upload data channel")
		} else {
			log.Warningln("Created unknown data channel")
		}
	}
}

func (s *Session) createMasterSession() error {
	if err := s.sess.CreateOffer(); err != nil {
		log.Errorln(err)
		return err
	}

	fmt.Println("Please, paste the remote SDP:")
	if err := s.sess.ReadSDP(); err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func (s *Session) createSlaveSession() error {
	fmt.Println("Please, paste the remote SDP:")
	if err := s.sess.ReadSDP(); err != nil {
		log.Errorln(err)
		return err
	}

	fmt.Println("SDP:")
	if err := s.sess.CreateAnswer(); err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}
