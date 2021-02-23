package cmd

import (
	"github.com/antonito/gfile/cmd/receive"
	"github.com/antonito/gfile/cmd/send"
	"sort"

	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

// Install all the commands
func Install(app *cli.App) {
	app.Commands = []cli.Command{
		send.New(),
		receive.New(),
	}
	log.Trace("Installed commands")

	sort.Sort(cli.CommandsByName(app.Commands))
}
