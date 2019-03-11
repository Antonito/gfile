package cmd

import (
	"sort"

	"github.com/antonito/gfile/cmd/receive"
	"github.com/antonito/gfile/cmd/sdp"
	"github.com/antonito/gfile/cmd/send"
	"gopkg.in/urfave/cli.v1"
)

// Install all the commands
func Install(app *cli.App) {
	app.Commands = []cli.Command{
		send.New(),
		receive.New(),
		sdp.New(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
}
