package utils

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_HTTPHandler(t *testing.T) {
	assert := assert.New(t)

	stream := &bytes.Buffer{}

	sdpChan := make(chan string)
	handler := handleSDP(sdpChan)

	done := make(chan struct{})
	msg := "Hello\n"
	_, err := stream.WriteString(msg)
	go func() {
		assert.Nil(err)
		res := <-sdpChan
		assert.Equal(msg, res)
		close(done)
	}()

	req := httptest.NewRequest("POST", "http://localhost:8080/sdp", stream)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(200, resp.StatusCode)
	assert.Equal("done", string(body))

	<-done
}

func Test_HTTP(t *testing.T) {
	assert := assert.New(t)

	sdpChan := HTTPSDPServer()

	msg := "Hello\n"
	go func() {
		stream := &bytes.Buffer{}
		_, err := stream.WriteString(msg)
		assert.Nil(err)

		client := &http.Client{}

		req, err := http.NewRequest("POST", "http://localhost:8080/sdp", stream)
		assert.Nil(err)
		resp, err := client.Do(req)
		assert.Nil(err)
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		assert.Equal(200, resp.StatusCode)
		assert.Equal("done", string(body))
	}()

	sdp := <-sdpChan
	assert.Equal(msg, sdp)
}
