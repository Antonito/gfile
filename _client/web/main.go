// +build js,wasm
package main

import (
	"syscall/js"

	"github.com/antonito/gfile/internal/buffer"
	"github.com/antonito/gfile/internal/utils"
	"github.com/antonito/gfile/pkg/session/common"
	"github.com/antonito/gfile/pkg/session/receiver"
	"github.com/antonito/gfile/pkg/session/sender"
)

// var globalSess Session

func getElementByID(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}

func onMenuReceiveFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	getElementByID("menu-container").Set("hidden", true)
	getElementByID("menu-receive-container").Set("hidden", false)
	return js.Undefined()
}

func onMenuSendFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	getElementByID("menu-container").Set("hidden", true)
	getElementByID("menu-send-container").Set("hidden", false)

	sdpOutputBox := getElementByID("send-sdpOutput")
	sdpOutputBox.Set("textContent", "Generating SDP...")

	sdpOutput := &buffer.Buffer{}
	sdpInput := &buffer.Buffer{}

	sess := sender.NewWith(sender.Config{
		Stream: nil,
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

	return js.Undefined()
}

func onSendFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	sdpInputBox := getElementByID("send-sdpInput")
	sdpInputBoxText := sdpInputBox.Get("textContent").String()

	sess := globalSess.(*sender.Session)
	sess.SDPProvider().WriteString(sdpInputBoxText)

	// TODO: Start file stream
	sess.SetStream(nil)

	// Notify client, in progress
	if err := sess.Start(); err != nil {
		// Notifiy client of error
		// TODO: Handle error
	}
	// Notifiy client of end

	return js.Undefined()
}

func onReceiveFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	sdpInputBox := getElementByID("receive-sdpInput")
	sdpInputBoxText := sdpInputBox.Get("textContent").String()

	sdpOutput := &buffer.Buffer{}
	sdpInput := &buffer.Buffer{}

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

	return js.Undefined()
}

func main() {
	js.Global().Set("onMenuSendAFileClick", js.FuncOf(onMenuSendFileClickHandler))
	js.Global().Set("onSendFileButtonClick", js.FuncOf(onSendFileButtonClick))
	js.Global().Set("onMenuReceiveFileClick", js.FuncOf(onMenuReceiveFileClickHandler))
	js.Global().Set("onReceiveFileButtonClick", js.FuncOf(onReceiveFileButtonClick))

	select {}
}
