package send

import (
	"fmt"
	"os"

	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/session/sender"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

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
	conf := sender.Config{
		Stream: f,
	}

	customSTUN := c.String("stun")
	if customSTUN != "" {
		if err := utils.ParseSTUN(customSTUN); err != nil {
			return err
		}
		conf.STUN = customSTUN
	}

	sess := sender.NewWith(conf)
	return sess.Start()
}

// New creates the command
func New() cli.Command {
	log.Traceln("Installing 'send' command")
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
			cli.StringFlag{
				Name:  "stun",
				Usage: "Use a specific STUN server (ex: --stun stun.l.google.com:19302)",
			},
		},
	}
}
