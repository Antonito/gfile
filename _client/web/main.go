// +build js,wasm

package main

import (
	"bytes"

	"github.com/antonito/gfile/pkg/session"
	log "github.com/sirupsen/logrus"
)

// TODO: Store in a struct
var globalSess session.Session
var sdpOutput *bytes.Buffer
var sdpInput *bytes.Buffer
var processDone chan struct{}

func main() {
	log.SetLevel(log.TraceLevel)
	processDone = make(chan struct{})

	sdpOutput = &bytes.Buffer{}
	sdpInput = &bytes.Buffer{}
	setupEmitter()
	setupReceiver()

	<-processDone
}
