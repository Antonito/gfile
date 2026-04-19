package protocol

// FrameType is the first byte of every frame. See PROTOCOL.md.
type FrameType uint8

const (
	// FrameTypeMetadata is the first frame, before any DATA
	FrameTypeMetadata FrameType = 0x01

	// FrameTypeData is the file payload with byte offset
	FrameTypeData FrameType = 0x02

	// FrameTypeEOF means the sender has no more chunks (single-PC only)
	FrameTypeEOF FrameType = 0x03

	// FrameTypeAbort is a clean-error signal with UTF-8 reason
	FrameTypeAbort FrameType = 0x04

	// FrameTypeAddPeerOffer is sender→receiver: invite a data PC by peer_id+SDP
	FrameTypeAddPeerOffer FrameType = 0x05

	// FrameTypeAddPeerAnswer is receiver→sender: answer an offer by peer_id+SDP
	FrameTypeAddPeerAnswer FrameType = 0x06

	// FrameTypeTransferComplete is sender→receiver: all data PCs flushed
	FrameTypeTransferComplete FrameType = 0x07
)
