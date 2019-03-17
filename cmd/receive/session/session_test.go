package session

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	sender "github.com/antonito/gfile/cmd/send/session"
	"github.com/antonito/gfile/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func Test_CreateSession(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}

	ses := NewSession(stream)
	assert.NotNil(ses)
}

func Test_TransferSmallMessage(t *testing.T) {
	assert := assert.New(t)

	// Create client receiver
	clientStream := &bytes.Buffer{}
	clientSDPProvider := &bytes.Buffer{}
	clientSDPOutput := &bytes.Buffer{}
	clientConfig := Config{
		Stream:      clientStream,
		SDPProvider: clientSDPProvider,
		SDPOutput:   clientSDPOutput,
	}
	clientSession := NewSessionWith(clientConfig)
	assert.NotNil(clientSession)

	// Create sender
	senderStream := &bytes.Buffer{}
	senderSDPProvider := &bytes.Buffer{}
	senderSDPOutput := &bytes.Buffer{}
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
	time.Sleep(1 * time.Second) // TODO: Improve reliability

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
	time.Sleep(1 * time.Second) // TODO: Improve reliability

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
