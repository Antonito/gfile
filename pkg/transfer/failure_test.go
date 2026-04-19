package transfer_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShutdownRespectsTimeout guards against deadlocks in the send/receive
// orchestration. A 256 KB transfer should complete in well under a second on
// loopback; runTransfer's per-leg 60s selects will fail the test if either
// side hangs.
func TestShutdownRespectsTimeout(t *testing.T) {
	payload := make([]byte, 256*1024)
	_, err := rand.Read(payload)
	require.NoError(t, err, "rand")
	runTransfer(t, payload)
}
