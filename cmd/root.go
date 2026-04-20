package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/antonito/gfile/internal/output"
)

func newRootCmd() *cobra.Command {
	var jsonOutput bool
	flags := &globalFlags{}

	root := &cobra.Command{
		Use:     "gfile",
		Short:   "A WebRTC based file transfer tool",
		Long:    "Send and receive files directly between two computers using WebRTC, without any third-party server.",
		Version: "0.2",

		// SilenceErrors: let main.go dispatch errors so JSON mode can emit
		// a structured `error` event on stdout.
		//
		// SilenceUsage keeps cobra's usage dump out of error paths.
		SilenceErrors: true,
		SilenceUsage:  true,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				output.SetMode(output.ModeJSON)
				// JSON mode: raw JSON logger on stderr so a parser can ingest
				// it alongside stdout events.
				log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
			} else {
				output.SetMode(output.ModeText)
			}
		},
	}
	root.PersistentFlags().StringSliceVar(&flags.stunServers, "stun",
		[]string{"stun.l.google.com:19302"},
		`STUN servers as comma-separated host:port (e.g. --stun a:3478,b:3478). `+
			`Pass --stun="" to disable STUN and rely on host/mDNS candidates only.`,
	)
	root.PersistentFlags().BoolVar(&jsonOutput, "json-output", false,
		"Emit newline-delimited JSON events on stdout; route human text to stderr",
	)

	root.AddCommand(
		newSendCmd(flags),
		newReceiveCmd(flags),
		newBenchCmd(),
	)

	return root
}

// Execute runs the root command and dispatches any error through the
// output package so JSON mode emits a structured error event.
func Execute() error {
	err := newRootCmd().Execute()
	if err != nil {
		output.Fatal(err, "internal")
	}

	return err
}
