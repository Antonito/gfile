package transfer

import (
	"io"
	"os"

	internalSess "github.com/antonito/gfile/internal/session"
)

// IOConfig is the shared subset of sender/receiver Config driving SDP I/O and PeerConnection setup.
type IOConfig struct {
	// SDPProvider reads the remote SDP. Nil falls back to os.Stdin.
	SDPProvider io.Reader
	// SDPOutput writes the local SDP. Nil falls back to os.Stdout.
	SDPOutput io.Writer
	// STUN is an optional STUN host[:port]. Empty disables STUN.
	STUN string
	// DisableQR suppresses the QR render of the local SDP.
	DisableQR bool
	// LoopbackOnly pins ICE to lo0 and drops STUN (bench only).
	LoopbackOnly bool
}

// ResolveIO fills in stdin/stdout defaults and returns the resolved pair.
func ResolveIO(cfg IOConfig) (in io.Reader, out io.Writer) {
	in, out = cfg.SDPProvider, cfg.SDPOutput

	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}

	return in, out
}

// BuildInternalConfig maps IOConfig to an internalSess.Config.
func BuildInternalConfig(cfg IOConfig) internalSess.Config {
	var stun []string
	if cfg.STUN != "" {
		stun = []string{"stun:" + cfg.STUN}
	}

	return internalSess.Config{
		STUNServers:  stun,
		LoopbackOnly: cfg.LoopbackOnly,
	}
}
