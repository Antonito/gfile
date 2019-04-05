package bench

import (
	"fmt"

	"github.com/pion/webrtc"
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

		// Useful for unit tests
		if dc != nil {
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				s.downloadNetworkStats.AddBytes(uint64(len(msg.Data)))
			})
		} else {
			log.Warningln("No DataChannel provided")
		}

		fmt.Printf("Downloading random datas ... (%d s)\n", int(s.testDuration.Seconds()))
	DOWNLOAD_LOOP:
		for {
			select {
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
