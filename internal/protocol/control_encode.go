package protocol

import (
	"encoding/binary"
)

// EncodeAddPeerOffer returns a complete ADD_PEER_OFFER frame
func EncodeAddPeerOffer(peerID uint8, sdp string) []byte {
	return encodePeerSDP(FrameTypeAddPeerOffer, peerID, sdp)
}

// EncodeAddPeerAnswer returns a complete ADD_PEER_ANSWER frame.
func EncodeAddPeerAnswer(peerID uint8, sdp string) []byte {
	return encodePeerSDP(FrameTypeAddPeerAnswer, peerID, sdp)
}

// encodePeerSDP builds an ADD_PEER_OFFER / ADD_PEER_ANSWER frame.
//
// Layout: [type:1][peer_id:1][sdp_len:4][sdp:N].
func encodePeerSDP(frameType FrameType, peerID uint8, sdp string) []byte {
	buf := make([]byte, peerSDPHeaderLen+len(sdp))
	buf[0] = byte(frameType)
	buf[1] = peerID

	binary.BigEndian.PutUint32(buf[2:6], uint32(len(sdp)))
	copy(buf[peerSDPHeaderLen:], sdp)

	return buf
}

// EncodeTransferComplete returns a one-byte TRANSFER_COMPLETE frame.
func EncodeTransferComplete() []byte {
	return []byte{byte(FrameTypeTransferComplete)}
}
