package bench

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *Session) onOpenUploadHandler(dc *webrtc.DataChannel) func() {
	return func() {
		if !s.master {
			<-s.startPhase2
		}

		log.Debugln("Starting to upload data...")
		defer log.Debugln("Stopped uploading data...")

		lenToken := uint64(4096)
		token := make([]byte, lenToken)
		rand.Read(token)

		s.uploadNetworkStats.Start()

		// Useful for unit tests
		if dc != nil {
			dc.SetBufferedAmountLowThreshold(s.bufferThreshold)
			dc.OnBufferedAmountLow(func() {
				if err := dc.Send(token); err == nil {
					s.uploadNetworkStats.AddBytes(lenToken)
				}
			})
		} else {
			log.Warningln("No DataChannel provided")
		}

		fmt.Printf("Uploading random datas ... (%d s)\n", int(s.testDuration.Seconds()))
		timeout := time.After(s.testDuration)
		timeoutErr := time.After(s.testDurationError)

		if dc != nil {
			dc.Send(token)
		}
	SENDING_LOOP:
		for {
			select {
			case <-timeoutErr:
				log.Error("Time'd out")
				break SENDING_LOOP

			case <-timeout:
				log.Traceln("Done uploading")
				break SENDING_LOOP
			}
		}
		s.uploadNetworkStats.Stop()

		if dc != nil {
			dc.Close()
		}

		if s.master {
			close(s.startPhase2)
		}

		s.wg.Done()
	}
}
