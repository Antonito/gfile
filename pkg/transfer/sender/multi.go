package sender

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/internal/protocol"
	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/pkg/transfer"
)

type multiChunk struct {
	offset  uint64
	payload []byte  // raw (uncompressed); a subslice of *buf
	buf     *[]byte // pool-owned full-capacity buffer; release via releaseChunk
}

// chunkBufPool recycles ChunkSize read buffers so the hot path doesn't
// allocate 256 KB per chunk. Entries are *[]byte for single-pointer storage.
var chunkBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, protocol.ChunkSize)
		return &buf
	},
}

// releaseChunk returns a chunk's buffer to the pool. Zero-value chunks are a no-op.
func releaseChunk(chunk multiChunk) {
	if chunk.buf == nil {
		return
	}
	// Reset to full cap so the next ReadFull can refill the whole buffer.
	*chunk.buf = (*chunk.buf)[:cap(*chunk.buf)]
	chunkBufPool.Put(chunk.buf)
}

// multiProduce reads reader in ChunkSize pieces onto out, closing at EOF.
// Workers must releaseChunk each chunk so its buffer returns to the pool.
func multiProduce(reader io.Reader, out chan<- multiChunk) error {
	defer close(out)
	var offset uint64
	for {
		bufPtr := chunkBufPool.Get().(*[]byte)
		buf := (*bufPtr)[:protocol.ChunkSize]
		numBytes, err := io.ReadFull(reader, buf)
		switch err {
		case nil:
			out <- multiChunk{offset: offset, payload: buf[:numBytes], buf: bufPtr}
			offset += uint64(numBytes)
		case io.EOF:
			chunkBufPool.Put(bufPtr)
			return nil
		case io.ErrUnexpectedEOF:
			if numBytes > 0 {
				out <- multiChunk{offset: offset, payload: buf[:numBytes], buf: bufPtr}
			} else {
				chunkBufPool.Put(bufPtr)
			}
			return nil
		default:
			chunkBufPool.Put(bufPtr)
			return fmt.Errorf("read: %w", err)
		}
	}
}

// multiWorker drains in and sends DATA frames on peer.ch. One zstd encoder per worker so encoding parallelizes.
func (s *Session) multiWorker(ctx context.Context, peer *dataPeer, in <-chan multiChunk) error {
	var enc *zstd.Encoder
	if s.zstdLevel > 0 {
		encoder, err := transfer.NewDataEncoder(s.zstdLevel)
		if err != nil {
			return fmt.Errorf("peer %d encoder: %w", peer.id, err)
		}
		defer func() {
			_ = encoder.Close()
		}()
		enc = encoder
	}
	var compressed []byte
	for {
		select {
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			payload := chunk.payload
			if enc != nil {
				compressed = enc.EncodeAll(chunk.payload, compressed[:0])
				payload = compressed
			}
			uncompressedLen := uint64(len(chunk.payload))
			if err := peer.ch.SendData(ctx, chunk.offset, payload); err != nil {
				releaseChunk(chunk)
				return fmt.Errorf("peer %d send: %w", peer.id, err)
			}
			releaseChunk(chunk)
			s.sess.NetworkStats.AddBytes(uncompressedLen)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// dataPeer holds one data-carrying PeerConnection on the sender side.
type dataPeer struct {
	id   uint8
	sess internalSess.Session
	ch   *internalSess.Channel
}

// peerAnswer is delivered through s.answerCh when a receiver ANSWER arrives.
type peerAnswer struct {
	peerID uint8
	sdp    string
}

const peerSetupTimeout = 30 * time.Second

// peerSetupTimeoutOverride holds a test-only nanoseconds override.
var peerSetupTimeoutOverride atomic.Int64

// SetPeerSetupTimeoutForTest overrides the peer-setup timeout for tests. Pass 0 to reset.
func SetPeerSetupTimeoutForTest(timeout time.Duration) {
	peerSetupTimeoutOverride.Store(int64(timeout))
}

// runMulti sends METADATA, runs the ADD_PEER handshake, and drives the data flow.
func (s *Session) runMulti(ctx context.Context) {
	defer s.close(false)

	if err := s.preTransfer(ctx); err != nil {
		return
	}

	peers, err := s.buildDataPeers()
	if err != nil {
		s.sendAbort(fmt.Sprintf("build peers: %v", err))
		return
	}
	s.dataPeers = peers
	if err := s.negotiatePeers(ctx, peers); err != nil {
		s.sendAbort(err.Error())
		return
	}
	log.Debug().Msgf("all %d data peers connected", len(peers))

	// Workers aggregate into s.sess.NetworkStats; reuse the single-PC
	// progress format so bench.py's parser works identically.
	stopProgress := transfer.StartProgressEmitter(ctx, transfer.RoleSender, s.sess.NetworkStats)
	defer stopProgress()

	chunks := make(chan multiChunk, 2*len(peers))
	prodErr := make(chan error, 1)
	go func() {
		prodErr <- multiProduce(s.stream, chunks)
	}()

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Go(func() {
			errs <- s.multiWorker(ctx, peer, chunks)
		})
	}
	wg.Wait()
	stopProgress()
	// Drain remaining chunks so the producer unblocks and buffers return to the pool.
	for chunk := range chunks {
		releaseChunk(chunk)
	}

	if readErr := <-prodErr; readErr != nil {
		s.sendAbort(fmt.Sprintf("read: %v", readErr))
		return
	}
	close(errs)
	for workerErr := range errs {
		if workerErr != nil {
			s.sendAbort(workerErr.Error())
			return
		}
	}

	// Flush each data PC's send buffer in parallel; serial wait would
	// scale wall-time as N * 10s under the worst-case per-peer timeout.
	var flushWg sync.WaitGroup
	for _, peer := range peers {
		flushWg.Go(func() {
			peer.ch.Flush(10*time.Second, time.Now)
		})
	}
	flushWg.Wait()
	if err := s.ch.SendTransferComplete(); err != nil {
		log.Error().Err(err).Msg("send TRANSFER_COMPLETE failed")
		return
	}
	s.ch.Flush(2*time.Second, time.Now)
	s.sess.NetworkStats.Stop()
}

// closePeers best-effort closes every peer's session. Safe on partial slices.
func closePeers(peers []*dataPeer) {
	for _, peer := range peers {
		if peer != nil {
			_ = peer.sess.Close()
		}
	}
}

// buildDataPeers creates s.connections data peers with one "data-<i>" DataChannel each.
func (s *Session) buildDataPeers() ([]*dataPeer, error) {
	peers := make([]*dataPeer, 0, s.connections)

	for ndx := range s.connections {
		peer := &dataPeer{
			id: uint8(ndx),
			sess: internalSess.New(internalSess.Config{
				LoopbackOnly: s.sess.IsLoopbackOnly(),
				ICELite:      s.sess.IsICELite(),
			}),
		}
		peerID := peer.id

		if err := peer.sess.CreateConnection(func(state webrtc.ICEConnectionState) {
			log.Debug().Msgf("data-%d ICE state %s", peerID, state)
		}); err != nil {
			closePeers(peers)
			_ = peer.sess.Close()
			return nil, fmt.Errorf("data-%d CreateConnection: %w", ndx, err)
		}

		ch, err := peer.sess.CreateChannel(protocol.LabelForDataPeer(ndx))
		if err != nil {
			closePeers(peers)
			_ = peer.sess.Close()
			return nil, fmt.Errorf("data-%d CreateChannel: %w", ndx, err)
		}

		peer.ch = ch
		peers = append(peers, peer)
	}

	return peers, nil
}

// cancelHandshake closes every peer PC to unblock pending offers,
// then waits up to 5s for offer goroutines.
// Called on all negotiatePeers error paths.
func (s *Session) cancelHandshake(peers []*dataPeer, wg *sync.WaitGroup) {
	closePeers(peers)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Warn().Msg("offer goroutines did not exit within 5s")
	}
}

// negotiatePeers runs the parallel ADD_PEER handshake under a single wall-clock deadline.
func (s *Session) negotiatePeers(ctx context.Context, peers []*dataPeer) error {
	timeout := peerSetupTimeout
	if ns := peerSetupTimeoutOverride.Load(); ns > 0 {
		timeout = time.Duration(ns)
	}
	setupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Go(func() {
			offer, err := peer.sess.MakeOffer()
			if err != nil {
				errs <- fmt.Errorf("peer %d MakeOffer: %w", peer.id, err)
				return
			}
			if err := s.ch.SendAddPeerOffer(peer.id, offer); err != nil {
				errs <- fmt.Errorf("peer %d send offer: %w", peer.id, err)
				return
			}
		})
	}

	waiting := make(map[uint8]*dataPeer, len(peers))
	for _, peer := range peers {
		waiting[peer.id] = peer
	}

	for len(waiting) > 0 {
		select {
		case answer := <-s.answerCh:
			peer, ok := waiting[answer.peerID]
			if !ok {
				s.cancelHandshake(peers, &wg)
				return fmt.Errorf("answer for unknown peer_id %d", answer.peerID)
			}
			delete(waiting, answer.peerID)
			if err := peer.sess.AcceptAnswer(answer.sdp); err != nil {
				s.cancelHandshake(peers, &wg)
				return fmt.Errorf("peer %d AcceptAnswer: %w", answer.peerID, err)
			}
		case err := <-errs:
			s.cancelHandshake(peers, &wg)
			return err
		case <-setupCtx.Done():
			s.cancelHandshake(peers, &wg)
			return fmt.Errorf("peer setup timeout after %s (%d still pending)",
				timeout, len(waiting))
		}
	}

	// Should be no-op on most cases, but kept for correctness
	wg.Wait()

	for _, peer := range peers {
		select {
		case <-peer.ch.Open():
		case <-setupCtx.Done():
			s.cancelHandshake(peers, &wg)
			return fmt.Errorf("peer %d open timeout", peer.id)
		}
	}

	return nil
}

// senderCtrlHandler handles the sender's control-channel frames in multi-PC mode.
// Only ADD_PEER_ANSWER and ABORT are expected.
type senderCtrlHandler struct {
	protocol.UnexpectedFrameHandler
	s *Session
}

func (h *senderCtrlHandler) OnAddPeerAnswer(id uint8, sdp string) error {
	select {
	case h.s.answerCh <- peerAnswer{peerID: id, sdp: sdp}:
	default:
		log.Error().Msg("sender answerCh full; aborting (protocol violation)")
		h.s.stopSend()
	}

	return nil
}

func (h *senderCtrlHandler) OnAbort(reason string) error {
	log.Warn().Msgf("receiver ABORT: %s", reason)
	h.s.stopSend()

	return nil
}
