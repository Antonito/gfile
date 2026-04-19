package protocol

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddPeerOfferRoundTrip(t *testing.T) {
	asrt := assert.New(t)
	sdp := "v=0\r\no=- 123 2 IN IP4 127.0.0.1\r\n" + strings.Repeat("a=ice\r\n", 50)
	frame := EncodeAddPeerOffer(3, sdp)
	require.Equal(t, FrameTypeAddPeerOffer, FrameType(frame[0]), "type byte")
	id, got, err := DecodeAddPeerOffer(frame[1:])
	require.NoError(t, err, "decode")
	asrt.Equal(uint8(3), id)
	asrt.Equal(sdp, got)
}

func TestAddPeerAnswerRoundTrip(t *testing.T) {
	asrt := assert.New(t)
	sdp := "v=0\r\no=- 456 2 IN IP4 127.0.0.1\r\n"
	frame := EncodeAddPeerAnswer(7, sdp)
	require.Equal(t, FrameTypeAddPeerAnswer, FrameType(frame[0]), "type byte")
	id, got, err := DecodeAddPeerAnswer(frame[1:])
	require.NoError(t, err, "decode")
	asrt.Equal(uint8(7), id)
	asrt.Equal(sdp, got)
}

func TestAddPeerOfferEmptyBody(t *testing.T) {
	_, _, err := DecodeAddPeerOffer(nil)
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestAddPeerOfferTruncatedLength(t *testing.T) {
	// peer_id + length prefix but 0 byte body when length says 10
	body := []byte{0x01, 0x00, 0x00, 0x00, 0x0A}
	_, _, err := DecodeAddPeerOffer(body)
	assert.ErrorIs(t, err, ErrTruncatedFrame)
}

func TestTransferCompleteSingleByte(t *testing.T) {
	frame := EncodeTransferComplete()
	assert.Equal(t, []byte{byte(FrameTypeTransferComplete)}, frame)
}
