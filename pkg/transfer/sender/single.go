package sender

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/antonito/gfile/internal/protocol"
	"github.com/antonito/gfile/pkg/transfer"
)

// run drives a single-PC transfer: pre-hash, METADATA, DATA+EOF, flush, close.
func (s *Session) run(ctx context.Context) {
	defer s.close(false)

	if err := s.preTransfer(ctx); err != nil {
		return
	}

	progressCtx, stopProgress := context.WithCancel(ctx)
	go transfer.EmitProgressSamples(progressCtx, transfer.RoleSender, s.sess.NetworkStats)
	defer stopProgress()

	if err := s.sendFile(ctx); err != nil {
		s.sendAbort(err.Error())
		return
	}
	s.sess.NetworkStats.Stop()

	// Flush pion's SCTP buffer before the defer closes the PC — otherwise
	// buffered EOF bytes get dropped and the receiver sees OnClose first.
	s.ch.Flush(10*time.Second, time.Now)
}

// preTransfer pre-hashes, waits for the channel to open, starts stats, and sends METADATA.
func (s *Session) preTransfer(ctx context.Context) error {
	s.readingStats.Start()
	sum, size, err := preHash(s.stream)
	s.readingStats.Stop()
	if err != nil {
		log.Error().Err(err).Msg("pre-hash failed")
		return err
	}

	select {
	case <-s.ch.Open():
	case <-ctx.Done():
		return ctx.Err()
	}
	s.sess.NetworkStats.Start()

	codec := protocol.CodecNone
	if s.zstdLevel > 0 {
		codec = protocol.CodecZstd
	}

	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    codec,
		FileSize: size,
		SHA256:   sum,
	}
	if err := s.ch.SendMetadata(meta); err != nil {
		log.Error().Err(err).Msg("send METADATA failed")
		s.sendAbort(fmt.Sprintf("metadata send failed: %v", err))
		return err
	}

	return nil
}

// sendFile streams the input in ChunkSize DATA frames and ends with EOF.
func (s *Session) sendFile(ctx context.Context) error {
	var offset uint64
	var compressed []byte
	readBuf := make([]byte, protocol.ChunkSize)

	for {
		numBytes, readErr := io.ReadFull(s.stream, readBuf)
		if numBytes > 0 {
			payload := readBuf[:numBytes]
			if s.encoder != nil {
				compressed = s.encoder.EncodeAll(payload, compressed[:0])
				payload = compressed
			}
			if err := s.ch.SendData(ctx, offset, payload); err != nil {
				return err
			}
			s.sess.NetworkStats.AddBytes(uint64(numBytes))
			offset += uint64(numBytes)
		}

		switch readErr {
		case nil:
			continue
		case io.EOF, io.ErrUnexpectedEOF:
			return s.ch.SendEOF()
		default:
			return fmt.Errorf("read: %w", readErr)
		}
	}
}

// preHash streams file through SHA256 and rewinds to 0. file must be seekable.
func preHash(file io.ReadSeeker) (sum [32]byte, size uint64, err error) {
	hasher := sha256.New()
	numBytes, err := io.Copy(hasher, file)

	if err != nil {
		return sum, 0, fmt.Errorf("pre-hash: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return sum, 0, fmt.Errorf("pre-hash rewind: %w", err)
	}
	copy(sum[:], hasher.Sum(nil))

	return sum, uint64(numBytes), nil
}
