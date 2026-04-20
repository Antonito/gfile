package session

import (
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
)

func Test_New_ZeroConfig(t *testing.T) {
	asrt := assert.New(t)

	sess := New(Config{})

	asrt.NotNil(sess.Done)
	asrt.NotNil(sess.NetworkStats)
	asrt.Nil(sess.cfg.STUNServers)
	asrt.False(sess.cfg.LoopbackOnly)
	asrt.False(sess.detach)
	asrt.False(sess.IsLoopbackOnly())
}

func Test_New_CustomConfig(t *testing.T) {
	asrt := assert.New(t)

	sess := New(Config{
		STUNServers:  []string{"stun:custom:3478"},
		LoopbackOnly: true,
	})

	asrt.Equal([]string{"stun:custom:3478"}, sess.cfg.STUNServers)
	asrt.True(sess.cfg.LoopbackOnly)
	asrt.False(sess.detach)
	asrt.True(sess.IsLoopbackOnly())
}

func Test_NewReceiver_EnablesDetach(t *testing.T) {
	asrt := assert.New(t)
	sess := NewReceiver(Config{})
	asrt.True(sess.detach)
}

func Test_CreateConnection_NoSTUN(t *testing.T) {
	asrt := assert.New(t)
	sess := New(Config{}) // nil STUNServers → no ICE server, host-only candidates

	err := sess.CreateConnection(func(webrtc.ICEConnectionState) {})
	asrt.NoError(err)
	asrt.NoError(sess.Close())
}

func Test_CreateConnection_WithSTUN(t *testing.T) {
	asrt := assert.New(t)
	sess := New(Config{STUNServers: []string{"stun:stun.l.google.com:19302"}})

	err := sess.CreateConnection(func(webrtc.ICEConnectionState) {})
	asrt.NoError(err)
	asrt.NoError(sess.Close())
}

func Test_CreateConnection_LoopbackSkipsMDNS(t *testing.T) {
	// Regression guard: LoopbackOnly path should succeed even when
	// DisableMDNS is left at its zero value, since session.go is supposed
	// to short-circuit mDNS gathering on loopback.
	asrt := assert.New(t)
	sess := New(Config{LoopbackOnly: true})

	err := sess.CreateConnection(func(webrtc.ICEConnectionState) {})
	asrt.NoError(err)
	asrt.NoError(sess.Close())
}
