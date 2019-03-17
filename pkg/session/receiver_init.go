package session

import (
	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (r *receiver) createDataHandler() {
	r.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Debugf("New DataChannel %s %d\n", d.Label, d.ID)
		r.networkStats.Start()
		d.OnMessage(r.onMessage())
		d.OnClose(r.onClose())
	})
}

func (r *receiver) createAnswer() error {
	// Create an answer
	answer, err := r.peerConnection.CreateAnswer(nil)
	if err != nil {
		return err
	}
	return r.createSessionDescription(answer)
}

func (r *receiver) Connect() error {
	if err := r.createConnection(r.onConnectionStateChange()); err != nil {
		log.Errorln(err)
		return err
	}
	r.createDataHandler()
	if err := r.readSDP(); err != nil {
		log.Errorln(err)
		return err
	}
	if err := r.createAnswer(); err != nil {
		log.Errorln(err)
		return err
	}

	// Handle data
	r.receiveData()
	return nil
}
