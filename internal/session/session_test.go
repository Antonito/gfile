package session

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	assert := assert.New(t)
	input := bufio.NewReader(&bytes.Buffer{})
	output := bufio.NewWriter(&bytes.Buffer{})

	sess := New(nil, nil, "")
	assert.Equal(os.Stdin, sess.sdpInput)
	assert.Equal(os.Stdout, sess.sdpOutput)
	assert.Equal(1, len(sess.stunServers))
	assert.Equal("stun:stun.l.google.com:19302", sess.stunServers[0])

	sess = New(input, output, "test:123")
	assert.Equal(input, sess.sdpInput)
	assert.Equal(output, sess.sdpOutput)
	assert.Equal(1, len(sess.stunServers))
	assert.Equal(true, strings.HasPrefix(sess.stunServers[0], "stun:"))
	arr := strings.Split(sess.stunServers[0], ":")
	assert.Equal(3, len(arr))
	assert.Equal("stun", arr[0])
	assert.Equal("test", arr[1])
	assert.Equal("123", arr[2])
}
