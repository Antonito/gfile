package sender

import (
	"fmt"
	"io"

	"github.com/pion/webrtc/v3"

	log "github.com/sirupsen/logrus"
)

func (s *Session) readFile() {
	log.Infof("Starting to read data...")
	s.readingStats.Start()
	defer func() {
		s.readingStats.Stop()
		close(s.output)
		log.Infof("Stopped reading data...")
	}()

	for {
		// Read file
		s.dataBuff = s.dataBuff[:cap(s.dataBuff)]

		n, err := s.stream.Read(s.dataBuff)
		if err != nil {
			if err == io.EOF {
				log.Debugf("Got EOF after %v bytes!\n", s.readingStats.Bytes())
				return
			}
			log.Errorf("Read Error: %v\n", err)
			return
		}
		if n == 0 {
			return
		}

		s.dataBuff = s.dataBuff[:n]
		s.readingStats.AddBytes(uint64(n))

		s.output <- outputMsg{
			n: n,
			// Make a copy of the buffer
			buff: append([]byte(nil), s.dataBuff...),
		}
	}
}

func (s *Session) writeToNetwork(dataChannel *webrtc.DataChannel) {
	defer func() {
		currentSpeed := s.sess.NetworkStats.Bandwidth()
		fmt.Printf("Transferred at %.2f MB/s\n", currentSpeed)
	}()

	for {
		select {
		case msg := <-s.output:
			if msg.n == 0 {
				log.Debugf("done writing file\n")
				return
			}

			currentSpeed := s.sess.NetworkStats.Bandwidth()
			fmt.Printf("Transferring at %.2f MB/s\r", currentSpeed)

			if err := dataChannel.Send(msg.buff); err != nil {
				log.Errorf("Error, cannot send to client: %v\n", err)
				return
			}

			s.sess.NetworkStats.AddBytes(uint64(msg.n))
		}
	}
}
