package session

import (
	"fmt"

	"github.com/pions/webrtc"
)

func (s *Session) receiveData() {
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
		s.msgChannel <- msg
	}
}

func (s *Session) onClose() func() {
	return func() {
		fmt.Println("Done !")
		close(s.done)
	}
}
