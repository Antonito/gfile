package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/pkg/transfer"
	"github.com/antonito/gfile/pkg/transfer/sender"
)

// newSendCmd builds the `send` subcommand
func newSendCmd(globalFlags *globalFlags) *cobra.Command {
	var (
		file             string
		qr               bool
		compressionLevel int
		connections      int
	)

	cmd := &cobra.Command{
		Use:          "send",
		Aliases:      []string{"s"},
		Short:        "Send a file",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fh, err := os.Open(file)
			if err != nil {
				return err
			}
			defer func() {
				_ = fh.Close()
			}()

			stuns, err := globalFlags.ResolvedSTUNs()
			if err != nil {
				return err
			}

			cfg := sender.Config{
				IOConfig: transfer.IOConfig{
					STUNServers: stuns,
					DisableQR:   !qr,
				},
				Stream:           fh,
				CompressionLevel: compressionLevel,
				Connections:      connections,
			}
			if err := cfg.Validate(); err != nil {
				return err
			}

			sess, err := sender.NewWith(cfg)
			if err != nil {
				return err
			}

			if err := sess.Start(); err != nil {
				return err
			}

			fi, err := fh.Stat()
			if err != nil {
				return err
			}
			output.TransferComplete("sender", file, fi.Size())
			return nil
		},
	}

	cmd.Flags().
		StringVarP(&file, "file", "f", "", "File to send")
	_ = cmd.MarkFlagRequired("file")

	cmd.Flags().
		BoolVar(
			&qr, "qr", false,
			"Display the SDP offer as a QR code in addition to text",
		)

	cmd.Flags().
		IntVar(&compressionLevel, "compression-level", 1,
			"zstd compression level (0 disables, 1..22)",
		)

	cmd.Flags().
		IntVar(&connections, "connections", 1,
			"Number of parallel data PeerConnections (1..16)",
		)

	return cmd
}
