package bench

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

const (
	bufferThreshold   = 64 * 1024 // 64kB
	testDuration      = 20 * time.Second
	testDurationError = (testDuration * 10) / 7
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
		dc.SetBufferedAmountLowThreshold(bufferThreshold)
		dc.OnBufferedAmountLow(func() {
			if err := dc.Send(token); err == nil {
				s.uploadNetworkStats.AddBytes(lenToken)
			}
		})

		fmt.Printf("Uploading random datas ... (%d s)\n", int(testDuration.Seconds()))
		timeout := time.After(testDuration)
		timeoutErr := time.After(testDurationError)
		dc.Send(token)
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
		dc.Close()

		if s.master {
			close(s.startPhase2)
		}

		s.wg.Done()
	}
}
