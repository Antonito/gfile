package cmd

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/transfer"
	"github.com/antonito/gfile/pkg/transfer/receiver"
)

// newReceiveCmd builds the `receive` subcommand
func newReceiveCmd(globalFlags *globalFlags) *cobra.Command {
	var (
		outputPath string
		sdp        string
	)

	cmd := &cobra.Command{
		Use:          "receive",
		Aliases:      []string{"r"},
		Short:        "Receive a file",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var sdpReader io.Reader
			if sdp != "" {
				encoded, err := utils.ResolveSDPFlag(sdp, os.Stdin)
				if err != nil {
					return err
				}
				sdpReader = strings.NewReader(encoded)
			}

			fh, err := os.OpenFile(outputPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			defer func() {
				_ = fh.Close()
			}()

			stun, err := globalFlags.ResolvedSTUN()
			if err != nil {
				return err
			}

			conf := receiver.Config{
				IOConfig: transfer.IOConfig{
					STUN:        stun,
					SDPProvider: sdpReader,
				},
				Stream: fh,
				Path:   outputPath,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sess := receiver.NewWith(conf)
			if err := sess.Start(ctx); err != nil {
				return err
			}

			// Stat by path: the receiver core closes fh on success.
			fi, err := os.Stat(outputPath)
			if err != nil {
				return err
			}
			output.TransferComplete("receiver", outputPath, fi.Size())
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	_ = cmd.MarkFlagRequired("output")

	cmd.Flags().StringVar(&sdp, "sdp", "",
		"Remote SDP to use (skip the interactive prompt). "+
			"Prefix with @ to read from a file (e.g. --sdp @answer.txt, --sdp @- for stdin).",
	)
	return cmd
}
