package bench

import (
	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/session/bench"
	"github.com/antonito/gfile/pkg/session/common"
	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

func handler(c *cli.Context) error {
	isMaster := c.Bool("master")

	conf := bench.Config{
		Master: isMaster,
		Configuration: common.Configuration{
			OnCompletion: func() {
			},
		},
	}

	customSTUN := c.String("stun")
	if customSTUN != "" {
		if err := utils.ParseSTUN(customSTUN); err != nil {
			return err
		}
		conf.STUN = customSTUN
	}

	sess := bench.NewWith(conf)
	return sess.Start()
}

// New creates the command
func New() cli.Command {
	log.Traceln("Installing 'bench' command")
	return cli.Command{
		Name:    "bench",
		Aliases: []string{"b"},
		Usage:   "Benchmark the connexion",
		Action:  handler,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "master, m",
				Usage: "Is creating the SDP offer?",
			},
			cli.StringFlag{
				Name:  "stun",
				Usage: "Use a specific STUN server (ex: --stun stun.l.google.com:19302)",
			},
		},
	}
}
