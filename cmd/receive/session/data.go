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
			return
		case msg := <-s.msgChannel:
			if _, err := s.stream.Write(msg.Data); err != nil {
				fmt.Printf("Error: %v\n", err)
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
		close(s.done)
	}
}
