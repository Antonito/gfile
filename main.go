package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/cmd"
)

func setupLogger() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	logLevel := zerolog.WarnLevel

	if lvl, ok := os.LookupEnv("GFILE_LOG"); ok {
		switch lvl {
		case "TRACE":
			logLevel = zerolog.TraceLevel
		case "DEBUG":
			logLevel = zerolog.DebugLevel
		case "INFO":
			logLevel = zerolog.InfoLevel
		case "WARN":
			logLevel = zerolog.WarnLevel
		case "PANIC":
			logLevel = zerolog.PanicLevel
		case "ERROR":
			logLevel = zerolog.ErrorLevel
		case "FATAL":
			logLevel = zerolog.FatalLevel
		}
	}
	zerolog.SetGlobalLevel(logLevel)
}

func init() {
	setupLogger()
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
