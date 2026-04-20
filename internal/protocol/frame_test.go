package protocol

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataRoundTrip(t *testing.T) {
	asrt := assert.New(t)
	in := Metadata{
		Version:  ProtocolVersion,
		Codec:    CodecNone,
		FileSize: 1 << 40,
		SHA256:   sha256.Sum256([]byte("hello")),
	}
	frame := EncodeMetadata(in)
	require.Len(t, frame, 1+metadataBodyLen, "frame len")
	require.Equal(t, FrameTypeMetadata, FrameType(frame[0]), "type byte")
	got, err := DecodeMetadata(frame[1:])
	require.NoError(t, err, "DecodeMetadata")
	asrt.Equal(in, got)
}

func TestMetadataTruncated(t *testing.T) {
	_, err := DecodeMetadata(make([]byte, 10))
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestDataRoundTrip(t *testing.T) {
	asrt := assert.New(t)
	payload := bytes.Repeat([]byte{0xab}, 1024)
	frame := EncodeData(0xdeadbeef, payload)
	require.Equal(t, FrameTypeData, FrameType(frame[0]), "type byte")
	got, err := DecodeData(frame[1:])
	require.NoError(t, err, "DecodeData")
	asrt.Equal(uint64(0xdeadbeef), got.Offset)
	asrt.Equal(payload, got.Payload)
}

func TestDataEmptyPayload(t *testing.T) {
	asrt := assert.New(t)
	frame := EncodeData(42, nil)
	got, err := DecodeData(frame[1:])
	require.NoError(t, err, "DecodeData")
	asrt.Equal(uint64(42), got.Offset)
	asrt.Empty(got.Payload)
}

func TestDataTruncated(t *testing.T) {
	_, err := DecodeData(make([]byte, 5))
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestEOF(t *testing.T) {
	frame := EncodeEOF()
	require.Len(t, frame, 1, "eof frame len")
	assert.Equal(t, FrameTypeEOF, FrameType(frame[0]), "type byte")
}

func TestAbortRoundTrip(t *testing.T) {
	frame := EncodeAbort("integrity check failed")
	require.Equal(t, FrameTypeAbort, FrameType(frame[0]), "type byte")
	assert.Equal(t, "integrity check failed", DecodeAbort(frame[1:]))
}

func TestAbortEmptyReason(t *testing.T) {
	frame := EncodeAbort("")
	require.Len(t, frame, 1, "frame len")
	assert.Empty(t, DecodeAbort(frame[1:]))
}

func TestPeekType(t *testing.T) {
	cases := []struct {
		name string
		msg  []byte
		want FrameType
	}{
		{"metadata", EncodeMetadata(Metadata{Version: ProtocolVersion}), FrameTypeMetadata},
		{"data", EncodeData(0, nil), FrameTypeData},
		{"eof", EncodeEOF(), FrameTypeEOF},
		{"abort", EncodeAbort("x"), FrameTypeAbort},
		{"add_peer_offer", []byte{byte(FrameTypeAddPeerOffer), 0x00}, FrameTypeAddPeerOffer},
		{"add_peer_answer", []byte{byte(FrameTypeAddPeerAnswer), 0x00}, FrameTypeAddPeerAnswer},
		{"transfer_complete", []byte{byte(FrameTypeTransferComplete)}, FrameTypeTransferComplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			asrt := assert.New(t)
			got, body, err := peekType(tc.msg)
			require.NoError(t, err, "peekType")
			asrt.Equal(tc.want, got)
			asrt.Len(body, len(tc.msg)-1)
		})
	}
}

func TestPeekTypeUnknown(t *testing.T) {
	_, _, err := peekType([]byte{0x99})
	assert.ErrorIs(t, err, ErrUnknownFrameType)
}

func TestPeekTypeEmpty(t *testing.T) {
	_, _, err := peekType(nil)
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestDataLargeOffset(t *testing.T) {
	asrt := assert.New(t)
	payload := []byte("tail")
	const offset = uint64(1<<63 + 12345)
	frame := EncodeData(offset, payload)
	got, err := DecodeData(frame[1:])
	require.NoError(t, err, "DecodeData")
	asrt.Equal(offset, got.Offset)
	asrt.Equal(payload, got.Payload)
}

func TestMetadataRejectsOverlongBody(t *testing.T) {
	// Body must be exactly metadataBodyLen; a too-long body is also invalid.
	_, err := DecodeMetadata(make([]byte, metadataBodyLen+1))
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestPeekTypeFeedsDecoders(t *testing.T) {
	asrt := assert.New(t)
	payload := []byte("abc")
	full := EncodeData(7, payload)
	ft, body, err := peekType(full)
	require.NoError(t, err, "peekType")
	require.Equal(t, FrameTypeData, ft, "type")
	data, err := DecodeData(body)
	require.NoError(t, err, "DecodeData on peekType body")
	asrt.Equal(uint64(7), data.Offset)
	asrt.Equal(payload, data.Payload)
}
