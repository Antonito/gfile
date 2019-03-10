package session

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateSession(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}

	ses := NewSession(stream)
	assert.NotNil(ses)
}
