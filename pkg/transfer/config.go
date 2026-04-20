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
	// STUNServers is a list of STUN host:port entries. Each is prefixed
	// with "stun:" before being passed through to the ICE config. A nil
	// or empty slice disables STUN entirely — useful on a LAN where
	// host/mDNS candidates are enough.
	STUNServers []string
	// DisableQR suppresses the QR render of the local SDP.
	DisableQR bool
	// LoopbackOnly pins ICE to lo0 and drops STUN (bench only).
	LoopbackOnly bool
	// DisableMDNS suppresses mDNS candidate gathering. Zero-value means
	// mDNS is on (peers advertise `.local` hostnames for host candidates).
	DisableMDNS bool
	// ICELite enables pion's ICE-Lite mode on every PC this session opens.
	//
	// Test-only: it's only safe when both peers are ICE-lite on a guaranteed-
	// routable path (loopback in-process).
	//
	// Production callers must leave this false; the CLI never sets it.
	ICELite bool
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
	if len(cfg.STUNServers) > 0 {
		stun = make([]string, len(cfg.STUNServers))
		for i, s := range cfg.STUNServers {
			stun[i] = "stun:" + s
		}
	}

	return internalSess.Config{
		STUNServers:  stun,
		LoopbackOnly: cfg.LoopbackOnly,
		DisableMDNS:  cfg.DisableMDNS,
		ICELite:      cfg.ICELite,
	}
}
