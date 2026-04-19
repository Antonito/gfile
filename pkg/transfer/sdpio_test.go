package transfer

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/internal/utils"
)

// encodeTestSDP builds an EncodeSDP-encoded string that DecodeSDP will accept.
func encodeTestSDP(t *testing.T) string {
	t.Helper()
	sdp := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\n"}
	enc, err := utils.EncodeSDP(sdp)
	require.NoError(t, err, "EncodeSDP")
	return enc
}

// swapOutput replaces the process-wide emitter so output.Prompt / output.SDP
// don't leak stderr/stdout into the test runner.
func swapOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	stderr := &bytes.Buffer{}
	restore := output.SetDefault(output.New(output.ModeText, io.Discard, stderr))
	t.Cleanup(restore)
	return stderr
}

func TestEmitSDP(t *testing.T) {
	stderr := swapOutput(t)
	out := &bytes.Buffer{}

	EmitSDP(out, RoleSender, "encoded-sdp")

	assert.Equal(t, "encoded-sdp\n", out.String())
	assert.Equal(t, "Send this SDP:\n", stderr.String())
}

func TestEmitSDPReceiver(t *testing.T) {
	stderr := swapOutput(t)
	out := &bytes.Buffer{}

	EmitSDP(out, RoleReceiver, "encoded-sdp")

	assert.Equal(t, "encoded-sdp\n", out.String())
	assert.Equal(t, "Send this SDP:\n", stderr.String())
}

func TestMaybeShowQRDisabled(t *testing.T) {
	stderr := swapOutput(t)
	MaybeShowQR("encoded-sdp", true)
	assert.Empty(t, stderr.String(),
		"MaybeShowQR with disableQR=true must not write to stderr")
}

func TestMaybeShowQRSkipsInJSONMode(t *testing.T) {
	stderr := &bytes.Buffer{}
	restore := output.SetDefault(output.New(output.ModeJSON, io.Discard, stderr))
	defer restore()
	MaybeShowQR("encoded-sdp", false)
	assert.Empty(t, stderr.String(),
		"MaybeShowQR in JSON mode must not write to stderr")
}

func TestReadRemoteSDPPlain(t *testing.T) {
	swapOutput(t)
	enc := encodeTestSDP(t)

	got, err := ReadRemoteSDP(strings.NewReader(enc + "\n"))

	require.NoError(t, err)
	assert.Equal(t, enc, got)
}

func TestReadRemoteSDPJSONEnvelope(t *testing.T) {
	swapOutput(t)
	enc := encodeTestSDP(t)
	env, err := json.Marshal(struct {
		SDP string `json:"sdp"`
	}{SDP: enc})
	require.NoError(t, err)

	got, err := ReadRemoteSDP(strings.NewReader(string(env) + "\n"))

	require.NoError(t, err)
	assert.Equal(t, enc, got)
}
