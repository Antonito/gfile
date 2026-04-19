package receiver

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"

	"github.com/antonito/gfile/internal/protocol"
	"github.com/antonito/gfile/internal/stats"
	"github.com/antonito/gfile/pkg/transfer"
)

// singleHandler is the single-PC frame-ingest adapter. It owns the zstd decoder
// and the EOF gate; state (meta, bounds, writes, verification) lives in transferCore.
type singleHandler struct {
	protocol.UnexpectedFrameHandler
	core    *transferCore
	decoder *zstd.Decoder
	scratch []byte
	// eofSeen races OnClose (failOnEarlyClose) vs OnMessage (OnEOF), so atomic.
	eofSeen atomic.Bool
}

// newSingleHandler creates a singleHandler writing to file. Non-empty path
// removes the partial file on failure; nil netStats is allowed (tests).
func newSingleHandler(file *os.File, path string, netStats *stats.Stats) *singleHandler {
	h := &singleHandler{}
	h.core = newTransferCore(file, path, netStats, h.closeDecoder)
	return h
}

func (h *singleHandler) OnMetadata(meta protocol.Metadata) error {
	if err := h.core.handleMetadata(meta); err != nil {
		return h.core.fail(err)
	}

	if meta.Codec == protocol.CodecZstd {
		dec, err := transfer.NewDataDecoder()
		if err != nil {
			return h.core.fail(fmt.Errorf("zstd decoder: %w", err))
		}
		h.decoder = dec
	}

	return nil
}

func (h *singleHandler) OnData(data protocol.Data) error {
	scratch, err := h.core.ingestData(h.decoder, h.scratch, data)
	h.scratch = scratch
	return err
}

// OnEOF triggers hash verification. pion's per-channel OnMessage
// serialization guarantees every DATA has been delivered before OnEOF runs.
func (h *singleHandler) OnEOF() error {
	if h.core.loadedMeta() == nil {
		return h.core.fail(errors.New("protocol error: EOF before METADATA"))
	}
	if h.eofSeen.Load() {
		return h.core.fail(errors.New("protocol error: duplicate EOF"))
	}

	h.eofSeen.Store(true)
	go h.core.verifyAndClose()

	return nil
}

func (h *singleHandler) OnAbort(reason string) error {
	return h.core.fail(fmt.Errorf("abort: %s", reason))
}

// closeDecoder is the singleHandler-side cleanup closure bound on the core
// at construction time; transferCore runs it once on fail or verify.
func (h *singleHandler) closeDecoder() {
	if h.decoder != nil {
		h.decoder.Close()
	}
}

// waitDone blocks until the transfer completes or fails.
func (h *singleHandler) waitDone() error {
	return h.core.waitDone()
}

// failOnEarlyClose is called from the DataChannel's OnClose. It only fails
// the transfer when the channel closes before EOF was received; no-op
// after EOF (verifyAndClose may still be hashing) or after done.
func (h *singleHandler) failOnEarlyClose() error {
	if h.eofSeen.Load() {
		return nil
	}

	select {
	case <-h.core.done:
		return nil
	default:
		return h.core.fail(errors.New("channel closed before EOF"))
	}
}
