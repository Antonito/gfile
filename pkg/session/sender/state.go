package sender

import (
	"fmt"

	"github.com/pion/webrtc/v3"
	log "github.com/sirupsen/logrus"
)

func (s *Session) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Infof("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			// TODO: Implement retry mechanism
			panic("lost connection")
		}
	}
}

func (s *Session) onOpenHandler() func(*webrtc.DataChannel) {
	return func(dataChannel *webrtc.DataChannel) {
		s.sess.NetworkStats.Start()

		log.Infof("Starting to send data...")
		defer log.Infof("Stopped sending data...")

		s.writeToNetwork(dataChannel)
	}
}

func (s *Session) onCloseHandler() func() {
	return func() {
		s.close()
	}
}

func (s *Session) close() {
	// Sometime, onCloseHandler is not invoked, so it's a work-around
	s.doneCheckLock.Lock()
	if s.doneCheck {
		s.doneCheckLock.Unlock()
		return
	}
	s.doneCheck = true
	s.doneCheckLock.Unlock()
	s.dumpStats()
	close(s.sess.Done)

	// Done writing
	s.sess.Close()
}

func (s *Session) dumpStats() {
	fmt.Printf(`
Disk   : %s
Network: %s
`, s.readingStats.String(), s.sess.NetworkStats.String())
}
