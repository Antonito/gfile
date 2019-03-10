package session

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	sender "github.com/Antonito/gfile/cmd/send/session"
	"github.com/stretchr/testify/assert"
)

func Test_TransferSmallMessage(t *testing.T) {
	assert := assert.New(t)
	stream := &bytes.Buffer{}
	sdpAnswer := &bytes.Buffer{}
	sdpInput := &bytes.Buffer{}

	ses := NewSession(stream)
	assert.NotNil(ses)
	ses.sdpOutput = sdpAnswer
	ses.sdpInput = sdpInput

	// Start emitter
	sdpChan := make(chan *bytes.Buffer)
	go func() {
		sdpOffer := &bytes.Buffer{}
		senderSdpInput := &bytes.Buffer{}
		senderStream := &bytes.Buffer{}
		_, err := senderStream.WriteString("Hello World\n")
		assert.Nil(err)
		sendSes := sender.NewSession(senderStream)
		assert.NotNil(sendSes)
		sendSes.sdpOutput = sdpOffer
		sdpChan <- sdpOffer

		// Send SDP whenever possible
		go func() {
			sdp, err := sdpOffer.ReadString('\n')
			assert.Nil(err)

			client := &http.Client{}
			req, err := http.NewRequest("POST", "http://localhost:8080/sdp", strings.NewReader(sdp))
			assert.Nil(err)
			resp, err := client.Do(req)
			assert.Nil(err)
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			assert.Equal(200, resp.StatusCode)
			assert.Equal("done", string(body))
		}()

		err = sendSes.Connect()
		assert.Nil(err)
	}()

	sdp, err := (<-sdpChan).ReadString('\n')
	assert.Nil(err)
	_, err = sdpInput.WriteString(sdp)
	assert.Nil(err)

	err = ses.Connect()
	assert.Nil(err)

	assert.Equal("Done", stream.String())
}
