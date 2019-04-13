package receive

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/session/receiver"
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

	conf := receiver.Config{
		Stream: f,
	}

	customSTUN := c.String("stun")
	if customSTUN != "" {
		if err := utils.ParseSTUN(customSTUN); err != nil {
			return err
		}
		conf.STUN = customSTUN
	}

	sess := receiver.NewWith(conf)
	return sess.Start()
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
			cli.StringFlag{
				Name:  "stun",
				Usage: "Use a specific STUN server (ex: --stun stun.l.google.com:19302)",
			},
		},
	}
}
