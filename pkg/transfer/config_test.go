package transfer

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveIODefaults(t *testing.T) {
	in, out := ResolveIO(IOConfig{})
	assert.Same(t, os.Stdin, in, "nil SDPProvider should fall back to os.Stdin")
	assert.Same(t, os.Stdout, out, "nil SDPOutput should fall back to os.Stdout")
}

func TestResolveIOUsesProvided(t *testing.T) {
	reader := strings.NewReader("x")
	buf := &bytes.Buffer{}
	in, out := ResolveIO(IOConfig{
		SDPProvider: reader,
		SDPOutput:   buf,
	})
	assert.Equal(t, reader, in)
	assert.Equal(t, buf, out)
}

func TestBuildInternalConfigEmptySTUN(t *testing.T) {
	cfg := BuildInternalConfig(IOConfig{})
	assert.Empty(t, cfg.STUNServers)
	assert.False(t, cfg.LoopbackOnly)
}

func TestBuildInternalConfigWrapsSTUN(t *testing.T) {
	cfg := BuildInternalConfig(IOConfig{STUN: "stun.example.com:3478"})
	assert.Equal(t, []string{"stun:stun.example.com:3478"}, cfg.STUNServers)
}

func TestBuildInternalConfigLoopback(t *testing.T) {
	cfg := BuildInternalConfig(IOConfig{LoopbackOnly: true})
	assert.True(t, cfg.LoopbackOnly)
}
