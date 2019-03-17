package session

import (
	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (r *receiver) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Infof("ICE Connection State has changed: %s\n", connectionState.String())
	}
}

func (r *receiver) onMessage() func(msg webrtc.DataChannelMessage) {
	return func(msg webrtc.DataChannelMessage) {
		// Store each message in the message channel
		r.msgChannel <- msg
	}
}

func (r *receiver) onClose() func() {
	return func() {
		close(r.done)
	}
}

func (r *receiver) receiveData() {
	log.Infoln("Starting to receive data...")
	defer log.Infoln("Stopped receiving data...")

	// Consume the message channel, until done
	// Does not stop on error
	for {
		select {
		case <-r.done:
			r.networkStats.Stop()
			log.Infof("Stats: %s\n", r.networkStats.String())
			return
		case msg := <-r.msgChannel:
			n, err := r.stream.Write(msg.Data)

			if err != nil {
				log.Errorln(err)
			} else {
				r.networkStats.AddBytes(uint64(n))
			}
		}
	}
}
