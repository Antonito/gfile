// +build js,wasm

package main

import "syscall/js"

func getElementByID(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}

type jsCallback func(_ js.Value, _ []js.Value) interface{}

func setCallback(id string, cb jsCallback) {
	js.Global().Set(id, js.FuncOf(cb))
}
