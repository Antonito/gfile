package main

import (
	"os"

	"github.com/antonito/gfile/cmd"
	"gopkg.in/urfave/cli.v1"

	log "github.com/sirupsen/logrus"
)

func setupLogger() {
	log.SetOutput(os.Stdout)

	logLevel := log.WarnLevel

	if lvl, ok := os.LookupEnv("GFILE_LOG"); ok {
		switch lvl {
		case "TRACE":
			logLevel = log.TraceLevel
		case "DEBUG":
			logLevel = log.DebugLevel
		case "INFO":
			logLevel = log.InfoLevel
		case "WARN":
			logLevel = log.WarnLevel
		case "PANIC":
			logLevel = log.PanicLevel
		case "ERROR":
			logLevel = log.ErrorLevel
		case "FATAL":
			logLevel = log.FatalLevel
		}
	}
	log.SetLevel(logLevel)
}

func init() {
	setupLogger()
}

func run(args []string) error {
	app := cli.NewApp()
	app.Name = "gfile"
	app.Version = "0.1"
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version, V",
		Usage: "print only the version",
	}
	log.Tracef("Starting %s v%v\n", app.Name, app.Version)

	cmd.Install(app)
	return app.Run(args)
}

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}
