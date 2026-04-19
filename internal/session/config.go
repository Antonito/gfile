package session

// Config describes the networking knobs for a PeerConnection-backed Session.
// A zero value is valid: default STUN (applied at CreateConnection time),
// no loopback pinning.
type Config struct {
	// STUNServers are the full ICE STUN URLs ("stun:host:port"). A nil or
	// empty slice resolves to the default Google STUN URL at
	// CreateConnection time. Passed through to webrtc.ICEServer.URLs.
	STUNServers []string

	// LoopbackOnly pins ICE to loopback-only interfaces and drops STUN,
	// so the session produces only host candidates on lo0. Intended for
	// the in-process benchmark (just bench) — deterministic path, no
	// network interface required.
	LoopbackOnly bool
}
