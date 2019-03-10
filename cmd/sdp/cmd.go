package sdp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Antonito/gfile/pkg/utils"
	"github.com/pions/webrtc"
	"gopkg.in/urfave/cli.v1"
)

func handler(c *cli.Context) error {
	var encoded string
	for {
		data, err := utils.MustReadStdin()
		if err == nil {
			// We decode it, but never use it,
			// just to make sure the data is correct
			offer := webrtc.SessionDescription{}
			if err := utils.Decode(data, &offer); err == nil {
				encoded = data
				break
			}
		}
		fmt.Println("Invalid SDP, try again...")
	}

	body := strings.NewReader(encoded)
	req, err := http.NewRequest("POST", "http://localhost:8080/sdp", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// New creates the command
func New() cli.Command {
	return cli.Command{
		Name:   "sdp",
		Usage:  "Sends a SDP to the already running instance of the `send` command",
		Action: handler,
	}
}
