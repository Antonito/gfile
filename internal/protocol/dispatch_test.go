package protocol

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingHandler struct {
	UnexpectedFrameHandler
	called string
}

func (r *recordingHandler) OnMetadata(Metadata) error {
	r.called = "Metadata"
	return nil
}

func (r *recordingHandler) OnData(Data) error {
	r.called = "Data"
	return nil
}

func (r *recordingHandler) OnEOF() error {
	r.called = "EOF"
	return nil
}

func (r *recordingHandler) OnAbort(string) error {
	r.called = "Abort"
	return nil
}

func (r *recordingHandler) OnAddPeerOffer(uint8, string) error {
	r.called = "AddPeerOffer"
	return nil
}

func (r *recordingHandler) OnAddPeerAnswer(uint8, string) error {
	r.called = "AddPeerAnswer"
	return nil
}

func (r *recordingHandler) OnTransferComplete() error {
	r.called = "TransferComplete"
	return nil
}

func TestDispatchRouting(t *testing.T) {
	cases := []struct {
		name  string
		frame []byte
		want  string
	}{
		{"Metadata", EncodeMetadata(Metadata{Version: ProtocolVersion}), "Metadata"},
		{"Data", EncodeData(100, []byte("x")), "Data"},
		{"EOF", EncodeEOF(), "EOF"},
		{"Abort", EncodeAbort("reason"), "Abort"},
		{"AddPeerOffer", EncodeAddPeerOffer(1, "sdp"), "AddPeerOffer"},
		{"AddPeerAnswer", EncodeAddPeerAnswer(1, "sdp"), "AddPeerAnswer"},
		{"TransferComplete", EncodeTransferComplete(), "TransferComplete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &recordingHandler{}
			require.NoError(t, Dispatch(tc.frame, handler), "Dispatch")
			assert.Equal(t, tc.want, handler.called)
		})
	}
}

func TestDispatchTruncated(t *testing.T) {
	err := Dispatch(nil, &recordingHandler{})
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestDispatchUnknownType(t *testing.T) {
	err := Dispatch([]byte{0xFF}, &recordingHandler{})
	assert.ErrorIs(t, err, ErrUnknownFrameType)
}

func TestUnexpectedFrameHandler(t *testing.T) {
	var handler UnexpectedFrameHandler
	checks := []struct {
		name string
		fn   func() error
	}{
		{"Metadata", func() error { return handler.OnMetadata(Metadata{}) }},
		{"Data", func() error { return handler.OnData(Data{}) }},
		{"EOF", handler.OnEOF},
		{"Abort", func() error { return handler.OnAbort("") }},
		{"AddPeerOffer", func() error { return handler.OnAddPeerOffer(0, "") }},
		{"AddPeerAnswer", func() error { return handler.OnAddPeerAnswer(0, "") }},
		{"TransferComplete", handler.OnTransferComplete},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, tc.fn(), ErrUnexpectedFrame)
		})
	}
}

func TestDispatchPropagatesHandlerError(t *testing.T) {
	sentinel := errors.New("boom")
	handler := &errHandler{err: sentinel}
	err := Dispatch(EncodeEOF(), handler)
	assert.ErrorIs(t, err, sentinel)
}

type errHandler struct {
	UnexpectedFrameHandler
	err error
}

func (e *errHandler) OnEOF() error { return e.err }
