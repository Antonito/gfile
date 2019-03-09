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
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			s.stopSending <- struct{}{}
		}
	})
}

func (s *Session) readFileTick() error {
	// Read file
	s.dataBuff = s.dataBuff[:cap(s.dataBuff)]
	n, err := s.stream.Read(s.dataBuff)
	if err != nil {
		if err == io.EOF {
			fmt.Printf("Got EOF after %v bytes!\n", s.nbBytesRead)
			s.close(false)
			return io.EOF
		}
		fmt.Printf("Read Error: %v\n", err)
		return err
	}
	s.dataBuff = s.dataBuff[:n]
	s.nbBytesRead += uint64(n)

	// Writing packet
	if err := s.dataChannel.Send(s.dataBuff); err != nil {
		fmt.Printf("Error, cannot send to client: %v\n", err)
		return err
	}
	s.nbBytesSent += uint64(n)
	return nil
}

func (s *Session) onOpenHandler() func() {
	return func() {
		if s.timeStart.IsZero() {
			s.timeStart = time.Now()
		}
		fmt.Println("Starting to send data...")
		defer fmt.Println("Stopped sending data...")

		for {
			select {
			case <-s.stopSending:
				return
			default:
				if err := s.readFileTick(); err != nil {
					if err == io.EOF {
						return
					}
					fmt.Printf("Error read: %v\n", err)
				}
			}
		}
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
	if calledFromCloseHandler == false {
		s.dataChannel.Close()
	}

	// Sometime, onCloseHandler is not invoked, so it's a work-around
	s.doneCheckLock.Lock()
	if s.doneCheck == true {
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
