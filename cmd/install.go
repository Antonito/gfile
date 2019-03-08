package cmd

import (
	"sort"

	receive "github.com/Antonito/gfile/cmd/receive"
	send "github.com/Antonito/gfile/cmd/send"
	"gopkg.in/urfave/cli.v1"
)

// Install all the commands
func Install(app *cli.App) {
	app.Commands = []cli.Command{
		send.New(),
		receive.New(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
}
