package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/antonito/gfile/internal/protocol"
)

const (
	// bufferThreshold is the BufferedAmount above which SendData blocks.
	bufferThreshold = 512 * 1024

	// bpPollInterval is the fallback poll period for the back-pressure loop.
	bpPollInterval = 100 * time.Millisecond
)

// ErrOnFramesAlreadyRunning is returned by Channel.OnFrames
// when called more than once on the same Channel.
var ErrOnFramesAlreadyRunning = errors.New("OnFrames already running on this channel")

// Channel is a framed wrapper over a pion DataChannel.
type Channel struct {
	dc          *webrtc.DataChannel
	detached    bool
	open        chan struct{}
	closed      chan struct{}
	sendSig     chan struct{}
	framesGuard atomic.Bool
	frameBuf    []byte
}

func newChannel(dc *webrtc.DataChannel, detached bool) *Channel {
	ch := &Channel{
		dc:       dc,
		detached: detached,
		open:     make(chan struct{}),
		closed:   make(chan struct{}),
		sendSig:  make(chan struct{}, 1),
	}
	dc.SetBufferedAmountLowThreshold(bufferThreshold)
	dc.OnBufferedAmountLow(func() {
		select {
		case ch.sendSig <- struct{}{}:
		default:
		}
	})

	var openOnce, closedOnce sync.Once
	dc.OnOpen(func() {
		openOnce.Do(func() {
			close(ch.open)
		})
	})
	dc.OnClose(func() {
		closedOnce.Do(func() {
			close(ch.closed)
		})
	})

	return ch
}

// Label returns the underlying DataChannel's label.
func (c *Channel) Label() string {
	return c.dc.Label()
}

// Open returns a channel closed when the underlying DataChannel opens.
func (c *Channel) Open() <-chan struct{} {
	return c.open
}

// Closed returns a channel closed when the underlying DataChannel closes.
func (c *Channel) Closed() <-chan struct{} {
	return c.closed
}

// Close closes the underlying DataChannel.
func (c *Channel) Close() error {
	return c.dc.Close()
}

// SendMetadata encodes meta and sends it as a METADATA frame.
func (c *Channel) SendMetadata(meta protocol.Metadata) error {
	return c.dc.Send(protocol.EncodeMetadata(meta))
}

// SendEOF sends a one-byte EOF frame.
func (c *Channel) SendEOF() error {
	return c.dc.Send(protocol.EncodeEOF())
}

// SendAbort sends an ABORT frame carrying a free-form reason string.
func (c *Channel) SendAbort(reason string) error {
	return c.dc.Send(protocol.EncodeAbort(reason))
}

// SendAddPeerOffer sends an ADD_PEER_OFFER frame with peerID and the offer SDP.
func (c *Channel) SendAddPeerOffer(peerID uint8, sdp string) error {
	return c.dc.Send(protocol.EncodeAddPeerOffer(peerID, sdp))
}

// SendAddPeerAnswer sends an ADD_PEER_ANSWER frame with peerID and the answer SDP.
func (c *Channel) SendAddPeerAnswer(peerID uint8, sdp string) error {
	return c.dc.Send(protocol.EncodeAddPeerAnswer(peerID, sdp))
}

// SendTransferComplete sends a one-byte TRANSFER_COMPLETE frame.
func (c *Channel) SendTransferComplete() error {
	return c.dc.Send(protocol.EncodeTransferComplete())
}

// SendData encodes payload as a DATA frame, sends it, then waits for the
// outgoing buffer to drain below bufferThreshold before returning.
func (c *Channel) SendData(
	ctx context.Context,
	offset uint64,
	payload []byte,
) error {
	c.frameBuf = protocol.AppendData(c.frameBuf[:0], offset, payload)
	if err := c.dc.Send(c.frameBuf); err != nil {
		return err
	}

	for c.dc.BufferedAmount() > bufferThreshold {
		select {
		case <-c.sendSig:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bpPollInterval):
		}
	}

	return nil
}

// OnFrames runs the receive loop, dispatching every incoming frame to handler
// via protocol.Dispatch.
func (c *Channel) OnFrames(
	ctx context.Context,
	handler protocol.FrameHandler,
	bufSize int,
) error {
	if !c.framesGuard.CompareAndSwap(false, true) {
		return ErrOnFramesAlreadyRunning
	}

	if c.detached {
		return c.runDetached(ctx, handler, bufSize)
	}

	return c.runOnMessage(ctx, handler)
}

func (c *Channel) runDetached(
	ctx context.Context,
	handler protocol.FrameHandler,
	bufSize int,
) error {
	select {
	case <-c.open:
	case <-c.closed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

	rwc, err := c.dc.Detach()
	if err != nil {
		return fmt.Errorf("detach: %w", err)
	}

	// Watch ctx: close rwc to unblock the Read below when cancelled.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = rwc.Close()
		case <-stop:
		}
	}()

	buf := make([]byte, bufSize)
	for {
		numBytes, rerr := rwc.Read(buf)
		if numBytes > 0 {
			if derr := protocol.Dispatch(buf[:numBytes], handler); derr != nil {
				return derr
			}
		}
		if rerr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return rerr
		}
	}
}

func (c *Channel) runOnMessage(
	ctx context.Context,
	handler protocol.FrameHandler,
) error {
	errCh := make(chan error, 1)
	c.dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if derr := protocol.Dispatch(msg.Data, handler); derr != nil {
			select {
			case errCh <- derr:
			default:
			}
		}
	})

	select {
	case err := <-errCh:
		return err
	case <-c.closed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Flush waits up to timeout for the outgoing buffer to drain
func (c *Channel) Flush(
	timeout time.Duration,
	now func() time.Time,
) {
	deadline := now().Add(timeout)

	for c.dc.BufferedAmount() > 0 && now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
}
