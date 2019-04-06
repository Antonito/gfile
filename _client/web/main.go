// +build js,wasm
package main

import "github.com/antonito/gfile/pkg/session"

var globalSess session.Session

func main() {
	setupEmitter()
	setupReceiver()

	select {}
}
