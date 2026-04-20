package transfer

import (
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// NewDataEncoder builds a single-threaded zstd encoder for DATA payloads.
// One per PC so the library pool doesn't multiply by N.
func NewDataEncoder(level int) (*zstd.Encoder, error) {
	return zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)),
		zstd.WithEncoderConcurrency(1),
	)
}

// NewDataDecoder builds a single-threaded zstd decoder for DATA payloads.
func NewDataDecoder() (*zstd.Decoder, error) {
	return zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
}

// DecodeData decompresses in with dec, reusing scratch to avoid allocation.
// When dec is nil it returns in unchanged.
func DecodeData(dec *zstd.Decoder, scratch, in []byte) (payload, newScratch []byte, err error) {
	if dec == nil {
		return in, scratch, nil
	}

	newScratch, err = dec.DecodeAll(in, scratch[:0])
	if err != nil {
		return nil, scratch, fmt.Errorf("protocol error: zstd decode: %w", err)
	}

	return newScratch, newScratch, nil
}
