package bench

import (
	"fmt"
	"time"

	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *Session) onOpenHandlerDownload(dc *webrtc.DataChannel) func() {
	// If master, wait for the upload to complete
	// If not master, close the channel so the  upload can start
	return func() {
		if s.master {
			<-s.startPhase2
		}

		log.Debugf("Starting to download data...")
		defer log.Debugf("Stopped downloading data...")

		s.downloadNetworkStats.Start()

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			s.downloadNetworkStats.AddBytes(uint64(len(msg.Data)))
		})

		timeoutErr := time.After(testDurationError)

		fmt.Printf("Downloading random datas ... (%d s)\n", int(testDuration.Seconds()))
	DOWNLOAD_LOOP:
		for {
			select {
			case <-timeoutErr:
				log.Error("Download time'd out")
				break DOWNLOAD_LOOP

			case <-s.downloadDone:
				log.Traceln("Done downloading")
				break DOWNLOAD_LOOP
			}
		}

		if !s.master {
			close(s.startPhase2)
		}

		s.downloadNetworkStats.Stop()
		s.wg.Done()
	}
}

func (s *Session) onCloseHandlerDownload() func() {
	return func() {
		close(s.downloadDone)
	}
}
