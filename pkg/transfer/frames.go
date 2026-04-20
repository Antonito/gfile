package transfer

import (
	"context"
	"errors"

	"github.com/antonito/gfile/internal/protocol"
	"github.com/antonito/gfile/internal/session"
)

// RunFrames spawns a goroutine that drives ch.OnFrames and forwards any
// non-context.Canceled error to onErr. Cancel ctx to stop the loop.
func RunFrames(
	ctx context.Context,
	ch *session.Channel,
	fh protocol.FrameHandler,
	bufSize int,
	onErr func(error),
) {
	go func() {
		err := ch.OnFrames(ctx, fh, bufSize)
		if err != nil && !errors.Is(err, context.Canceled) {
			onErr(err)
		}
	}()
}
