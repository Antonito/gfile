package sender

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

func (s *Session) readFile() {
	log.Infof("Starting to read data...")
	s.readingStats.Start()
	defer func() {
		s.readingStats.Pause()
		log.Infof("Stopped reading data...")
		close(s.output)
	}()

	for {
		// Read file
		s.dataBuff = s.dataBuff[:cap(s.dataBuff)]
		n, err := s.stream.Read(s.dataBuff)
		if err != nil {
			if err == io.EOF {
				s.readingStats.Stop()
				log.Debugf("Got EOF after %v bytes!\n", s.readingStats.Bytes())
				return
			}
			log.Errorf("Read Error: %v\n", err)
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

func (s *Session) writeToNetwork() {
	log.Infof("Starting to send data...")
	defer log.Infof("Stopped sending data...")

	currentTime := time.Now()

	for {
		select {
		case <-s.stopSending:
			s.sess.NetworkStats.Pause()
			log.Infof("Pausing network I/O... (remaining at least %v packets)\n", len(s.output))
			return
		case data := <-s.output:
			if data.n == 0 {
				// The channel is closed, nothing more to send
				s.sess.NetworkStats.Stop()
				s.close(false)
				return
			}

			// Limit upload speed
			triggered := time.Now()
			if triggered.Sub(currentTime) < 1e8*time.Microsecond {
				time.Sleep((1 * time.Microsecond) - (triggered.Sub(currentTime)))
			}
			currentTime = triggered

			s.msgToBeSent = append(s.msgToBeSent, data)

			for len(s.msgToBeSent) != 0 {
				cur := s.msgToBeSent[0]
				// Writing packet
				if err := s.dataChannel.Send(cur.buff); err != nil {
					log.Errorf("Error, cannot send to client: %v\n", err)
					return
				}
				s.sess.NetworkStats.AddBytes(uint64(cur.n))
				s.msgToBeSent = s.msgToBeSent[1:]
			}
		}
	}
}
