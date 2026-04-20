package receiver

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonito/gfile/internal/protocol"
)

func newCoreForTest(t *testing.T) (core *transferCore, path string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "out.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err, "open")
	t.Cleanup(func() {
		_ = file.Close()
	})
	core = newTransferCore(file, path, nil, func() {})
	return
}

func newCoreForTestWithCleanup(t *testing.T, cleanup func()) (core *transferCore, path string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "out.bin")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err, "open")
	t.Cleanup(func() {
		_ = file.Close()
	})
	core = newTransferCore(file, path, nil, cleanup)
	return
}

func TestCoreHandleMetadataHappy(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: 16,
	}
	require.NoError(t, core.handleMetadata(meta))
	got := core.loadedMeta()
	require.NotNil(t, got)
	assert.Equal(t, meta, *got)
}

func TestCoreHandleMetadataZeroSizeClosesAllBytesCh(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{
		Version: protocol.ProtocolVersion,
		Codec:   protocol.CodecNone,
	}
	require.NoError(t, core.handleMetadata(meta))
	select {
	case <-core.allBytesCh:
	default:
		t.Fatal("zero-size metadata must close allBytesCh immediately")
	}
}

func TestCoreHandleMetadataRejectsBadVersion(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{Version: 0x99, Codec: protocol.CodecNone}
	assert.Error(t, core.handleMetadata(meta))
}

func TestCoreHandleMetadataRejectsBadCodec(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{Version: protocol.ProtocolVersion, Codec: 0x02}
	assert.Error(t, core.handleMetadata(meta))
}

func TestCoreHandleMetadataRejectsDuplicate(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{Version: protocol.ProtocolVersion, Codec: protocol.CodecNone}
	require.NoError(t, core.handleMetadata(meta))
	assert.Error(t, core.handleMetadata(meta))
}

func TestCoreWriteChunkHappy(t *testing.T) {
	core, path := newCoreForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: 5,
	}
	require.NoError(t, core.handleMetadata(meta))
	require.NoError(t, core.writeChunk(0, []byte("hello")))
	select {
	case <-core.allBytesCh:
	default:
		t.Fatal("allBytesCh should close after final byte written")
	}
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestCoreWriteChunkOutOfBounds(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: 4,
	}
	require.NoError(t, core.handleMetadata(meta))
	err := core.writeChunk(3, []byte("XX"))
	assert.Error(t, err)
}

func TestCoreFailIsIdempotent(t *testing.T) {
	cleaned := 0
	core, path := newCoreForTestWithCleanup(t, func() {
		cleaned++
	})
	_ = core.fail(errors.New("first"))
	_ = core.fail(errors.New("second"))
	assert.Equal(t, 1, cleaned, "cleanup must run exactly once")
	require.EqualError(t, core.waitDone(), "first")
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "fail should remove the partial file")
}

func TestCoreVerifyAndCloseHappy(t *testing.T) {
	core, path := newCoreForTest(t)
	payload := []byte("abcd")
	sum := sha256.Sum256(payload)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: uint64(len(payload)),
		SHA256:   sum,
	}
	require.NoError(t, core.handleMetadata(meta))
	require.NoError(t, core.writeChunk(0, payload))
	core.verifyAndClose()
	require.NoError(t, core.waitDone())
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestCoreVerifyAndCloseRacedByFail(t *testing.T) {
	core, _ := newCoreForTest(t)
	meta := protocol.Metadata{
		Version:  protocol.ProtocolVersion,
		Codec:    protocol.CodecNone,
		FileSize: 1,
	}
	require.NoError(t, core.handleMetadata(meta))
	_ = core.fail(errors.New("external"))
	core.verifyAndClose()
	require.EqualError(t, core.waitDone(), "external")
}
