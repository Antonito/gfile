package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// ChunkSize is the DATA payload size for a full chunk; the final chunk
	// may be smaller. 256 KB sits well under pion's ~1 GB SCTP max message
	// size.
	ChunkSize = 256 * 1024

	// PrimaryLabel is the DataChannel label used for the single channel.
	PrimaryLabel = "primary"
	// ControlLabel is the DataChannel label on the control PeerConnection
	// in multi-PC mode. Data PCs use LabelForDataPeer(i).
	ControlLabel = "control"

	// METADATA body: [version:1][codec:1][file_size:8][sha256:32].
	metadataBodyLen = 1 + 1 + 8 + 32
	// DATA header: [type:1][offset:8].
	dataHeaderLen = 1 + 8
	// ADD_PEER_OFFER/ANSWER header: [type:1][peer_id:1][sdp_len:4].
	// Body-level checks subtract 1 since decoders see the body with the
	// type byte already stripped.
	peerSDPHeaderLen = 1 + 1 + 4

	// MaxDataReadBufSize sizes detached reads on data-bearing DataChannels.
	// Must exceed dataHeaderLen + ChunkSize plus zstd's non-compressible
	// expansion margin.
	MaxDataReadBufSize = 288 * 1024
	// MaxControlReadBufSize sizes detached reads on control DataChannels.
	// The largest frame is ADD_PEER_OFFER/ANSWER carrying a base64-encoded
	// SDP.
	MaxControlReadBufSize = 16 * 1024
)

var (
	// ErrTruncatedFrame means the frame is shorter than its type requires.
	ErrTruncatedFrame = errors.New("truncated frame")
	// ErrUnknownFrameType means the type byte is not one of the known values.
	ErrUnknownFrameType = errors.New("unknown frame type")
)

// Metadata is the handshake frame the sender emits before any DATA. All
// multi-byte fields are big-endian on the wire.
type Metadata struct {
	Version  Version
	Codec    Codec
	FileSize uint64
	SHA256   [32]byte
}

// EncodeMetadata returns a complete METADATA frame ready for DataChannel.Send.
//
// Layout: [type:1][version:1][codec:1][file_size:8][sha256:32].
func EncodeMetadata(meta Metadata) []byte {
	buf := make([]byte, 1+metadataBodyLen)
	buf[0] = byte(FrameTypeMetadata)
	buf[1] = byte(meta.Version)
	buf[2] = byte(meta.Codec)

	binary.BigEndian.PutUint64(buf[3:11], meta.FileSize)
	copy(buf[11:43], meta.SHA256[:])

	return buf
}

// DecodeMetadata parses the body (everything after the type byte).
//
// Body layout: [version:1][codec:1][file_size:8][sha256:32].
func DecodeMetadata(body []byte) (Metadata, error) {
	if len(body) != metadataBodyLen {
		return Metadata{}, fmt.Errorf("%w: metadata body %d bytes (want %d)",
			ErrTruncatedFrame, len(body), metadataBodyLen)
	}

	var meta Metadata
	meta.Version = Version(body[0])
	meta.Codec = Codec(body[1])
	meta.FileSize = binary.BigEndian.Uint64(body[2:10])
	copy(meta.SHA256[:], body[10:42])

	return meta, nil
}

// Data is a decoded DATA frame.
type Data struct {
	Offset  uint64
	Payload []byte
}

// EncodeData returns a complete DATA frame. payload may be empty (zero-length
// final chunk) but is typically up to ChunkSize bytes.
func EncodeData(offset uint64, payload []byte) []byte {
	return AppendData(nil, offset, payload)
}

// AppendData appends a DATA frame to dst and returns the extended slice.
// Layout: [type:1][offset:8][payload:N]. Hot-path callers pass `buf[:0]`
// to reuse the backing array — pion copies the argument into its own send
// queue, so aliasing is safe.
func AppendData(dst []byte, offset uint64, payload []byte) []byte {
	total := dataHeaderLen + len(payload)

	if cap(dst) < total {
		dst = make([]byte, total)
	} else {
		dst = dst[:total]
	}

	dst[0] = byte(FrameTypeData)
	binary.BigEndian.PutUint64(dst[1:9], offset)
	copy(dst[9:], payload)

	return dst
}

// DecodeData parses the body (everything after the type byte).
//
// Body layout: [offset:8][payload:N]. The returned Payload aliases the
// input slice — callers that need to keep the bytes must copy them before
// the owning OnMessage callback returns.
func DecodeData(body []byte) (Data, error) {
	const bodyHeaderLen = dataHeaderLen - 1 // strip the type byte

	if len(body) < bodyHeaderLen {
		return Data{}, fmt.Errorf("%w: data body %d bytes (want >= %d)",
			ErrTruncatedFrame, len(body), bodyHeaderLen)
	}

	return Data{
		Offset:  binary.BigEndian.Uint64(body[:bodyHeaderLen]),
		Payload: body[bodyHeaderLen:],
	}, nil
}

// EncodeEOF returns a one-byte EOF frame.
func EncodeEOF() []byte {
	return []byte{byte(FrameTypeEOF)}
}

// EncodeAbort returns an ABORT frame with a UTF-8 reason string.
func EncodeAbort(reason string) []byte {
	buf := make([]byte, 1+len(reason))

	buf[0] = byte(FrameTypeAbort)
	copy(buf[1:], reason)

	return buf
}

// DecodeAbort returns the reason string from an ABORT body.
func DecodeAbort(body []byte) string {
	return string(body)
}

// LabelForDataPeer returns the DataChannel label for
// the peerID'th data PC in multi-PC mode (0..peers-1).
func LabelForDataPeer(peerID int) string {
	return fmt.Sprintf("data-%d", peerID)
}

func peekType(msg []byte) (FrameType, []byte, error) {
	if len(msg) < 1 {
		return 0, nil, ErrTruncatedFrame
	}

	frameType := FrameType(msg[0])

	switch frameType {
	case FrameTypeMetadata, FrameTypeData, FrameTypeEOF, FrameTypeAbort,
		FrameTypeAddPeerOffer, FrameTypeAddPeerAnswer, FrameTypeTransferComplete:
		return frameType, msg[1:], nil
	default:
		return 0, nil, fmt.Errorf("%w: 0x%02x", ErrUnknownFrameType, uint8(frameType))
	}
}
