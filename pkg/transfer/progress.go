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

// StartProgressEmitter runs EmitProgressSamples in a goroutine. The returned
// stop cancels the emitter and waits for the goroutine to exit.
func StartProgressEmitter(
	ctx context.Context,
	role Role,
	ns *stats.Stats,
) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		EmitProgressSamples(ctx, role, ns)
	}()

	return func() {
		cancel()
		<-done
	}
}
