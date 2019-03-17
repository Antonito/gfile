package receive

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/antonito/gfile/pkg/session"
	"gopkg.in/urfave/cli.v1"
)

func handler(c *cli.Context) error {
	output := c.String("output")
	if output == "" {
		return fmt.Errorf("output parameter missing")
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	sess := session.NewReceiverWith(session.ReceiverConfig{
		Stream: f,
	})
	return sess.Connect()
}

// New creates the command
func New() cli.Command {
	log.Traceln("Installing 'receive' command")
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
