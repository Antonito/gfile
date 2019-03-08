package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/Antonito/gfile/pkg/utils"
	"github.com/Antonito/gfile/pkg/webrtc"
	pions "github.com/pions/webrtc"
	"gopkg.in/urfave/cli.v1"
)

func receiveFile(f io.Writer) error {
	peerConnection, err := webrtc.NewConnection()
	if err != nil {
		return err
	}
	// Register data channel creation handling
	done := webrtc.DataChannelHandler(peerConnection, func(d *pions.DataChannel, stop chan struct{}) {
		// Register text message handling
		d.OnMessage(func(msg pions.DataChannelMessage) {
			if _, err := f.Write(msg.Data); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		})

		d.OnClose(func() {
			fmt.Println("Done !")
			close(stop)
		})
	})

	// Wait for the offer to be pasted
	offer := pions.SessionDescription{}
	utils.Decode(utils.MustReadStdin(), &offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(utils.Encode(answer))

	<-done
	return nil
}

func handler(c *cli.Context) error {
	output := c.String("output")
	if output == "" {
		return fmt.Errorf("output parameter missing")
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return receiveFile(f)
}

// New creates the command
func New() cli.Command {
	return cli.Command{
		Name:    "receive",
		Aliases: []string{"r"},
		Usage:   "Receive a file",
		Action:  handler,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "output, o",
				Usage: "Output",
			},
		},
	}
}
