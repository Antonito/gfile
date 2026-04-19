package receiver

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/protocol"
)

func newHandlerForTest(t *testing.T) (*singleHandler, *os.File, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err, "open")
	t.Cleanup(func() { file.Close() })
	return newSingleHandler(file, path, nil), file, path
}

func TestHandlerHappyPath(t *testing.T) {
	payload := []byte("hello, gfile")
	sum := sha256.Sum256(payload)

	h, _, path := newHandlerForTest(t)

	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: uint64(len(payload)),
		SHA256:   sum,
	}
	require.NoError(t, protocol.Dispatch(protocol.EncodeMetadata(meta), h), "metadata")
	require.NoError(t, protocol.Dispatch(protocol.EncodeData(0, payload), h), "data")
	require.NoError(t, protocol.Dispatch(protocol.EncodeEOF(), h), "eof")

	require.NoError(t, h.waitDone(), "waitDone")
	got, err := os.ReadFile(path)
	require.NoError(t, err, "readfile")
	assert.Equal(t, payload, got)
}

func TestHandlerRejectsVersionMismatch(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	meta := protocol.Metadata{Version: 0x99, Codec: protocol.CodecNone}
	err := protocol.Dispatch(protocol.EncodeMetadata(meta), h)
	assert.Error(t, err)
}

func TestHandlerRejectsCodec(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version: protocol.ProtocolVersion,
		Codec:   0x02,
	}
	err := protocol.Dispatch(protocol.EncodeMetadata(meta), h)
	assert.Error(t, err)
}

func TestHandlerRejectsDuplicateMetadata(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version: protocol.ProtocolVersion,
		Codec:   protocol.CodecNone,
	}
	require.NoError(t, protocol.Dispatch(protocol.EncodeMetadata(meta), h), "first")
	assert.Error(t, protocol.Dispatch(protocol.EncodeMetadata(meta), h), "duplicate metadata")
}

func TestHandlerRejectsDataOutOfBounds(t *testing.T) {
	payload := []byte("1234")
	sum := sha256.Sum256(payload)
	h, _, _ := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: uint64(len(payload)),
		SHA256:   sum,
	}
	_ = protocol.Dispatch(protocol.EncodeMetadata(meta), h)
	err := protocol.Dispatch(protocol.EncodeData(3, []byte("XX")), h)
	assert.Error(t, err, "out-of-bounds (offset 3 + len 2 > size 4)")
}

func TestHandlerRejectsDataBeforeMetadata(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	err := protocol.Dispatch(protocol.EncodeData(0, []byte("x")), h)
	assert.Error(t, err, "DATA before METADATA")
}

func TestHandlerDetectsShaMismatch(t *testing.T) {
	payload := []byte("abcd")
	wrong := sha256.Sum256([]byte("different"))

	h, _, path := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: uint64(len(payload)),
		SHA256:   wrong,
	}
	_ = protocol.Dispatch(protocol.EncodeMetadata(meta), h)
	_ = protocol.Dispatch(protocol.EncodeData(0, payload), h)
	_ = protocol.Dispatch(protocol.EncodeEOF(), h)
	require.Error(t, h.waitDone(), "SHA mismatch")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "output not removed on SHA mismatch")
}

func TestHandlerAbort(t *testing.T) {
	h, _, path := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: 4,
		SHA256:   sha256.Sum256([]byte("abcd")),
	}
	_ = protocol.Dispatch(protocol.EncodeMetadata(meta), h)
	_ = protocol.Dispatch(protocol.EncodeAbort("sender died"), h)
	require.EqualError(t, h.waitDone(), "abort: sender died")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "output not removed on abort")
}

func TestHandlerRejectsUnknownFrameType(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	err := protocol.Dispatch([]byte{0x99, 0x00}, h)
	assert.Error(t, err, "unknown frame type")
}

func TestHandlerZstdHappyPath(t *testing.T) {
	payload := make([]byte, 64*1024)
	for ndx := range payload {
		payload[ndx] = byte(ndx % 7) // repetitive, compresses well
	}
	sum := sha256.Sum256(payload)

	enc, err := zstd.NewWriter(nil)
	require.NoError(t, err, "encoder")
	defer enc.Close()
	compressed := enc.EncodeAll(payload, nil)
	require.Less(t, len(compressed), len(payload), "expected compression to shrink payload")

	h, _, path := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecZstd,
		FileSize: uint64(len(payload)),
		SHA256:   sum,
	}
	require.NoError(t, protocol.Dispatch(protocol.EncodeMetadata(meta), h), "metadata")
	require.NoError(t, protocol.Dispatch(protocol.EncodeData(0, compressed), h), "data")
	require.NoError(t, protocol.Dispatch(protocol.EncodeEOF(), h), "eof")
	require.NoError(t, h.waitDone(), "waitDone")
	got, err := os.ReadFile(path)
	require.NoError(t, err, "readfile")
	assert.Equal(t, payload, got)
}

func TestHandlerZstdRejectsCorruptFrame(t *testing.T) {
	h, _, _ := newHandlerForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecZstd,
		FileSize: 16,
		SHA256:   sha256.Sum256(make([]byte, 16)),
	}
	require.NoError(t, protocol.Dispatch(protocol.EncodeMetadata(meta), h), "metadata")
	// Feed bytes that are not a valid zstd frame.
	err := protocol.Dispatch(protocol.EncodeData(0, []byte{0x00, 0x01, 0x02, 0x03}), h)
	assert.Error(t, err, "zstd decode")
}

func TestHandlerAbortBeforeMetadata(t *testing.T) {
	h, _, path := newHandlerForTest(t)
	_ = protocol.Dispatch(protocol.EncodeAbort("upstream died"), h)
	require.EqualError(t, h.waitDone(), "abort: upstream died")
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "output not removed on abort before metadata")
}
