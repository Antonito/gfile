package session

// Config describes the networking knobs for a PeerConnection-backed Session.
// A zero value is valid: no STUN, no loopback pinning, mDNS gathering on.
// The CLI layer is responsible for providing a default STUN server when
// one is wanted.
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

	// DisableMDNS suppresses mDNS candidate gathering. When false (the
	// zero value), the session advertises a `.local` hostname for its
	// host candidates instead of a raw LAN IP, matching browser
	// behavior. Ignored under LoopbackOnly.
	DisableMDNS bool

	// ICELite enables pion's ICE-Lite mode, which skips the connectivity-
	// check loop.
	//
	// Test-only: it's only safe when both peers are ICE-lite
	// on a guaranteed-routable path (e.g. loopback in-process).
	//
	// Production transfers must leave this false.
	ICELite bool
}
