package receiver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/internal/protocol"
	internalSess "github.com/antonito/gfile/internal/session"
	"github.com/antonito/gfile/internal/stats"
	"github.com/antonito/gfile/pkg/transfer"
)

// multiRouter ingests control-channel frames and owns the data PCs created
// for each ADD_PEER_OFFER. Transfer state lives in transferCore.
type multiRouter struct {
	protocol.UnexpectedFrameHandler
	core *transferCore

	mu    sync.Mutex
	peers map[uint8]*receivePeer

	loopbackOnly bool
	ctrl         *internalSess.Channel
	runCtx       context.Context
}

type receivePeer struct {
	id      uint8
	sess    *internalSess.Session
	ch      *internalSess.Channel
	decoder *zstd.Decoder
	scratch []byte
}

func newMultiRouter(
	file *os.File,
	path string,
	ns *stats.Stats,
	loopback bool,
	ctrl *internalSess.Channel,
	runCtx context.Context,
) *multiRouter {
	r := &multiRouter{
		peers:        make(map[uint8]*receivePeer),
		loopbackOnly: loopback,
		ctrl:         ctrl,
		runCtx:       runCtx,
	}
	r.core = newTransferCore(file, path, ns, r.cleanup)
	return r
}

func (r *multiRouter) OnMetadata(meta protocol.Metadata) error {
	if err := r.core.handleMetadata(meta); err != nil {
		return r.core.fail(err)
	}

	return nil
}

func (r *multiRouter) OnAddPeerOffer(id uint8, offerSDP string) error {
	r.mu.Lock()
	if _, exists := r.peers[id]; exists {
		r.mu.Unlock()
		return fmt.Errorf("protocol error: duplicate peer_id %d", id)
	}
	// reserve
	r.peers[id] = nil
	r.mu.Unlock()

	go r.negotiateReceivePeer(id, offerSDP)

	return nil
}

func (r *multiRouter) OnTransferComplete() error {
	if r.core.loadedMeta() == nil {
		return r.core.fail(errors.New("protocol error: TRANSFER_COMPLETE before METADATA"))
	}

	go r.core.verifyAndClose()

	return nil
}

// OnAbort fails with "abort: <reason>" verbatim, matching singleHandler.
func (r *multiRouter) OnAbort(reason string) error {
	return r.core.fail(fmt.Errorf("abort: %s", reason))
}

func (r *multiRouter) negotiateReceivePeer(id uint8, offerSDP string) {
	sess := internalSess.NewReceiver(internalSess.Config{
		LoopbackOnly: r.loopbackOnly,
	})

	// Register sess into r.peers up front so cleanup() will close it even
	// if we're cancelled or fail mid-handshake. OnAddPeerOffer already
	// reserved the slot as nil; we overwrite with a real peer.
	r.mu.Lock()
	r.peers[id] = &receivePeer{id: id, sess: &sess}
	r.mu.Unlock()

	if err := sess.CreateConnection(func(state webrtc.ICEConnectionState) {
		log.Debug().Msgf("recv data-%d ICE state %s", id, state)
	}); err != nil {
		_ = r.core.fail(fmt.Errorf("peer %d CreateConnection: %w", id, err))
		return
	}

	expectedLabel := protocol.LabelForDataPeer(int(id))
	sess.OnChannel(func(ch *internalSess.Channel) {
		if ch.Label() != expectedLabel {
			log.Warn().Msgf("peer %d got unexpected DC label %q", id, ch.Label())
			return
		}
		r.installDataPeer(id, ch)
	})

	answerSDP, err := sess.AcceptOffer(offerSDP)
	if err != nil {
		_ = r.core.fail(fmt.Errorf("peer %d AcceptOffer: %w", id, err))
		return
	}

	// Bail before we publish an ANSWER if the transfer was cancelled or
	// already failed; cleanup() has (or will) close peer.sess.
	select {
	case <-r.runCtx.Done():
		return
	case <-r.core.done:
		return
	default:
	}

	if err := r.ctrl.SendAddPeerAnswer(id, answerSDP); err != nil {
		_ = r.core.fail(fmt.Errorf("peer %d send ANSWER: %w", id, err))
		return
	}
}

// cleanup tears down every peer's decoder and PC. Runs inside doneOnce.
func (r *multiRouter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, peer := range r.peers {
		if peer == nil {
			continue
		}

		if peer.decoder != nil {
			peer.decoder.Close()
			peer.decoder = nil
		}

		if peer.sess != nil {
			_ = peer.sess.Close()
		}
	}
}

func (r *multiRouter) waitDone() error {
	return r.core.waitDone()
}

// failOnEarlyClose is called from the control DataChannel's OnClose.
func (r *multiRouter) failOnEarlyClose() error {
	select {
	case <-r.core.done:
		return nil
	default:
		return r.core.fail(errors.New("control channel closed before TRANSFER_COMPLETE"))
	}
}

func (r *multiRouter) installDataPeer(id uint8, ch *internalSess.Channel) {
	var dec *zstd.Decoder
	if meta := r.core.loadedMeta(); meta != nil && meta.Codec == protocol.CodecZstd {
		d, err := transfer.NewDataDecoder()
		if err != nil {
			_ = r.core.fail(fmt.Errorf("peer %d decoder: %w", id, err))
			return
		}
		dec = d
	}

	// negotiateReceivePeer always populated the slot before registering the
	// OnChannel callback that calls us, so peer is guaranteed non-nil.
	r.mu.Lock()
	peer := r.peers[id]
	peer.ch = ch
	peer.decoder = dec
	r.mu.Unlock()

	transfer.RunFrames(r.runCtx, ch, &peerFrameHandler{r: r, p: peer}, protocol.MaxDataReadBufSize, func(err error) {
		_ = r.core.fail(fmt.Errorf("peer %d OnFrames: %w", id, err))
	})

	go func() {
		<-ch.Closed()
		select {
		case <-r.core.done:
			return
		default:
			meta := r.core.loadedMeta()
			if meta == nil || r.core.bytesWritten.Load() < meta.FileSize {
				_ = r.core.fail(fmt.Errorf("data peer %d closed before TRANSFER_COMPLETE", id))
			}
		}
	}()
}

func (r *multiRouter) onPeerData(peer *receivePeer, data protocol.Data) error {
	scratch, err := r.core.ingestData(peer.decoder, peer.scratch, data)
	peer.scratch = scratch
	return err
}

// peerFrameHandler dispatches DATA on a data peer's channel;
// other frame types fall through to UnexpectedFrameHandler.
type peerFrameHandler struct {
	protocol.UnexpectedFrameHandler
	r *multiRouter
	p *receivePeer
}

func (h *peerFrameHandler) OnData(data protocol.Data) error {
	return h.r.onPeerData(h.p, data)
}
