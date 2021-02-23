package receiver

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

// Start initializes the connection and the file transfer
func (s *Session) Start() error {
	if err := s.sess.Start(); err != nil {
		return err
	}

	// Handle data
	s.receiveData()
	return nil
}

func (s *Session) receiveData() {
	log.Infoln("Starting to receive data...")
	defer func() {
		s.sess.NetworkStats.Stop()

		log.Infoln("Stopped receiving data...")
		fmt.Printf("\nNetwork: %s\n", s.sess.NetworkStats.String())

		s.sess.Close()
	}()

	// Consume the message channel, until done
	// Does not stop on error
	for {
		select {
		case <-time.After(5 * time.Second):
			return
		case <-s.sess.Done:
			return
		case msg := <-s.msgChannel:
			n, err := s.stream.Write(msg.Data)

			if err != nil {
				log.Errorln(err)
			} else {
				currentSpeed := s.sess.NetworkStats.Bandwidth()
				fmt.Printf("Transferring at %.2f MB/s\r", currentSpeed)
				s.sess.NetworkStats.AddBytes(uint64(n))
			}
		}
	}
}

