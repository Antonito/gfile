package send

import (
	"fmt"
	"os"

	"github.com/antonito/gfile/cmd/send/session"
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
	session := session.NewSession(f)
	return session.Connect()
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
		},
	}
}
