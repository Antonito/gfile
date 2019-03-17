package session

import (
	"io"
	"time"

	"github.com/pions/webrtc"
	log "github.com/sirupsen/logrus"
)

func (s *sender) onConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Infof("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			s.stopSending <- struct{}{}
		}
	}
}

func (s *sender) onOpenHandler() func() {
	return func() {
		s.networkStats.Start()
		s.writeToNetwork()
	}
}

func (s *sender) onCloseHandler() func() {
	return func() {
		s.close(true)
	}
}

func (s *sender) close(calledFromCloseHandler bool) {
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
	close(s.done)
}

func (s *sender) dumpStats() {
	log.Infof(`
Disk   : %s
Network: %s
`, s.readingStats.String(), s.networkStats.String())
}

func (s *sender) readFile() {
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

func (s *sender) writeToNetwork() {
	log.Infof("Starting to send data...")
	defer log.Infof("Stopped sending data...")

	currentTime := time.Now()

	for {
		select {
		case <-s.stopSending:
			s.networkStats.Pause()
			log.Infof("Pausing network I/O... (remaining at least %v packets)\n", len(s.output))
			return
		case data := <-s.output:
			if data.n == 0 {
				// The channel is closed, nothing more to send
				s.networkStats.Stop()
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
				s.networkStats.AddBytes(uint64(cur.n))
				s.msgToBeSent = s.msgToBeSent[1:]
			}
		}
	}
}
