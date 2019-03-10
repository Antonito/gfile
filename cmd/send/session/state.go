package session

import (
	"fmt"
	"io"
	"time"

	"github.com/pions/webrtc"
)

func (s *Session) setStateManager() {
	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	s.peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		fmt.Printf("Connection state is %v\n", s.peerConnection.ConnectionState)
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			s.stopSending <- struct{}{}
		}
	})
}

func (s *Session) writeToNetwork() {
	fmt.Println("Starting to send data...")
	defer fmt.Println("Stopped sending data...")

	for {
	SELECT:
		select {
		case <-s.stopSending:
			fmt.Printf("Pausing network I/O... (remaining at least %v packets)", len(s.output))
			return
		case data := <-s.output:
			if data.n == 0 {
				// The channel is closed, nothing more to send
				s.close(false)
				return
			}

			s.msgToBeSent = append(s.msgToBeSent, data)

			for len(s.msgToBeSent) != 0 {
				cur := s.msgToBeSent[0]

				// TODO: Correct check
				if s.dataChannel.ReadyState != webrtc.DataChannelStateOpen {
					fmt.Printf("Status: %v, dropping %v bytes\n", s.dataChannel.ReadyState, data.n)
					break SELECT
				}

				// Writing packet
				if err := s.dataChannel.Send(cur.buff); err != nil {
					fmt.Printf("Error, cannot send to client: %v\n", err)
					return
				}
				s.nbBytesSent += uint64(cur.n)
				s.msgToBeSent = s.msgToBeSent[1:]
			}
		}
	}
}

func (s *Session) readFile() {
	fmt.Println("Starting to read data...")
	defer func() {
		fmt.Println("Stopped reading data...")
		close(s.output)
	}()

	for {
		// Read file
		s.dataBuff = s.dataBuff[:cap(s.dataBuff)]
		n, err := s.stream.Read(s.dataBuff)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Got EOF after %v bytes!\n", s.nbBytesRead)
				return
			}
			fmt.Printf("Read Error: %v\n", err)
			return
		}
		s.dataBuff = s.dataBuff[:n]
		s.nbBytesRead += uint64(n)

		s.output <- outputMsg{
			n: n,
			// Make a copy of the buffer
			buff: append([]byte(nil), s.dataBuff...),
		}
	}
}

func (s *Session) onOpenHandler() func() {
	return func() {
		if s.timeStart.IsZero() {
			s.timeStart = time.Now()
		}
		s.writeToNetwork()
	}
}

func (s *Session) dumpStats() {
	duration := time.Since(s.timeStart)
	speedMb := (float64(s.nbBytesSent) / 1024 / 1024) / duration.Seconds()
	fmt.Printf("Bytes read: %v\n", s.nbBytesRead)
	fmt.Printf("Bytes sent: %v\n", s.nbBytesSent)
	fmt.Printf("Duration:   %v\n", duration.String())
	fmt.Printf("Speed:      %.04f MB/s\n", speedMb)
}

func (s *Session) close(calledFromCloseHandler bool) {
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

func (s *Session) onCloseHandler() func() {
	return func() {
		s.close(true)
	}
}
