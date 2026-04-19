package protocol

import (
	"fmt"
)

// Dispatch parses one DataChannel message, decodes the body,
// and calls the matching method on h.
func Dispatch(msg []byte, h FrameHandler) error {
	t, body, err := peekType(msg)
	if err != nil {
		return err
	}

	switch t {
	case FrameTypeMetadata:
		m, err := DecodeMetadata(body)
		if err != nil {
			return err
		}
		return h.OnMetadata(m)

	case FrameTypeData:
		d, err := DecodeData(body)
		if err != nil {
			return err
		}
		return h.OnData(d)

	case FrameTypeEOF:
		return h.OnEOF()

	case FrameTypeAbort:
		return h.OnAbort(DecodeAbort(body))

	case FrameTypeAddPeerOffer:
		id, sdp, err := DecodeAddPeerOffer(body)
		if err != nil {
			return err
		}
		return h.OnAddPeerOffer(id, sdp)

	case FrameTypeAddPeerAnswer:
		id, sdp, err := DecodeAddPeerAnswer(body)
		if err != nil {
			return err
		}
		return h.OnAddPeerAnswer(id, sdp)

	case FrameTypeTransferComplete:
		return h.OnTransferComplete()

	// Unreachable
	default:
		return fmt.Errorf("%w: 0x%02x", ErrUnknownFrameType, uint8(t))
	}
}
