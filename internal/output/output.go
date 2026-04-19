package output

import (
	"io"
	"os"

	"github.com/antonito/gfile/internal/stats"
)

// Default is the process-wide emitter.
var Default = New(ModeText, os.Stdout, os.Stderr)

// SetDefault replaces the process-wide Default emitter and returns a
// restore function that puts the previous emitter back.
func SetDefault(emitter *Emitter) (restore func()) {
	prev := Default
	Default = emitter

	return func() {
		Default = prev
	}
}

// SetMode is called once from cobra's PersistentPreRun on rootCmd
// before any subcommand runs.
func SetMode(mode Mode) {
	Default.SetMode(mode)
}

// CurrentMode returns the active output mode.
func CurrentMode() Mode {
	return Default.CurrentMode()
}

// Prompt writes an interactive message to stderr. No-op in JSON mode.
func Prompt(msg string) {
	Default.Prompt(msg)
}

// BenchTotal writes the pre-transfer total byte count.
func BenchTotal(bytes int64) {
	Default.BenchTotal(bytes)
}

// Stats writes per-side final transfer statistics.
func Stats(role string, sts *stats.Stats) {
	Default.Stats(role, sts)
}

// TransferComplete emits a one-line confirmation that the transfer
// finished successfully for role, identifying the file and its size.
func TransferComplete(role, path string, bytes int64) {
	Default.TransferComplete(role, path, bytes)
}

// SDP writes the local session description.
func SDP(writer io.Writer, role, sdp string) {
	Default.SDP(writer, role, sdp)
}

// Progress emits a transfer progress event (throttled per-role).
func Progress(role string, bytes int64, bytesPerSec float64) {
	Default.Progress(role, bytes, bytesPerSec)
}

// Sample records a fresh bandwidth sample from sts and emits the resulting
// byte count + current bandwidth as a progress event for role.
func Sample(role string, sts *stats.Stats) {
	sts.Sample()
	Default.Progress(role, int64(sts.Bytes()), sts.CurrentBandwidth())
}

// Fatal reports a fatal error before non-zero process exit.
func Fatal(err error, kindDefault string) {
	Default.Fatal(err, kindDefault)
}
