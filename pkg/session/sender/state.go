package sender

import (
	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *Session) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Infof("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			s.stopSending <- struct{}{}
		}
	}
}

func (s *Session) onOpenHandler() func() {
	return func() {
		s.sess.NetworkStats.Start()
		s.writeToNetwork()
	}
}

func (s *Session) onCloseHandler() func() {
	return func() {
		s.close(true)
	}
}

func (s *Session) close(calledFromCloseHandler bool) {
	if !calledFromCloseHandler {
		s.dataChannel.Close()
	}

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
}

func (s *Session) dumpStats() {
	log.Infof(`
Disk   : %s
Network: %s
`, s.readingStats.String(), s.sess.NetworkStats.String())
}
