package transfer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDataEncoderRoundTrip(t *testing.T) {
	enc, err := NewDataEncoder(3)
	require.NoError(t, err, "NewDataEncoder")
	defer enc.Close()

	payload := bytes.Repeat([]byte("hello-compress"), 512)
	compressed := enc.EncodeAll(payload, nil)
	assert.Less(t, len(compressed), len(payload),
		"repetitive input should compress")

	dec, err := NewDataDecoder()
	require.NoError(t, err, "NewDataDecoder")
	defer dec.Close()

	got, err := dec.DecodeAll(compressed, nil)
	require.NoError(t, err, "DecodeAll")
	assert.Equal(t, payload, got)
}
