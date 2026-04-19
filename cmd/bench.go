package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/antonito/gfile/internal/debug"
	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/pkg/transfer"
	"github.com/antonito/gfile/pkg/transfer/receiver"
	"github.com/antonito/gfile/pkg/transfer/sender"
)

// newBenchCmd builds the `bench` subcommand. Flag state is captured via
// closure so each Execute() invocation gets a fresh command tree.
func newBenchCmd() *cobra.Command {
	var (
		asSender         bool
		sizeMB           int
		compressionLevel int
		connections      int
		loopback         bool
	)

	cmd := &cobra.Command{
		Use:          "bench",
		Aliases:      []string{"b"},
		Short:        "Benchmark the connection via a real send/receive transfer",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sizeMB < 1 {
				return fmt.Errorf("--size must be >= 1 MB (got %d)", sizeMB)
			}

			cfg := sender.Config{
				IOConfig: transfer.IOConfig{
					DisableQR:    true,
					LoopbackOnly: loopback,
				},
				CompressionLevel: compressionLevel,
				Connections:      connections,
			}
			if err := cfg.Validate(); err != nil {
				return err
			}

			envKey := "GFILE_PPROF_RECEIVER"
			if asSender {
				envKey = "GFILE_PPROF_SENDER"
			}

			debug.StartPprof(envKey)
			if asSender {
				return runBenchSender(cfg, sizeMB)
			}

			return runBenchReceiver(loopback)
		},
	}
	cmd.Flags().BoolVarP(&asSender, "sender", "s", false, "Create the SDP offer")
	cmd.Flags().IntVar(&sizeMB, "size", 500, "Transfer size in MB (sender only)")
	cmd.Flags().IntVar(&compressionLevel, "compression-level", 0,
		"zstd compression level (0 disables, 1..22). Defaults to 0 since bench uses random data.")
	cmd.Flags().IntVar(&connections, "connections", 1,
		"Number of parallel data PeerConnections (1..16)")
	cmd.Flags().BoolVar(&loopback, "loopback", false,
		"Restrict ICE to loopback (skip STUN). Off by default — enable to bench a "+
			"local same-host run; leave off to measure a real cross-host path.")
	return cmd
}

// runBenchSender generates a random file of sizeMB and sends it using
// the real gfile sender. Emits an `Upload: <bytes> | <duration> | <MB/s>`
// line that scripts/bench.py parses.
func runBenchSender(cfg sender.Config, sizeMB int) error {
	dir, err := os.MkdirTemp("", "gfile-bench-sender-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	path := filepath.Join(dir, "bench.bin")
	size := int64(sizeMB) * 1024 * 1024
	if err := writeRandomFile(path, size); err != nil {
		return fmt.Errorf("generate bench file: %w", err)
	}

	// Give scripts/bench.py the total byte count so its progress bar can be
	// size-based instead of guessing at a 20s fallback duration.
	output.BenchTotal(size)

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	cfg.Stream = file
	sess, err := sender.NewWith(cfg)
	if err != nil {
		return err
	}
	if err := sess.Start(); err != nil {
		return err
	}
	output.Stats("sender", sess.NetworkStats())
	return nil
}

// runBenchReceiver receives into a throwaway temp file and reports throughput
// as a `Download:` line.
func runBenchReceiver(loopback bool) error {
	dir, err := os.MkdirTemp("", "gfile-bench-receiver-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	outPath := filepath.Join(dir, "bench-out.bin")
	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sess := receiver.NewWith(receiver.Config{
		IOConfig: transfer.IOConfig{
			DisableQR:    true,
			LoopbackOnly: loopback,
		},
		Stream: file,
		Path:   outPath,
	})
	if err := sess.Start(ctx); err != nil {
		return err
	}
	output.Stats("receiver", sess.NetworkStats())
	return nil
}

// writeRandomFile creates path filled with size bytes of crypto-random
// data and fsyncs it. Uses a fixed-size buffer so large bench sizes
// don't balloon RAM.
func writeRandomFile(path string, size int64) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(file, rand.Reader, size); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
