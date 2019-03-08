package send

import (
	"fmt"
	"io"
	"os"

	"github.com/Antonito/gfile/pkg/utils"
	"github.com/Antonito/gfile/pkg/webrtc"
	pions "github.com/pions/webrtc"
	"gopkg.in/urfave/cli.v1"
)

func sendFile(f io.Reader) error {
	peerConnection, err := webrtc.NewConnection()
	if err != nil {
		return err
	}

	// Create a datachannel with label 'data'
	//ordered := true
	//priority := pions.PriorityTypeHigh
	//maxRetries := uint16(128)
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return err
	}
	done := setupDatachannel(f, dataChannel)
	sdpChan := utils.HTTPSDPServer()

	// Create an offer to send to the browser
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		return err
	}

	// Output the offer in base64 so we can paste it in browser
	fmt.Println(utils.Encode(offer))

	// Wait for the answer to be pasted
	fmt.Println(`Please, provide the SDP via:
curl localhost:8080/sdp --data "$SDP"`)
	answer := pions.SessionDescription{}
	utils.Decode(<-sdpChan, &answer)

	// Apply the answer as the remote description
	err = peerConnection.SetRemoteDescription(answer)
	if err != nil {
		return err
	}

	<-done
	return nil
}

func handler(c *cli.Context) error {
	fileToSend := c.String("file")
	if fileToSend == "" {
		return fmt.Errorf("file parameter missing")
	}
	f, err := os.Open(fileToSend)
	if err != nil {
		return err
	}
	defer f.Close()
	return sendFile(f)
}

// New creates the command
func New() cli.Command {
	return cli.Command{
		Name:    "send",
		Aliases: []string{"s"},
		Usage:   "Sends a file",
		Action:  handler,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file, f",
				Usage: "Send content of file `FILE`",
			},
		},
	}
}
