package session

import (
	"fmt"

	"github.com/pions/webrtc"
)

func (s *Session) receiveData() {
	fmt.Println("Starting to receive data...")
	defer fmt.Println("Stopped receiving data...")

	// Consume the message channel, until done
	// Does not stop on error
	for {
		select {
		case <-s.done:
			s.networkStats.Stop()
			return
		case msg := <-s.msgChannel:
			n, err := s.stream.Write(msg.Data)

			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				s.networkStats.AddBytes(uint64(n))
			}
		}
	}
}

func (s *Session) onMessage() func(msg webrtc.DataChannelMessage) {
	return func(msg webrtc.DataChannelMessage) {
		// Store each message in the message channel
		s.msgChannel <- msg
	}
}

func (s *Session) onClose() func() {
	return func() {
		fmt.Println("Done !")
		fmt.Printf("Stats: %s\n", s.networkStats.String())
		close(s.done)
	}
}
