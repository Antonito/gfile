package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonito/gfile/internal/stats"
)

func TestSessionBaseReturnsStats(t *testing.T) {
	sts := stats.New()
	base := NewSessionBase(sts, IOConfig{})
	assert.Same(t, sts, base.NetworkStats())
}

func TestSessionBaseNilIsAllowed(t *testing.T) {
	base := NewSessionBase(nil, IOConfig{})
	assert.Nil(t, base.NetworkStats())
}
