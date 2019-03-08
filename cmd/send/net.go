package send

import (
	"fmt"
	"io"

	pions "github.com/pions/webrtc"
)

func setupDatachannel(f io.Reader, dataChannel *pions.DataChannel) <-chan struct{} {
	done := make(chan struct{})
	dataChannel.OnOpen(func() {
		defer func() {
			fmt.Printf("Done!")
			dataChannel.Close()
			close(done)
		}()

		// Read file blocks and send them
		data := make([]byte, 8192)
		for {
			data = data[:cap(data)]
			n, err := f.Read(data)
			if err != nil {
				if err == io.EOF {
					fmt.Printf("Got EOF !")
					break
				}
				fmt.Printf("Read Error: %v\n", err)
				return
			}
			data = data[:n]
			if err := dataChannel.Send(data); err != nil {
				fmt.Printf("Error, cannot send to client: %v\n", err)
				return
			}
		}
	})
	return done
}
