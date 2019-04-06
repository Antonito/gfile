// +build js,wasm
package main

// var globalSess session.Session

func main() {
	setupEmitter()
	setupReceiver()

	select {}
}
