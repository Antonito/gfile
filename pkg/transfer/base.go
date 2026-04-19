package transfer

import (
	"io"

	"github.com/antonito/gfile/internal/stats"
)

// SessionBase is embedded by sender and receiver sessions to share the
// NetworkStats getter and the resolved SDP I/O handles.
type SessionBase struct {
	networkStats *stats.Stats
	sdpInput     io.Reader
	sdpOutput    io.Writer
}

// NewSessionBase returns a SessionBase holding ns and the SDP I/O handles
// resolved from cfg (stdin/stdout defaults applied).
func NewSessionBase(ns *stats.Stats, cfg IOConfig) SessionBase {
	in, out := ResolveIO(cfg)
	return SessionBase{
		networkStats: ns,
		sdpInput:     in,
		sdpOutput:    out,
	}
}

// NetworkStats returns the stats pointer. Meaningful after Start returns.
func (b *SessionBase) NetworkStats() *stats.Stats {
	return b.networkStats
}

// SDPInput returns the SDP input handle.
func (b *SessionBase) SDPInput() io.Reader {
	return b.sdpInput
}

// SDPOutput returns the SDP output handle.
func (b *SessionBase) SDPOutput() io.Writer {
	return b.sdpOutput
}
