package session

// Config describes the networking knobs for a PeerConnection-backed Session.
// A zero value is valid: no STUN, no loopback pinning. The CLI layer is
// responsible for providing a default STUN server when one is wanted.
type Config struct {
	// STUNServers are the full ICE STUN URLs ("stun:host:port"). A nil
	// or empty slice disables STUN entirely — the session will produce
	// only host (and mDNS) candidates. Passed through to webrtc.ICEServer.URLs.
	STUNServers []string

	// LoopbackOnly pins ICE to loopback-only interfaces and drops STUN,
	// so the session produces only host candidates on lo0. Intended for
	// the in-process benchmark (just bench) — deterministic path, no
	// network interface required.
	LoopbackOnly bool
}
