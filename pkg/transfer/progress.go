package transfer

import (
	"context"
	"time"

	"github.com/antonito/gfile/internal/output"
	"github.com/antonito/gfile/internal/stats"
)

// progressInterval is the per-role progress emission period.
const progressInterval = 100 * time.Millisecond

// EmitProgressSamples emits an immediate Sample tagged with role and then
// one every progressInterval until ctx is canceled. The initial sample
// guarantees at least one event for transfers that finish in tens of
// milliseconds (loopback benches, tiny files).
func EmitProgressSamples(
	ctx context.Context,
	role Role,
	ns *stats.Stats,
) {
	name := role.String()
	output.Sample(name, ns)

	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			output.Sample(name, ns)
		case <-ctx.Done():
			return
		}
	}
}
