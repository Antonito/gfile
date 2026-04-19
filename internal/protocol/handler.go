package protocol

import (
	"errors"
	"fmt"
)

// FrameHandler is the visitor interface Dispatch calls into.
// Each method corresponds to exactly one FrameType.
type FrameHandler interface {
	OnMetadata(Metadata) error
	OnData(Data) error
	OnEOF() error
	OnAbort(reason string) error
	OnAddPeerOffer(peerID uint8, sdp string) error
	OnAddPeerAnswer(peerID uint8, sdp string) error
	OnTransferComplete() error
}

// ErrUnexpectedFrame is the sentinel returned by UnexpectedFrameHandler
// methods.
var ErrUnexpectedFrame = errors.New("unexpected frame")

// UnexpectedFrameHandler provides a default implementation of every
// FrameHandler method that returns ErrUnexpectedFrame wrapped with the
// frame type byte
type UnexpectedFrameHandler struct{}

func unexpectedFrame(frameType FrameType) error {
	return fmt.Errorf("%w: 0x%02x", ErrUnexpectedFrame, uint8(frameType))
}

// OnMetadata rejects a METADATA frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnMetadata(Metadata) error {
	return unexpectedFrame(FrameTypeMetadata)
}

// OnData rejects a DATA frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnData(Data) error {
	return unexpectedFrame(FrameTypeData)
}

// OnEOF rejects an EOF frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnEOF() error {
	return unexpectedFrame(FrameTypeEOF)
}

// OnAbort rejects an ABORT frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnAbort(string) error {
	return unexpectedFrame(FrameTypeAbort)
}

// OnAddPeerOffer rejects an ADD_PEER_OFFER frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnAddPeerOffer(uint8, string) error {
	return unexpectedFrame(FrameTypeAddPeerOffer)
}

// OnAddPeerAnswer rejects an ADD_PEER_ANSWER frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnAddPeerAnswer(uint8, string) error {
	return unexpectedFrame(FrameTypeAddPeerAnswer)
}

// OnTransferComplete rejects a TRANSFER_COMPLETE frame with ErrUnexpectedFrame.
func (UnexpectedFrameHandler) OnTransferComplete() error {
	return unexpectedFrame(FrameTypeTransferComplete)
}
