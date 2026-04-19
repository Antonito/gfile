package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/protocol"
)

// connectedPair returns two opened *Channels backed by an in-process
// PeerConnection pair. Detach mode on each side is independently configured.
//
// The returned cleanup closes both PeerConnections; defer it from the test.
//
// Test design: both sides drive a real pion stack so the back-pressure,
// detach, and OnMessage paths all exercise pion's actual SCTP behavior —
// no mocks. This matches the existing pattern in pkg/transfer/integration.
func connectedPair(t *testing.T, detachA, detachB bool) (chA, chB *Channel, cleanup func()) {
	t.Helper()

	apiA := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine(detachA)))
	apiB := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine(detachB)))

	pcA, err := apiA.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	pcB, err := apiB.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)

	dcA, err := pcA.CreateDataChannel("test", nil)
	require.NoError(t, err)

	bCh := make(chan *webrtc.DataChannel, 1)
	pcB.OnDataChannel(func(dc *webrtc.DataChannel) { bCh <- dc })

	offer, err := pcA.CreateOffer(nil)
	require.NoError(t, err)
	require.NoError(t, pcA.SetLocalDescription(offer))
	<-webrtc.GatheringCompletePromise(pcA)
	require.NoError(t, pcB.SetRemoteDescription(*pcA.LocalDescription()))

	answer, err := pcB.CreateAnswer(nil)
	require.NoError(t, err)
	require.NoError(t, pcB.SetLocalDescription(answer))
	<-webrtc.GatheringCompletePromise(pcB)
	require.NoError(t, pcA.SetRemoteDescription(*pcB.LocalDescription()))

	var dcB *webrtc.DataChannel
	select {
	case dcB = <-bCh:
	case <-time.After(10 * time.Second):
		t.Fatal("OnDataChannel not fired within 10s")
	}

	chA = newChannel(dcA, detachA)
	chB = newChannel(dcB, detachB)

	cleanup = func() {
		_ = pcA.Close()
		_ = pcB.Close()
	}
	return chA, chB, cleanup
}

func settingEngine(detach bool) webrtc.SettingEngine {
	se := webrtc.SettingEngine{}
	if detach {
		se.DetachDataChannels()
	}
	se.SetIncludeLoopbackCandidate(true)
	return se
}

// waitOpen waits up to timeout for ch to fire its Open signal.
func waitOpen(t *testing.T, ch *Channel, timeout time.Duration) {
	t.Helper()
	select {
	case <-ch.Open():
	case <-time.After(timeout):
		t.Fatalf("channel %q did not open within %s", ch.Label(), timeout)
	}
}

func TestChannel_OpenAndClosed(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()

	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	// Closing the underlying DC on side A propagates to its own Closed signal.
	require.NoError(t, chA.Close())
	select {
	case <-chA.Closed():
	case <-time.After(5 * time.Second):
		t.Fatal("chA.Closed() did not fire within 5s")
	}
}

func TestChannel_ClosedFiresExactlyOnce(t *testing.T) {
	// The struct uses sync.Once internally; we verify that closing the
	// underlying DC twice does not panic (which would happen if the
	// closed-channel close happened twice).
	chA, _, cleanup := connectedPair(t, false, false)
	defer cleanup()

	waitOpen(t, chA, 5*time.Second)
	require.NoError(t, chA.Close())
	require.NoError(t, chA.Close()) // must not panic
}

// readOne reads one DataChannel message from ch's underlying DC via
// OnMessage; ch must be in non-detached mode.
func readOne(t *testing.T, ch *Channel, timeout time.Duration) []byte {
	t.Helper()
	got := make(chan []byte, 1)
	ch.dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case got <- msg.Data:
		default:
		}
	})
	select {
	case data := <-got:
		return data
	case <-time.After(timeout):
		t.Fatal("no message received within deadline")
		return nil
	}
}

func TestChannel_SendMetadata_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecZstd,
		FileSize: 12345,
	}
	for ndx := range meta.SHA256 {
		meta.SHA256[ndx] = byte(ndx)
	}
	require.NoError(t, chA.SendMetadata(meta))

	got := readOne(t, chB, 5*time.Second)
	require.Equal(t, protocol.EncodeMetadata(meta), got)
}

func TestChannel_SendEOF_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)
	require.NoError(t, chA.SendEOF())
	require.Equal(t, protocol.EncodeEOF(), readOne(t, chB, 5*time.Second))
}

func TestChannel_SendAbort_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)
	require.NoError(t, chA.SendAbort("nope"))
	require.Equal(t, protocol.EncodeAbort("nope"), readOne(t, chB, 5*time.Second))
}

func TestChannel_SendTransferComplete_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)
	require.NoError(t, chA.SendTransferComplete())
	require.Equal(t, protocol.EncodeTransferComplete(), readOne(t, chB, 5*time.Second))
}

func TestChannel_SendAddPeerOffer_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)
	require.NoError(t, chA.SendAddPeerOffer(7, "sdp-text"))
	require.Equal(t, protocol.EncodeAddPeerOffer(7, "sdp-text"), readOne(t, chB, 5*time.Second))
}

func TestChannel_SendAddPeerAnswer_Bytes(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)
	require.NoError(t, chA.SendAddPeerAnswer(3, "sdp-answer"))
	require.Equal(t, protocol.EncodeAddPeerAnswer(3, "sdp-answer"), readOne(t, chB, 5*time.Second))
}

func TestChannel_SendData_RoundTrip(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	payload := []byte("hello world")
	require.NoError(t, chA.SendData(context.Background(), 42, payload))

	got := readOne(t, chB, 5*time.Second)
	require.Equal(t, protocol.EncodeData(42, payload), got)
}

func TestChannel_SendData_ContextCancelDuringBackpressure(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	// Pin the receiver's OnMessage to never drain (block in the callback)
	// so the sender's outgoing buffer fills past bufferThreshold.
	blockUntil := make(chan struct{})
	chB.dc.OnMessage(func(_ webrtc.DataChannelMessage) {
		<-blockUntil
	})
	defer close(blockUntil)

	// Push enough bytes to exceed bufferThreshold (512 KB).
	chunk := make([]byte, 64*1024)
	ctx, cancel := context.WithCancel(t.Context())

	sendErr := make(chan error, 1)
	go func() {
		// Loop until SendData blocks on back-pressure (returns nil for first
		// few sends, then blocks on the throttle for-loop). We cancel after a
		// short delay; SendData must return ctx.Err().
		var lastErr error
		for ndx := range 32 {
			lastErr = chA.SendData(ctx, uint64(ndx)*uint64(len(chunk)), chunk)
			if lastErr != nil {
				break
			}
		}
		sendErr <- lastErr
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-sendErr:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("SendData did not unblock within 2s of cancel")
	}
}

// captureHandler implements protocol.FrameHandler and records every call.
// All non-overridden methods inherit UnexpectedFrameHandler's "return error"
// behavior, so unexpected frames surface as test failures.
//
// mu guards metas and eofs because OnFrames calls the handler from its own
// goroutine while require.Eventually reads the counters from a poller goroutine.
type captureHandler struct {
	protocol.UnexpectedFrameHandler
	mu    sync.Mutex
	metas []protocol.Metadata
	eofs  int
	err   error // set by On* to be returned (for error-propagation test)
}

func (h *captureHandler) OnMetadata(meta protocol.Metadata) error {
	h.mu.Lock()
	h.metas = append(h.metas, meta)
	h.mu.Unlock()
	return h.err
}
func (h *captureHandler) OnEOF() error {
	h.mu.Lock()
	h.eofs++
	h.mu.Unlock()
	return h.err
}

func (h *captureHandler) metaCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.metas)
}

func (h *captureHandler) eofCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.eofs
}

func TestChannel_OnFrames_Detached_RoutesFrames(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, true) // sender non-detached, receiver detached
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	handler := &captureHandler{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- chB.OnFrames(ctx, handler, protocol.MaxControlReadBufSize)
	}()

	require.NoError(t, chA.SendMetadata(protocol.Metadata{
		Version: protocol.ProtocolVersion, Codec: protocol.CodecNone, FileSize: 1,
	}))
	require.NoError(t, chA.SendEOF())

	// Wait for both frames to arrive at the handler.
	require.Eventually(t, func() bool {
		return handler.metaCount() == 1 && handler.eofCount() == 1
	}, 3*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("OnFrames did not exit within 2s of cancel")
	}
}

func TestChannel_OnFrames_NonDetached_RoutesFrames(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, false) // both non-detached
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	handler := &captureHandler{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- chB.OnFrames(ctx, handler, 0)
	}()

	require.NoError(t, chA.SendMetadata(protocol.Metadata{
		Version: protocol.ProtocolVersion, Codec: protocol.CodecNone, FileSize: 1,
	}))

	require.Eventually(t, func() bool { return handler.metaCount() == 1 },
		3*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("OnFrames did not exit within 2s of cancel")
	}
}

func TestChannel_OnFrames_DoubleCall(t *testing.T) {
	_, chB, cleanup := connectedPair(t, false, false)
	defer cleanup()
	waitOpen(t, chB, 5*time.Second)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		_ = chB.OnFrames(ctx, &captureHandler{}, 0)
	}()

	// Give the first call a moment to take the guard.
	time.Sleep(50 * time.Millisecond)
	err := chB.OnFrames(ctx, &captureHandler{}, 0)
	require.ErrorIs(t, err, ErrOnFramesAlreadyRunning)
}

func TestChannel_OnFrames_HandlerErrorPropagates(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, true)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	handler := &captureHandler{err: errors.New("boom")}
	done := make(chan error, 1)
	go func() {
		done <- chB.OnFrames(context.Background(), handler, protocol.MaxControlReadBufSize)
	}()

	require.NoError(t, chA.SendMetadata(protocol.Metadata{
		Version: protocol.ProtocolVersion, FileSize: 1,
	}))

	select {
	case err := <-done:
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	case <-time.After(3 * time.Second):
		t.Fatal("OnFrames did not propagate handler error within 3s")
	}
}

func TestChannel_OnFrames_PeerCloseExitsCleanly(t *testing.T) {
	chA, chB, cleanup := connectedPair(t, false, true)
	defer cleanup()
	waitOpen(t, chA, 5*time.Second)
	waitOpen(t, chB, 5*time.Second)

	done := make(chan error, 1)
	go func() {
		done <- chB.OnFrames(context.Background(), &captureHandler{}, protocol.MaxControlReadBufSize)
	}()

	// Give OnFrames a moment to enter Detach + Read.
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, chA.Close())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("OnFrames did not exit within 5s of remote close")
	}
}
