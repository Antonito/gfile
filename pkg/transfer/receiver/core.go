package receiver

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/antonito/gfile/internal/protocol"
	"github.com/antonito/gfile/internal/stats"
	"github.com/antonito/gfile/pkg/transfer"
)

// transferCore is the receive-side state machine shared by the singleHandler
// and the multiRouter. Callers decompress first and hand decompressed
// payloads to writeChunk.
type transferCore struct {
	sink     *os.File
	path     string       // "" disables os.Remove on failure
	netStats *stats.Stats // nil allowed (tests)
	cleanup  func()       // bound at construction; runs once inside doneOnce on fail/verify

	// meta is set once via CAS; subsequent loads are race-free.
	meta atomic.Pointer[protocol.Metadata]

	bytesWritten atomic.Uint64
	// allBytesCh closes when bytesWritten reaches FileSize; verifyAndClose selects on it.
	allBytesCh   chan struct{}
	allBytesOnce sync.Once

	done     chan struct{}
	doneOnce sync.Once
	result   error // set inside doneOnce before close(done)
}

// newTransferCore returns a core writing to file. cleanup runs exactly once
// inside the done gate (on fail or successful verify) so the owning handler
// can release its decoder/peers without re-implementing once-only semantics.
func newTransferCore(file *os.File, path string, ns *stats.Stats, cleanup func()) *transferCore {
	return &transferCore{
		sink:       file,
		path:       path,
		netStats:   ns,
		cleanup:    cleanup,
		done:       make(chan struct{}),
		allBytesCh: make(chan struct{}),
	}
}

// handleMetadata validates meta, CAS's it in (rejecting duplicates), and truncates the sink.
func (c *transferCore) handleMetadata(meta protocol.Metadata) error {
	if meta.Version != protocol.ProtocolVersion {
		return fmt.Errorf("unsupported protocol version: 0x%02x", meta.Version)
	}

	switch meta.Codec {
	case protocol.CodecNone, protocol.CodecZstd:
	default:
		return fmt.Errorf("unsupported codec: 0x%02x", meta.Codec)
	}

	if !c.meta.CompareAndSwap(nil, &meta) {
		return errors.New("protocol error: duplicate METADATA")
	}

	if err := c.sink.Truncate(int64(meta.FileSize)); err != nil {
		return fmt.Errorf("truncate output: %w", err)
	}

	if meta.FileSize == 0 {
		c.allBytesOnce.Do(func() { close(c.allBytesCh) })
	}

	return nil
}

// ingestData decompresses data.Payload with dec (nil skips decode) and
// writes the result at data.Offset. On failure, fails the transfer and
// returns the error. The returned scratch buffer (possibly nil) must be
// stored back on the caller's per-source state for reuse.
func (c *transferCore) ingestData(
	dec *zstd.Decoder,
	scratch []byte,
	data protocol.Data,
) ([]byte, error) {
	payload, newScratch, err := transfer.DecodeData(dec, scratch, data.Payload)
	if err != nil {
		return scratch, c.fail(err)
	}
	if err := c.writeChunk(data.Offset, payload); err != nil {
		return newScratch, c.fail(err)
	}
	return newScratch, nil
}

// writeChunk bounds-checks and writes payload at offset. Caller must have already decompressed.
func (c *transferCore) writeChunk(offset uint64, payload []byte) error {
	meta := c.meta.Load()
	if meta == nil {
		return errors.New("protocol error: DATA before METADATA")
	}

	end := offset + uint64(len(payload))
	if end < offset || end > meta.FileSize {
		return fmt.Errorf("protocol error: DATA out of bounds (offset %d + len %d > size %d)",
			offset, len(payload), meta.FileSize)
	}
	if _, err := c.sink.WriteAt(payload, int64(offset)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	newBytes := c.bytesWritten.Add(uint64(len(payload)))
	if newBytes >= meta.FileSize {
		c.allBytesOnce.Do(func() { close(c.allBytesCh) })
	}

	if c.netStats != nil {
		c.netStats.AddBytes(uint64(len(payload)))
	}

	return nil
}

func (c *transferCore) loadedMeta() *protocol.Metadata {
	return c.meta.Load()
}

func (c *transferCore) fail(err error) error {
	c.doneOnce.Do(func() {
		c.result = err
		c.cleanup()
		c.abortFile()
		close(c.done)
	})

	return err
}

// verifyAndClose waits up to 30s for all bytes, then
// runs sync + SHA256 + sink.Close
func (c *transferCore) verifyAndClose() {
	meta := c.meta.Load()
	if meta == nil {
		_ = c.fail(errors.New("protocol error: verifyAndClose before METADATA"))
		return
	}

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	select {
	case <-c.allBytesCh:
	case <-c.done:
		return
	case <-timeout.C:
		_ = c.fail(fmt.Errorf("transfer completion with only %d/%d bytes after 30s",
			c.bytesWritten.Load(), meta.FileSize))
		return
	}

	if c.bytesWritten.Load() > meta.FileSize {
		_ = c.fail(fmt.Errorf("received %d bytes, expected %d",
			c.bytesWritten.Load(), meta.FileSize))
		return
	}

	c.doneOnce.Do(func() {
		defer close(c.done)
		c.cleanup()
		if err := c.sink.Sync(); err != nil {
			c.result = fmt.Errorf("sync output: %w", err)
			c.abortFile()
			return
		}
		if err := verifyIntegrity(c.sink, int64(meta.FileSize), meta.SHA256); err != nil {
			c.result = err
			c.abortFile()
			return
		}
		_ = c.sink.Close()
	})
}

func (c *transferCore) waitDone() error {
	<-c.done
	return c.result
}

func (c *transferCore) abortFile() {
	_ = c.sink.Close()
	if c.path != "" {
		_ = os.Remove(c.path)
	}
}

// verifyIntegrity SHA-256s size bytes from reader and compares against expected.
func verifyIntegrity(reader io.ReaderAt, size int64, expected [32]byte) error {
	hasher := sha256.New()
	buf := make([]byte, 64*1024)
	var offset int64
	for offset < size {
		numBytes, err := reader.ReadAt(buf, offset)
		if numBytes > 0 {
			hasher.Write(buf[:numBytes])
			offset += int64(numBytes)
		}
		if err != nil {
			if errors.Is(err, io.EOF) && offset == size {
				break
			}
			return fmt.Errorf("hash: %w", err)
		}
	}
	var sum [32]byte
	copy(sum[:], hasher.Sum(nil))
	if sum != expected {
		return errors.New("integrity check failed")
	}
	return nil
}
