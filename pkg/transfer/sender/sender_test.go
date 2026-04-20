package sender

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	asrt := assert.New(t)
	input := bytes.NewReader([]byte("test content"))

	sess := New(input)

	asrt.NotNil(sess)
	asrt.Equal(input, sess.stream)
}
