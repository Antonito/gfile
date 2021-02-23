// +build js,wasm

package main

import (
	"bufio"
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
	}()
	return js.Undefined()
}

func onReceiveFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	go func() {
		sdpInputBox := getElementByID("receive-sdpInput")
		sdpInputBoxText := sdpInputBox.Get("value").String()

		sdpInput.WriteString(sdpInputBoxText + "\n")

		sess := receiver.NewWith(receiver.Config{
			Configuration: common.Configuration{
				SDPProvider: sdpInput,
				SDPOutput:   sdpOutput,
			},
		})

		globalSess = sess
		sess.Initialize()

		fmt.Printf("Reading SDP\n")
		sdp, err := utils.MustReadStream(sdpOutput)
		if err != nil {
			fmt.Printf("Got error -> %s\n", err)
			// TODO: Notify error
		}

		sdpOutputBox := getElementByID("receive-sdpOutput")
		sdpOutputBox.Set("textContent", sdp)

		buffer := &bytes.Buffer{}
		writerBuffer := bufio.NewWriter(buffer)
		sess.SetStream(writerBuffer)
		if err := sess.Start(); err != nil {
			fmt.Printf("Got an error: %v\n", err)
			// TOOD: Notify error
		} else {
			// Write file
			fmt.Println("Ready to write content")
			filename := "donwload.lol"
			bufferBytes := buffer.Bytes()
			array := js.TypedArrayOf(bufferBytes)
			js.Global().Get("window").Call("saveFile", filename, array)
			array.Release()
		}

		processDone <- struct{}{}
	}()

	return js.Undefined()
}

func setupReceiver() {
	setCallback("onMenuReceiveFileClick", onMenuReceiveFileClickHandler)
	setCallback("onReceiveFileButtonClick", onReceiveFileButtonClick)
}
