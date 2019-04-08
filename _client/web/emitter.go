// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"reflect"
	"syscall/js"
	"unsafe"

	"github.com/antonito/gfile/pkg/session/common"
	"github.com/antonito/gfile/pkg/session/sender"
	"github.com/antonito/gfile/pkg/utils"
)

func updateFilePlaceholder(_ js.Value, _ []js.Value) interface{} {
	// Check if file was selected
	fileSelector := getElementByID("send-file-input")
	fileList := fileSelector.Get("files")
	fileSelectorLabels := fileSelector.Get("labels")
	fileListLen := fileList.Length()

	if fileListLen == 0 {
		fileSelectorLabels.Index(0).Set("textContent", "Choose file")
	} else if fileListLen == 1 {
		filename := fileList.Index(0).Get("name").String()
		fileSelectorLabels.Index(0).Set("textContent", filename)
	} else {
		// Should never reach this part, but
		// TODO: Pop-up error
		fmt.Printf("Error, too many files")
	}
	return js.Undefined()
}

func sendFile(fileContent js.Value) {
	// Manually allocate a memory zone, and get its raw pointer
	// make it point to the JS internal's memory array
	fileContentLength := fileContent.Length()
	fileBuffer := make([]byte, fileContentLength)
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&fileBuffer))
	ptr := uintptr(unsafe.Pointer(hdr.Data))
	js.Global().Get("window").Call("setMemory", fileContent, ptr)

	reader := bytes.NewReader(fileBuffer)

	// Retrieve remote SDP
	sdpInputBox := getElementByID("send-sdpInput")
	sdpInputBoxText := sdpInputBox.Get("value").String()

	// Access session
	sess := globalSess.(*sender.Session)

	sdpInput.WriteString(sdpInputBoxText + "\n")

	sess.SetStream(reader)

	// Notify client, in progress
	if err := sess.Start(); err != nil {
		// Notifiy client of error
		// TODO: Handle error
	}
	// Notifiy client of end
	processDone <- struct{}{}
}

func onSendFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	go func() {
		// Check if file was selected
		fileSelector := getElementByID("send-file-input")
		fileList := fileSelector.Get("files")
		if fileList.Length() == 0 {
			// TODO: Pop-up error
			fmt.Println("No file selected")
			return
		}
		fileToSend := fileList.Index(0)

		js.Global().Call("readFileHelper", fileToSend, js.FuncOf(func(_ js.Value, res []js.Value) interface{} {
			if len(res) >= 1 {
				go sendFile(res[0])
			}
			return js.Undefined()
		}))
	}()
	return js.Undefined()
}

func onMenuSendFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	go func() {
		// Update UI
		getElementByID("menu-container").Set("hidden", true)
		getElementByID("menu-send-container").Set("hidden", false)

		sdpOutputBox := getElementByID("send-sdpOutput")
		sdpOutputBox.Set("textContent", "Generating SDP...")

		sess := sender.NewWith(sender.Config{
			Stream: nil,
			Configuration: common.Configuration{
				SDPProvider: sdpInput,
				SDPOutput:   sdpOutput,
				OnCompletion: func() {
					// TODO: Notify user ?
				},
			},
		})
		globalSess = sess
		sess.Initialize()
		sdp, err := utils.MustReadStream(sdpOutput)
		if err != nil {
			// TODO: Notify error
		}

		// Show SDP to the user
		sdpOutputBox.Set("textContent", sdp)
	}()

	return js.Undefined()
}

func setupEmitter() {
	setCallback("onMenuSendAFileClick", onMenuSendFileClickHandler)
	setCallback("onSendFileButtonClick", onSendFileButtonClick)
	setCallback("updateFilePlaceholder", updateFilePlaceholder)
}
