// +build js,wasm
package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/antonito/gfile/pkg/session/common"
	"github.com/antonito/gfile/pkg/session/receiver"
	"github.com/antonito/gfile/pkg/utils"
)

func onMenuReceiveFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	go func() {
		getElementByID("menu-container").Set("hidden", true)
		getElementByID("menu-receive-container").Set("hidden", false)

		sess := globalSess.(*receiver.Session)
		writer := bytes.Buffer{}

		sess.SetStream(writer)
		if err := sess.Start(); err != nil {
			// TOOD: Notify error
		}

		processDone <- struct{}{}
	}()
	return js.Undefined()
}

func onReceiveFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	go func() {
		sdpInputBox := getElementByID("receive-sdpInput")
		sdpInputBoxText := sdpInputBox.Get("textContent").String()

		sdpInput.WriteString(sdpInputBoxText + "\n")

		sess := receiver.NewWith(receiver.Config{
			Configuration: common.Configuration{
				SDPProvider: sdpInput,
				SDPOutput:   sdpOutput,
				OnCompletion: func() {
				},
			},
		})

		globalSess = sess

		go sess.Initialize()

		sdp, err := utils.MustReadStream(sdpOutput)
		if err != nil {
			fmt.Printf("Got error -> %s\n", err)
			// TODO: Notify error
		}

		fmt.Println("SDP is %s", sdp)
		sdpOutputBox := getElementByID("receive-sdpOutput")
		sdpOutputBox.Set("textContent", sdp)
	}()

	return js.Undefined()
}

func setupReceiver() {
	setCallback("onMenuReceiveFileClick", onMenuReceiveFileClickHandler)
	setCallback("onReceiveFileButtonClick", onReceiveFileButtonClick)
}
