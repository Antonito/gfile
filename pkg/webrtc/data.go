package webrtc

import (
	"fmt"

	"github.com/pions/webrtc"
)

type dataChannelHandler func(*webrtc.DataChannel, chan struct{})

// DataChannelHandler is a small wrapper around boiler plate code for OnDataChannel
func DataChannelHandler(pc *webrtc.PeerConnection, handler dataChannelHandler) <-chan struct{} {
	done := make(chan struct{})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label, d.ID)
		handler(d, done)
	})

	return done
}
