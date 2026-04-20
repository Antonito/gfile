package sender

import (
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

func (s *Session) stopSend() {
	if s.cancelRun != nil {
		s.cancelRun()
	}
}

func (s *Session) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Info().Msgf("ICE Connection State has changed: %s", connectionState.String())
		switch connectionState {
		case webrtc.ICEConnectionStateDisconnected, webrtc.ICEConnectionStateFailed:
			s.stopSend()
		}
	}
}

func (s *Session) sendAbort(reason string) {
	if s.ch == nil {
		return
	}
	_ = s.ch.SendAbort(reason)
	s.ch.Flush(2*time.Second, time.Now)
}

func (s *Session) close(fromWatcher bool) {
	if !fromWatcher && s.ch != nil {
		_ = s.ch.Close()
	}
	closePeers(s.dataPeers)
	s.closeOnce.Do(func() {
		if s.encoder != nil {
			_ = s.encoder.Close()
		}
		s.dumpStats()
		close(s.sess.Done)
	})
}

func (s *Session) dumpStats() {
	log.Info().
		Str("disk", s.readingStats.String()).
		Str("network", s.sess.NetworkStats.String()).
		Msg("session stats")
}
