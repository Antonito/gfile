// +build js,wasm
package main

import (
	"fmt"
	"syscall/js"
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
	fileBuffer := fileContent.Get("buffer").Call("slice")
	fmt.Printf("%v\n", []byte(fileBuffer))

	// reader := bytes.NewReader([]byte(fileContent))

	/*
		// Retrieve remote SDP
		sdpInputBox := getElementByID("send-sdpInput")
		sdpInputBoxText := sdpInputBox.Get("textContent").String()

		// Access session
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
	*/
}

func onSendFileButtonClick(_ js.Value, _ []js.Value) interface{} {
	// Check if file was selected
	fileSelector := getElementByID("send-file-input")
	fileList := fileSelector.Get("files")
	if fileList.Length() == 0 {
		// TODO: Pop-up error
		fmt.Printf("No file selected\n")
		return js.Undefined()
	}
	fileToSend := fileList.Index(0)

	js.Global().Call("readFileHelper", fileToSend, js.FuncOf(func(_ js.Value, res []js.Value) interface{} {
		if len(res) >= 1 {
			sendFile(res[0])
		}
		return js.Undefined()
	}))
	return js.Undefined()
}

func onMenuSendFileClickHandler(_ js.Value, _ []js.Value) interface{} {
	// Update UI
	getElementByID("menu-container").Set("hidden", true)
	getElementByID("menu-send-container").Set("hidden", false)

	sdpOutputBox := getElementByID("send-sdpOutput")
	sdpOutputBox.Set("textContent", "Generating SDP...")

	/*
		// Start session
		sdpOutput := &bytes.Buffer{}
		sdpInput := &bytes.Buffer{}

		sess := sender.NewWith(sender.Config{
			Stream: nil,
			Configuration: common.Configuration{
				SDPProvider: sdpInput,
				SDPOutput:   sdOutput,
				OnCompletion: func() {
					// TODO: Notify user ?
				},
			},
		})
		globalSess = sess
		sess.Initialize()
		sdp, err := utils.MustReadStream(sdpOutput)

		// Show SDP to the user
		sdpOutputBox.Set("textContent", sdp)
	*/

	return js.Undefined()
}

func setupEmitter() {
	setCallback("onMenuSendAFileClick", onMenuSendFileClickHandler)
	setCallback("onSendFileButtonClick", onSendFileButtonClick)
	setCallback("updateFilePlaceholder", updateFilePlaceholder)
}
