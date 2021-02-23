package session

import (
	"fmt"

	"github.com/antonito/gfile/pkg/utils"
	"github.com/pion/webrtc/v3"
)

func (s *Session) readSDPFromInput() webrtc.SessionDescription {
	var sdp webrtc.SessionDescription

	fmt.Println("Please, paste the remote SDP:")
	for {
		encoded, err := utils.MustReadStream(s.sdpIO.Input)
		if err == nil {
			if err := utils.Decode(encoded, &sdp); err == nil {
				break
			}
		}
		fmt.Println("Invalid SDP, try again...")
	}

	return sdp
}

// createSessionDescription set the local description and print the SDP
func (s *Session) printSDPToOutput(desc webrtc.SessionDescription) error {
	desc.SDP = utils.StripSDP(desc.SDP)

	// Output the SDP in base64 so we can paste it in another client
	resp, err := utils.Encode(desc)
	if err != nil {
		return err
	}

	fmt.Println("Send this SDP:")
	fmt.Fprintf(s.sdpIO.Output, "%s\n", resp)

	return nil
}