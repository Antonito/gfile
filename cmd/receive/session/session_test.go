package session

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	sender "github.com/antonito/gfile/cmd/send/session"
	"github.com/antonito/gfile/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Helper
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}
func (b *Buffer) ReadString(delim byte) (line string, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.ReadString(delim)
}
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}
func (b *Buffer) WriteString(s string) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.WriteString(s)
}
func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

// Tests

func Test_CreateSession(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}

	ses := NewSession(stream)
	assert.NotNil(ses)
}

func Test_TransferSmallMessage(t *testing.T) {
	assert := assert.New(t)

	// Create client receiver
	clientStream := &Buffer{}
	clientSDPProvider := &Buffer{}
	clientSDPOutput := &Buffer{}
	clientConfig := Config{
		Stream:      clientStream,
		SDPProvider: clientSDPProvider,
		SDPOutput:   clientSDPOutput,
	}
	clientSession := NewSessionWith(clientConfig)
	assert.NotNil(clientSession)

	// Create sender
	senderStream := &Buffer{}
	senderSDPProvider := &Buffer{}
	senderSDPOutput := &Buffer{}
	n, err := senderStream.WriteString("Hello World!\n")
	assert.Nil(err)
	assert.Equal(13, n) // Len "Hello World\n"
	senderConfig := sender.Config{
		Stream:      senderStream,
		SDPProvider: senderSDPProvider,
		SDPOutput:   senderSDPOutput,
	}
	senderSession := sender.NewSessionWith(senderConfig)
	assert.NotNil(senderSession)

	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		err := senderSession.Connect()
		assert.Nil(err)
	}()

	// Get SDP from sender and send it to the client
	sdp, err := utils.MustReadStream(senderSDPOutput)
	assert.Nil(err)
	sdp += "\n"
	n, err = clientSDPProvider.WriteString(sdp)
	assert.Nil(err)
	assert.Equal(len(sdp), n)

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		err := clientSession.Connect()
		assert.Nil(err)
	}()

	// Get SDP from client and send it to the sender
	sdp, err = utils.MustReadStream(clientSDPOutput)
	assert.Nil(err)
	n, err = senderSDPProvider.WriteString(sdp)
	assert.Nil(err)
	assert.Equal(len(sdp), n)

	fmt.Println("Waiting for everyone to be done...")
	<-senderDone
	<-clientDone

	msg, err := clientStream.ReadString('\n')
	assert.Nil(err)
	assert.Equal("Hello World!\n", msg)
}
