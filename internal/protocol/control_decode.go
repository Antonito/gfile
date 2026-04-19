package protocol

import (
	"encoding/binary"
	"fmt"
)

// DecodeAddPeerOffer parses an ADD_PEER_OFFER body.
func DecodeAddPeerOffer(body []byte) (peerID uint8, sdp string, err error) {
	return decodePeerSDP(body)
}

// DecodeAddPeerAnswer parses an ADD_PEER_ANSWER body.
func DecodeAddPeerAnswer(body []byte) (peerID uint8, sdp string, err error) {
	return decodePeerSDP(body)
}

// decodePeerSDP parses an ADD_PEER_OFFER / ADD_PEER_ANSWER body.
//
// Body layout: [peer_id:1][sdp_len:4][sdp:N].
func decodePeerSDP(body []byte) (peerID uint8, sdp string, err error) {
	// strip the type byte
	const bodyHeaderLen = peerSDPHeaderLen - 1

	if len(body) < bodyHeaderLen {
		return 0, "", fmt.Errorf("%w: peer-sdp body %d bytes (want >= %d)",
			ErrTruncatedFrame, len(body), bodyHeaderLen)
	}

	peerID = body[0]
	sdpLen := binary.BigEndian.Uint32(body[1:bodyHeaderLen])

	if uint32(len(body)-bodyHeaderLen) < sdpLen {
		return 0, "", fmt.Errorf("%w: sdp length %d > remaining %d",
			ErrTruncatedFrame, sdpLen, len(body)-bodyHeaderLen)
	}

	return peerID, string(body[bodyHeaderLen : bodyHeaderLen+sdpLen]), nil
}
