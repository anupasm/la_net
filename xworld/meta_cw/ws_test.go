package main

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

func Test_wsHandler(t *testing.T) {

	var done chan bool
	u := "ws://127.0.0.1:8081/secret"

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	var key TxKey
	var arr [32]byte
	copy(arr[:], "a01a7e08f652cf3a439ec78e3f7bff4d06253ac5db6520c4ad8173f5e3556212")
	key.Key = arr[:]
	key.TxId = "XXXXX"
	buf, err := json.Marshal(key)

	if err != nil {
		panic(err)
	}

	if err := ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		t.Fatalf("%v", err)
	}
	<-done

}
