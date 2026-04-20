package stream

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadStream(t *testing.T) {
	asrt := assert.New(t)
	stream := &bytes.Buffer{}

	_, err := stream.WriteString("Hello\n")
	require.NoError(t, err)

	str, err := MustReadStream(stream)
	asrt.Equal("Hello", str)
	require.NoError(t, err)
}
