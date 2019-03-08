package main

import (
	"log"
	"os"

	"github.com/Antonito/gfile/cmd"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "gfile"
	app.Version = "0.1"
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version, V",
		Usage: "print only the version",
	}

	cmd.Install(app)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
