// +build js,wasm
package main

import (
	"syscall/js"
)

func onMenuReceiveFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	getElementByID("menu-container").Set("hidden", true)
	getElementByID("menu-receive-container").Set("hidden", false)
	return js.Undefined()
}

func onReceiveFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	/*
		sdpInputBox := getElementByID("receive-sdpInput")
		sdpInputBoxText := sdpInputBox.Get("textContent").String()

			sdpOutput := &bytes.Buffer{}
			sdpInput := &bytes.Buffer{}

			sdpInput.WriteString(sdpInputBoxText + "\n")

			sess := receiver.NewWith(receiver.Config{
				Configuration: common.Configuration{
					SDPProvider: sdpInput,
					SDPOutput:   sdOutput,
					OnCompletion: func() {
					},
				},
			})

			globalSess = sess
			sess.Initialize()
			sdp, err := utils.MustReadStream(sdpOutput)

			sdpOutputBox.Set("textContent", sdp)
	*/

	return js.Undefined()
}

func setupReceiver() {
	setCallback("onMenuReceiveFileClick", onMenuReceiveFileClickHandler)
	setCallback("onReceiveFileButtonClick", onReceiveFileButtonClick)
}
